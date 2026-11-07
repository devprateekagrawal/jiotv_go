package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jiotv-go/jiotv_go/v3/web"
)

// /tvapp/ builds the /tv page as a Samsung Tizen web app (.wgt).
//
// Why: the TV's browser turns the remote's arrow keys into a mouse pointer, so
// the page never gets them. In an app the page gets every remote key. The page
// is packed into the app (a local, secure page), so EME/Widevine works without
// the https proxy, and it talks to this server through a <base> URL.
//
//	GET /tvapp/              lists the .wgt
//	GET /tvapp/jiotv-tv.wgt  the app, unsigned; the installer signs it

const (
	tvAppFile    = "jiotv-tv.wgt"
	tvAppPackage = "JioTVGoApp" // Tizen package ids are 10 alphanumerics
)

const tvAppConfig = `<?xml version="1.0" encoding="UTF-8"?>
<widget xmlns="http://www.w3.org/ns/widgets" xmlns:tizen="http://tizen.org/ns/widgets" id="http://jiotv-go/tvapp" version="1.0.0" viewmodes="maximized">
  <tizen:application id="` + tvAppPackage + `.tv" package="` + tvAppPackage + `" required_version="2.3"/>
  <content src="index.html"/>
  <feature name="http://tizen.org/feature/screen.size.all"/>
  <icon src="icon.png"/>
  <name>JioTV</name>
  <access origin="*" subdomains="true"/>
  <tizen:allow-navigation>*</tizen:allow-navigation>
  <tizen:privilege name="http://tizen.org/privilege/internet"/>
  <tizen:privilege name="http://tizen.org/privilege/tv.inputdevice"/>
  <tizen:profile name="tv-samsung"/>
  <tizen:setting screen-orientation="landscape" context-menu="disable" background-support="disable" encryption="disable" install-location="auto" hwkey-event="enable"/>
</widget>
`

// tvAppScript registers the remote's channel and media keys, and makes Back
// on the channel list close the app (in a browser it leaves the page).
const tvAppScript = `<script>
(function () {
  try {
    ["ChannelUp", "ChannelDown", "MediaPlayPause", "MediaPlay", "MediaPause", "MediaStop"].forEach(function (k) {
      try { tizen.tvinputdevice.registerKey(k); } catch (e) {}
    });
  } catch (e) {}
  document.addEventListener("keydown", function (e) {
    if (e.keyCode !== 10009) return;
    var pl = document.getElementById("pl");
    if (pl && pl.classList.contains("on")) return;
    try { tizen.application.getCurrentApplication().exit(); } catch (err) {}
  }, true);
})();
</script>
`

// tvAppIndex is the /tv page with its URLs pointing at baseURL.
func tvAppIndex(page, baseURL string) (string, error) {
	if !strings.Contains(page, "<head>") || !strings.Contains(page, "</body>") {
		return "", fmt.Errorf("tv page has no <head> or </body>")
	}
	page = strings.Replace(page, "<head>", "<head>\n<base href=\""+baseURL+"/\">", 1)
	page = strings.ReplaceAll(page, "new URL(req.uris[0], location.href)", "new URL(req.uris[0], document.baseURI)")
	i := strings.LastIndex(page, "</body>")
	return page[:i] + tvAppScript + page[i:], nil
}

func tvAppWgt(baseURL string) ([]byte, error) {
	page, err := fs.ReadFile(web.GetViewFiles(), "views/tv.html")
	if err != nil {
		return nil, err
	}
	index, err := tvAppIndex(string(page), baseURL)
	if err != nil {
		return nil, err
	}
	icon, err := fs.ReadFile(web.GetStaticFiles(), "static/icons/icon-medium.png")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	modified := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	files := []struct {
		name string
		body []byte
	}{
		{"config.xml", []byte(tvAppConfig)},
		{"index.html", []byte(index)},
		{"icon.png", icon},
	}
	for _, f := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Deflate, Modified: modified})
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, bytes.NewReader(f.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// TVAppHandler serves /tvapp/.
func TVAppHandler(c *fiber.Ctx) error {
	rest := strings.Trim(c.Params("*"), "/")
	switch rest {
	case tvAppFile:
		baseURL := strings.ToLower(c.Protocol()) + "://" + c.Hostname()
		wgt, err := tvAppWgt(baseURL)
		if err != nil {
			return ErrorMessageHandler(c, err)
		}
		c.Set("Cache-Control", "no-cache")
		c.Set("Content-Type", "application/widget")
		c.Set("Content-Disposition", "attachment; filename=\""+tvAppFile+"\"")
		return c.Send(wgt)
	case "":
		if !strings.HasSuffix(c.Path(), "/") {
			return c.Redirect(c.Path()+"/", fiber.StatusMovedPermanently)
		}
		return kodiListing(c, "JioTV TV app", []string{tvAppFile})
	}
	return c.SendStatus(fiber.StatusNotFound)
}
