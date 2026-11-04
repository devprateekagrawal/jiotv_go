package epg

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

// NowShow is the programme airing on a channel at a given time.
type NowShow struct {
	Title string `json:"t"`
	Start int64  `json:"s"` // unix seconds
	End   int64  `json:"e"` // unix seconds
}

type slot struct {
	start, end int64
	title      string
}

var nowCache struct {
	sync.Mutex
	path    string
	modTime time.Time
	byChan  map[string][]slot
}

// parseEPGTime parses the XMLTV time format written by formatTime.
func parseEPGTime(s string) (int64, bool) {
	t, err := time.Parse("20060102150405 -0700", s)
	if err != nil {
		return 0, false
	}
	return t.Unix(), true
}

// loadSlots streams the gzipped XMLTV file and keeps only start, stop and
// title for each programme, sorted by start time per channel.
func loadSlots(path string) (map[string][]slot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	dec := xml.NewDecoder(gz)
	dec.Strict = false
	out := make(map[string][]slot)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "programme" {
			continue
		}
		var p Programme
		if err := dec.DecodeElement(&p, &se); err != nil {
			continue
		}
		start, ok1 := parseEPGTime(p.Start)
		end, ok2 := parseEPGTime(p.Stop)
		if !ok1 || !ok2 || p.Channel == "" {
			continue
		}
		out[p.Channel] = append(out[p.Channel], slot{start: start, end: end, title: p.Title.Value})
	}
	for id := range out {
		s := out[id]
		sort.Slice(s, func(i, j int) bool { return s[i].start < s[j].start })
	}
	return out, nil
}

// NowPlaying returns the programme airing at time now on every channel found
// in the XMLTV file at path. The parsed file is cached and re-read only when
// its modification time changes.
func NowPlaying(path string, now time.Time) (map[string]NowShow, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	nowCache.Lock()
	if nowCache.byChan == nil || nowCache.path != path || !nowCache.modTime.Equal(st.ModTime()) {
		slots, err := loadSlots(path)
		if err != nil {
			nowCache.Unlock()
			return nil, err
		}
		nowCache.path, nowCache.modTime, nowCache.byChan = path, st.ModTime(), slots
	}
	byChan := nowCache.byChan
	nowCache.Unlock()

	ts := now.Unix()
	res := make(map[string]NowShow, len(byChan))
	for id, slots := range byChan {
		for _, s := range slots {
			if s.start > ts {
				break
			}
			if s.end > ts {
				res[id] = NowShow{Title: s.title, Start: s.start, End: s.end}
				break
			}
		}
	}
	return res, nil
}
