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
	"github.com/jiotv-go/jiotv_go/v3/pkg/utils"
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
<widget xmlns="http://www.w3.org/ns/widgets" xmlns:tizen="http://tizen.org/ns/widgets" id="http://jiotv-go/tvapp" version="1.0.1" viewmodes="maximized">
  <tizen:application id="` + tvAppPackage + `.tv" package="` + tvAppPackage + `" required_version="2.3"/>
  <content src="index.html"/>
  <feature name="http://tizen.org/feature/screen.size.all"/>
  <icon src="icon.png"/>
  <name>JioTV</name>
  <access origin="*" subdomains="true"/>
  <tizen:allow-navigation>*</tizen:allow-navigation>
  <tizen:content-security-policy>default-src * data: blob: 'unsafe-inline' 'unsafe-eval'; script-src * 'unsafe-inline' 'unsafe-eval'; style-src * 'unsafe-inline'; img-src * data: blob:; media-src * data: blob:; connect-src *;</tizen:content-security-policy>
  <tizen:privilege name="http://tizen.org/privilege/internet"/>
  <tizen:privilege name="http://tizen.org/privilege/tv.inputdevice"/>
  <tizen:profile name="tv-samsung"/>
  <tizen:setting screen-orientation="landscape" context-menu="disable" background-support="disable" encryption="disable" install-location="auto" hwkey-event="enable"/>
</widget>
`

// tvAppReport runs first in <head>: it shows script errors on screen and sends
// them to /tvapp/log, so a blank screen on the TV can be read in the router log.
const tvAppReport = `<script>
(function () {
  var base = document.baseURI.replace(/\/$/, "");
  var keep = [];
  function mask(m) { return String(m).replace(/(auth|hdnea|__hdnea__|token)=[^&"\s]*/g, "$1=***"); }
  function send(m) {
    var u = base + "/tvapp/log?m=" + encodeURIComponent(mask(m).slice(0, 400));
    try { fetch(u, { mode: "no-cors", cache: "no-store" }).catch(function () {}); return; } catch (e) {}
    try { var i = new Image(); keep.push(i); i.src = u; } catch (e) {}
  }
  function show(m) {
    try {
      var d = document.getElementById("tvapperr");
      if (!d) {
        d = document.createElement("div");
        d.id = "tvapperr";
        d.style.cssText = "position:fixed;left:0;right:0;top:0;z-index:99;background:rgba(96,0,0,.85);color:#fff;font:18px monospace;padding:6px;white-space:pre-wrap;max-height:30%;overflow:hidden";
        (document.body || document.documentElement).appendChild(d);
      }
      d.textContent += mask(m) + String.fromCharCode(10);
    } catch (e) {}
  }
  function str(a) {
    try {
      if (a && a.code !== undefined && a.severity !== undefined) {
        return "shaka code=" + a.code + " cat=" + a.category + " data=" + JSON.stringify(a.data).slice(0, 200);
      }
      if (a instanceof Error) return a.name + ": " + a.message;
      if (typeof a === "object") return JSON.stringify(a).slice(0, 200);
    } catch (e) {}
    return String(a);
  }
  ["warn", "error"].forEach(function (k) {
    var orig = console[k];
    console[k] = function () {
      var m = k + ": " + Array.prototype.map.call(arguments, str).join(" ");
      send(m); show(m);
      try { orig.apply(console, arguments); } catch (e) {}
    };
  });
  window.addEventListener("error", function (e) {
    var m = "error: " + (e.message || (e.target && (e.target.src || e.target.href)) || "?") + " @" + (e.filename || "") + ":" + (e.lineno || "");
    send(m); show(m);
  }, true);
  window.addEventListener("unhandledrejection", function (e) {
    var m = "rejection: " + str(e.reason);
    send(m); show(m);
  });
  send("start " + navigator.userAgent + " screen=" + screen.width + "x" + screen.height + " inner=" + innerWidth + " dpr=" + devicePixelRatio);
  window.addEventListener("load", function () {
    var d = "diag shaka=" + !!window.shaka;
    try { d += " supported=" + shaka.Player.isBrowserSupported(); } catch (e) { d += " supported=?"; }
    d += " MSE=" + !!window.MediaSource + " EME=" + !!navigator.requestMediaKeySystemAccess + " secure=" + window.isSecureContext + " origin=" + location.origin;
    if (!navigator.requestMediaKeySystemAccess) { send(d); return; }
    navigator.requestMediaKeySystemAccess("com.widevine.alpha", [{ initDataTypes: ["cenc"], videoCapabilities: [{ contentType: 'video/mp4; codecs="avc1.42E01E"' }] }])
      .then(function () { send(d + " widevine=yes"); }, function (e) { send(d + " widevine=no " + str(e)); });
  });
})();
</script>
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
	page = strings.Replace(page, "<head>", "<head>\n<base href=\""+baseURL+"/\">\n"+tvAppReport, 1)
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
	case "log":
		// One line from the app's error reporter.
		m := strings.Map(func(r rune) rune {
			if r < 32 || r == 127 {
				return ' '
			}
			return r
		}, c.Query("m"))
		if len(m) > 400 {
			m = m[:400]
		}
		utils.Log.Printf("TV app: %s", m)
		return c.SendStatus(fiber.StatusNoContent)
	case "":
		if !strings.HasSuffix(c.Path(), "/") {
			return c.Redirect(c.Path()+"/", fiber.StatusMovedPermanently)
		}
		return kodiListing(c, "JioTV TV app", []string{tvAppFile})
	}
	return c.SendStatus(fiber.StatusNotFound)
}
