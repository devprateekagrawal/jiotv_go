package handlers

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

const psshTestMPD = `<?xml version="1.0"?><MPD xmlns="urn:mpeg:dash:schema:mpd:2011" xmlns:cenc="urn:mpeg:cenc:2013"><Period>` +
	`<BaseURL>/render.dash/host/H/path/P/dash/</BaseURL><AdaptationSet>` +
	`<ContentProtection schemeIdUri="urn:mpeg:dash:mp4protection:2011" value="cenc" cenc:default_KID="%KID%"/>` +
	`<ContentProtection schemeIdUri="urn:uuid:EDEF8BA9-79D6-4ACE-A3C8-27DCD51D21ED"/>` +
	`<SegmentTemplate initialization="Ch-$RepresentationID$.dash" media="Ch-$RepresentationID$-$Time$.dash"/>` +
	`<Representation id="video=1272000" bandwidth="1"/></AdaptationSet></Period></MPD>`

func TestInjectWidevinePSSHFromInit(t *testing.T) {
	// Real JioTV init pssh: content_id + protection_scheme, 50 bytes.
	box, _ := hex.DecodeString("000000327073736800000000edef8ba979d64acea3c827dcd51d21ed00000012220a393839393938383035304" + "8e3dc959b06")
	seg := append(append([]byte("\x00\x00\x00\x10ftypiso6\x00\x00\x00\x00junk"), box...), []byte("trailing")...)
	if got := findWidevinePSSH(seg); string(got) != string(box) {
		t.Fatalf("findWidevinePSSH = %x", got)
	}

	in := strings.Replace(psshTestMPD, "%KID%", "11111111-2222-3333-4444-555555555555", 1)
	calls := 0
	fetch := func(body []byte, pos int) []byte {
		calls++
		set := string(adaptationSetAround(body, pos))
		if !strings.Contains(set, `id="video=1272000"`) {
			t.Fatalf("wrong AdaptationSet: %s", set)
		}
		return box
	}
	out := string(injectWidevinePSSH([]byte(in), fetch))
	want := `<ContentProtection schemeIdUri="urn:uuid:EDEF8BA9-79D6-4ACE-A3C8-27DCD51D21ED"><cenc:pssh>` +
		base64.StdEncoding.EncodeToString(box) + `</cenc:pssh></ContentProtection>`
	if !strings.Contains(out, want) {
		t.Fatalf("pssh not injected:\n%s", out)
	}
	if !strings.HasPrefix(out, `<?xml version="1.0"?><MPD`) || strings.Count(out, "xmlns:cenc=") != 1 {
		t.Fatalf("MPD tag or prolog damaged:\n%s", out)
	}
	// Cached per KID: no second fetch.
	injectWidevinePSSH([]byte(in), fetch)
	if calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", calls)
	}
	// Already has a pssh: left alone.
	if again := string(injectWidevinePSSH([]byte(out), fetch)); again != out {
		t.Fatalf("second pass changed the MPD")
	}
}

func TestInjectWidevinePSSHFallback(t *testing.T) {
	in := strings.Replace(psshTestMPD, "%KID%", "5360E12D-4BB2-519A-9148-D1089A55908D", 1)
	in = strings.Replace(in, ` xmlns:cenc="urn:mpeg:cenc:2013"`, "", 1)
	out := string(injectWidevinePSSH([]byte(in), func([]byte, int) []byte { return nil }))
	kid, _ := hex.DecodeString("5360E12D4BB2519A9148D1089A55908D")
	box := widevinePSSHBox(kid)
	if len(box) != 50 || !strings.Contains(out, base64.StdEncoding.EncodeToString(box)) {
		t.Fatalf("fallback pssh missing:\n%s", out)
	}
	if strings.Count(out, `xmlns:cenc="urn:mpeg:cenc:2013"`) != 1 {
		t.Fatalf("namespace not added:\n%s", out)
	}
	clear := `<MPD><Period></Period></MPD>`
	if string(injectWidevinePSSH([]byte(clear), nil)) != clear {
		t.Fatalf("clear MPD changed")
	}
}
