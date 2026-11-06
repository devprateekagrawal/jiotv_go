package handlers

import "testing"

func TestIsAudioOnlyMaster(t *testing.T) {
	cases := map[string]bool{
		"#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=40000,CODECS=\"mp4a.40.2\",AUDIO=\"a\"\n/r.m3u8\n":                         true,
		"#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1523000,CODECS=\"mp4a.40.2,avc1.4D401F\",RESOLUTION=1280x720\n/r.m3u8\n": false,
		"#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXTINF:6,\nseg.ts\n":                                                      false,
	}
	for in, want := range cases {
		if got := isAudioOnlyMaster([]byte(in)); got != want {
			t.Errorf("isAudioOnlyMaster(%q) = %v, want %v", in, got, want)
		}
	}
}
