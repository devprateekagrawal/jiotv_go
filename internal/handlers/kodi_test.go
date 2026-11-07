package handlers

import (
	"strings"
	"testing"

	"github.com/jiotv-go/jiotv_go/v3/pkg/television"
)

func TestKodiFileName(t *testing.T) {
	cases := map[string]string{
		"Colors HD":       "Colors HD.strm",
		"Sony SAB/HD":     "Sony SAB-HD.strm",
		"Star+  Bharat;x": "Star plus Bharat,x.strm",
	}
	for in, want := range cases {
		if got := kodiFileName(in); got != want {
			t.Errorf("kodiFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestKodiStrm(t *testing.T) {
	old := EnableDRM
	defer func() { EnableDRM = old }()
	EnableDRM = true
	drm := television.Channel{ID: drmList[0]}
	got := kodiStrm(drm, "http://h:5001", "high")
	for _, want := range []string{
		"#KODIPROP:inputstream=inputstream.adaptive\n",
		"#KODIPROP:inputstream.adaptive.license_key=http://h:5001/live/key/" + drmList[0] + "?q=high\n",
		"\nhttp://h:5001/live/mpd/" + drmList[0] + "?q=high\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DRM strm missing %q:\n%s", want, got)
		}
	}
	if got := kodiStrm(television.Channel{ID: "notdrm-x"}, "http://h:5001", "high"); got != "http://h:5001/live/high/notdrm-x.m3u8\n" {
		t.Errorf("HLS strm = %q", got)
	}
}
