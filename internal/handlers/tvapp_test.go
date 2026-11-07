package handlers

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestTVAppWgt(t *testing.T) {
	z, err := tvAppWgt("http://192.168.1.1:5001")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(z), int64(len(z)))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		r, _ := f.Open()
		var b bytes.Buffer
		b.ReadFrom(r)
		got[f.Name] = b.String()
	}
	idx := got["index.html"]
	for _, want := range []string{`<base href="http://192.168.1.1:5001/">`, "document.baseURI", "tizen.tvinputdevice.registerKey", "shaka-player.ui.js"} {
		if !strings.Contains(idx, want) {
			t.Errorf("index.html lacks %q", want)
		}
	}
	if strings.Contains(idx, "location.href)") {
		t.Error("index.html still resolves against location.href")
	}
	if !strings.Contains(got["config.xml"], `package="JioTVGoApp"`) || len(got["icon.png"]) == 0 {
		t.Error("config.xml or icon.png missing")
	}
}
