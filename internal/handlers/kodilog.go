package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jiotv-go/jiotv_go/v3/pkg/utils"
)

// /kodilog/ gets Kodi's log onto this server, with no paste site in between
// (paste.kodi.tv takes 2 MB at most, and a debug log is larger).
//
//	GET  /kodilog/                          lists the add-on zip, for a Kodi file source
//	GET  /kodilog/script.jiotv.logsend.zip  a script add-on that sends kodi.log here
//	POST /kodilog/upload?name=kodi.log      saves the body (PUT /kodilog/upload/<name> too)
//
// Logs are kept in <temp dir>/jiotv_go/kodi-logs (RAM on a router, not
// flash), the newest kodiLogKeep files.

const (
	kodiLogAddonID  = "script.jiotv.logsend"
	kodiLogMaxBytes = 64 << 20
	kodiLogKeep     = 3
)

var kodiLogName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

const kodiLogAddonXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<addon id="` + kodiLogAddonID + `" name="JioTV: send log to router" version="1.0.0" provider-name="jiotv_go">
  <requires>
    <import addon="xbmc.python" version="3.0.0"/>
  </requires>
  <extension point="xbmc.python.script" library="default.py">
    <provides>executable</provides>
  </extension>
  <extension point="xbmc.addon.metadata">
    <summary lang="en_GB">Sends kodi.log to the JioTV Go server</summary>
    <description lang="en_GB">Uploads kodi.log and kodi.old.log to the JioTV Go server it was installed from.</description>
    <platform>all</platform>
  </extension>
</addon>
`

// kodiLogScript is default.py; %s is the server's base URL.
const kodiLogScript = `import urllib.request

import xbmc
import xbmcgui
import xbmcvfs

BASE = %q


def read(path):
    f = xbmcvfs.File(path)
    try:
        return bytes(f.readBytes())
    finally:
        f.close()


def send_post(name, data):
    req = urllib.request.Request(BASE + "/kodilog/upload?name=" + name, data=data,
                                 method="POST", headers={"Content-Type": "text/plain"})
    with urllib.request.urlopen(req, timeout=180) as r:
        return r.read().decode("utf-8", "replace").strip()


def send_put(name, data):
    f = xbmcvfs.File(BASE + "/kodilog/upload/" + name, "w")
    try:
        if not f.write(bytearray(data)):
            raise IOError("write failed")
    finally:
        f.close()
    return "sent (PUT)"


def main():
    lines = []
    progress = xbmcgui.DialogProgress()
    progress.create("JioTV log", "Sending...")
    try:
        for name in ("kodi.log", "kodi.old.log"):
            path = "special://logpath/" + name
            if not xbmcvfs.exists(path):
                continue
            progress.update(0, "Sending " + name)
            try:
                data = read(path)
            except Exception as e:
                lines.append("%%s: cannot read: %%s" %% (name, e))
                continue
            if not data:
                continue
            try:
                lines.append("%%s: %%s" %% (name, send_post(name, data)))
            except Exception as e:
                xbmc.log("jiotv logsend: POST failed: %%s" %% e, xbmc.LOGWARNING)
                try:
                    lines.append("%%s: %%s" %% (name, send_put(name, data)))
                except Exception as e2:
                    lines.append("%%s: failed: %%s / %%s" %% (name, e, e2))
    finally:
        progress.close()
    xbmcgui.Dialog().ok("JioTV log", "\n".join(lines) or "No log found")


main()
`

func kodiLogDir() string {
	return filepath.Join(os.TempDir(), "jiotv_go", "kodi-logs")
}

func kodiLogAddonZip(baseURL string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := []struct{ name, body string }{
		{kodiLogAddonID + "/addon.xml", kodiLogAddonXML},
		{kodiLogAddonID + "/default.py", fmt.Sprintf(kodiLogScript, baseURL)},
	}
	for _, f := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.name, Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(w, f.body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// saveKodiLog writes r to a new file in dir and removes all but the newest
// kodiLogKeep files. It returns the file's path and size.
func saveKodiLog(dir, name string, r io.Reader) (string, int64, error) {
	name = kodiLogName.ReplaceAllString(name, "_")
	if name == "" || strings.Trim(name, ".") == "" {
		name = "kodi.log"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, err
	}
	path := filepath.Join(dir, time.Now().UTC().Format("20060102-150405")+"-"+name)
	f, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	n, err := io.Copy(f, io.LimitReader(r, kodiLogMaxBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > kodiLogMaxBytes {
		err = fmt.Errorf("log larger than %d MB", kodiLogMaxBytes>>20)
	}
	if err != nil {
		os.Remove(path)
		return "", 0, err
	}

	if entries, err := os.ReadDir(dir); err == nil {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for i := 0; i < len(names)-kodiLogKeep; i++ {
			os.Remove(filepath.Join(dir, names[i]))
		}
	}
	return path, n, nil
}

// KodiLogHandler serves /kodilog/.
func KodiLogHandler(c *fiber.Ctx) error {
	rest := strings.Trim(c.Params("*"), "/")
	baseURL := strings.ToLower(c.Protocol()) + "://" + c.Hostname()

	switch {
	case rest == "upload" || strings.HasPrefix(rest, "upload/"):
		if c.Method() != fiber.MethodPost && c.Method() != fiber.MethodPut {
			return c.SendStatus(fiber.StatusMethodNotAllowed)
		}
		name := c.Query("name", strings.TrimPrefix(strings.TrimPrefix(rest, "upload"), "/"))
		var body io.Reader = bytes.NewReader(c.Body())
		if s := c.Context().RequestBodyStream(); s != nil {
			body = s
		}
		path, n, err := saveKodiLog(kodiLogDir(), name, body)
		if err != nil {
			utils.Log.Printf("Kodi log upload failed: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		utils.Log.Printf("Kodi log saved: %s (%d bytes)", path, n)
		if c.Method() == fiber.MethodPut {
			return c.SendStatus(fiber.StatusCreated)
		}
		return c.SendString(fmt.Sprintf("saved on the server, %.1f MB", float64(n)/(1<<20)))

	case rest == kodiLogAddonID+".zip":
		z, err := kodiLogAddonZip(baseURL)
		if err != nil {
			return ErrorMessageHandler(c, err)
		}
		c.Set("Cache-Control", "no-cache")
		c.Type("zip")
		return c.Send(z)

	case rest == "":
		if !strings.HasSuffix(c.Path(), "/") {
			return c.Redirect(c.Path()+"/", fiber.StatusMovedPermanently)
		}
		return kodiListing(c, "JioTV tools", []string{kodiLogAddonID + ".zip"})
	}
	return c.SendStatus(fiber.StatusNotFound)
}
