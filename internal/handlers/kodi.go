package handlers

import (
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jiotv-go/jiotv_go/v3/pkg/television"
	"github.com/jiotv-go/jiotv_go/v3/pkg/utils"
)

// /kodi/ serves the channels as an HTTP directory of .strm files, one folder
// per category, for Kodi's "Web server directory (HTTP)" source.
//
// Why: Kodi builds that play DRM through the browser (kodi-tizen-wasm) choose
// that player from the item's #KODIPROP lines. A channel of the IPTV Simple
// PVR client gets those properties only after the player is chosen, so its DRM
// channels end up in InputStream Adaptive, which cannot decrypt there. A .strm
// file carries the properties from the start.

var kodiLanguages = map[string]bool{"Hindi": true, "English": true}

// kodiFileName is the file name of a channel's .strm, without characters that
// break Kodi's HTTP directory parsing or paths.
func kodiFileName(name string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", "+", " plus ", ";", ",", "?", "", "#", "", "\"", "", "<", "", ">", "", "%", "")
	return strings.Join(strings.Fields(r.Replace(name)), " ") + ".strm"
}

type kodiFolder struct {
	name     string
	channels map[string]television.Channel // file name -> channel
}

func kodiFolders(lang string) (map[string]*kodiFolder, error) {
	resp, err := television.Channels()
	if err != nil {
		return nil, err
	}
	folders := map[string]*kodiFolder{}
	for _, ch := range television.PreferLanguageVariants(resp.Result, lang) {
		if !kodiLanguages[television.LanguageMap[ch.Language]] || strings.Contains(strings.ToUpper(ch.Name), "TEST") {
			continue
		}
		cat := television.CategoryMap[ch.Category]
		if cat == "" {
			cat = "Other"
		}
		f := folders[cat]
		if f == nil {
			f = &kodiFolder{name: cat, channels: map[string]television.Channel{}}
			folders[cat] = f
		}
		file := kodiFileName(ch.Name)
		for i := 2; ; i++ {
			if _, dup := f.channels[file]; !dup {
				break
			}
			file = kodiFileName(fmt.Sprintf("%s %d", ch.Name, i))
		}
		f.channels[file] = ch
	}
	return folders, nil
}

// kodiStrm returns the .strm content: the #KODIPROP lines and the stream URL,
// as in the M3U playlist.
func kodiStrm(ch television.Channel, hostURL, quality string) string {
	if EnableDRM && utils.ContainsString(ch.ID, drmList) {
		return fmt.Sprintf("#KODIPROP:inputstream=inputstream.adaptive\n"+
			"#KODIPROP:inputstream.adaptive.manifest_type=mpd\n"+
			"#KODIPROP:inputstream.adaptive.license_type=com.widevine.alpha\n"+
			"#KODIPROP:inputstream.adaptive.license_key=%s/live/key/%s?q=%s\n"+
			"%s/live/mpd/%s?q=%s\n", hostURL, ch.ID, quality, hostURL, ch.ID, quality)
	}
	return fmt.Sprintf("%s/live/%s/%s.m3u8\n", hostURL, quality, ch.ID)
}

func kodiListing(c *fiber.Ctx, title string, names []string) error {
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title></head><body><h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1><pre>\n")
	for _, n := range names {
		link := url.PathEscape(strings.TrimSuffix(n, "/"))
		if strings.HasSuffix(n, "/") {
			link += "/"
		}
		fmt.Fprintf(&b, "<a href=\"%s\">%s</a>\n", html.EscapeString(link), html.EscapeString(n))
	}
	b.WriteString("</pre></body></html>\n")
	c.Set("Cache-Control", "no-cache")
	c.Type("html", "utf-8")
	return c.SendString(b.String())
}

// KodiHandler serves /kodi/, /kodi/<category>/ and /kodi/<category>/<channel>.strm.
func KodiHandler(c *fiber.Ctx) error {
	raw := c.Params("*")
	rest, err := url.PathUnescape(raw)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	if !strings.HasSuffix(c.Path(), "/") && !strings.HasSuffix(rest, ".strm") {
		return c.Redirect(c.Path()+"/", fiber.StatusMovedPermanently)
	}

	folders, err := kodiFolders(preferredLanguage(c))
	if err != nil {
		return ErrorMessageHandler(c, err)
	}

	parts := strings.SplitN(strings.Trim(rest, "/"), "/", 2)
	if parts[0] == "" {
		names := make([]string, 0, len(folders))
		for n := range folders {
			names = append(names, n+"/")
		}
		return kodiListing(c, "JioTV", names)
	}

	f := folders[parts[0]]
	if f == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	if len(parts) == 1 || parts[1] == "" {
		names := make([]string, 0, len(f.channels))
		for n := range f.channels {
			names = append(names, n)
		}
		return kodiListing(c, f.name, names)
	}

	ch, ok := f.channels[parts[1]]
	if !ok {
		return c.SendStatus(fiber.StatusNotFound)
	}
	quality := c.Query("q", "high")
	hostURL := strings.ToLower(c.Protocol()) + "://" + c.Hostname()
	c.Set("Cache-Control", "no-cache")
	c.Type("txt", "utf-8")
	return c.SendString(kodiStrm(ch, hostURL, quality))
}
