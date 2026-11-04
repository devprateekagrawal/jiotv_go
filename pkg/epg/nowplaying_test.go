package epg

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNowPlaying(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "epg.xml.gz")
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE tv SYSTEM "http://www.w3.org/2006/05/tv"><tv>
<channel id="1"><display-name>A</display-name></channel>
<programme channel="1" start="20261105100000 +0530" stop="20261105110000 +0530"><title lang="en">Morning &amp; News</title></programme>
<programme channel="1" start="20261105110000 +0530" stop="20261105120000 +0530"><title lang="en">Noon Show</title></programme>
<programme channel="2" start="20261105090000 +0530" stop="20261105100000 +0530"><title lang="en">Old</title></programme>
</tv>`
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	gz.Write([]byte(xmlData))
	gz.Close()
	f.Close()

	ist := time.FixedZone("IST", 5*3600+1800)
	got, err := NowPlaying(path, time.Date(2026, 11, 5, 10, 30, 0, 0, ist))
	if err != nil {
		t.Fatal(err)
	}
	if got["1"].Title != "Morning & News" {
		t.Fatalf("channel 1: %+v", got["1"])
	}
	if _, ok := got["2"]; ok {
		t.Fatalf("channel 2 should have nothing airing: %+v", got["2"])
	}
	got, _ = NowPlaying(path, time.Date(2026, 11, 5, 11, 0, 0, 0, ist))
	if got["1"].Title != "Noon Show" {
		t.Fatalf("at 11:00: %+v", got["1"])
	}
}
