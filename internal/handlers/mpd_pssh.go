package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jiotv-go/jiotv_go/v3/pkg/utils"
)

var (
	// Self-closing Widevine ContentProtection, i.e. one without a <cenc:pssh>.
	widevineCPEmpty = regexp.MustCompile(`(?i)<ContentProtection\s+[^>]*schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"[^>]*?/>`)
	defaultKIDAttr  = regexp.MustCompile(`(?i)default_KID="([0-9a-f-]{32,36})"`)
	mpdOpenTag      = regexp.MustCompile(`<MPD[^>]*>`)
	baseURLTag      = regexp.MustCompile(`<BaseURL>([^<]*)</BaseURL>`)
	initAttr        = regexp.MustCompile(`initialization="([^"]+)"`)
	reprIDAttr      = regexp.MustCompile(`<Representation\s[^>]*?id="([^"]+)"`)
)

var widevineSystemID = []byte{0xed, 0xef, 0x8b, 0xa9, 0x79, 0xd6, 0x4a, 0xce, 0xa3, 0xc8, 0x27, 0xdc, 0xd5, 0x1d, 0x21, 0xed}

// psshCache maps a default KID (lower-case hex) to the base64 Widevine pssh box
// read from the init segment. KIDs change rarely, so this is fetched once.
var psshCache sync.Map

var psshClient = &http.Client{Timeout: 6 * time.Second}

// widevinePSSHBox builds a version 0 Widevine pssh box whose data holds only the
// key ID. Used when the init segment cannot be read.
func widevinePSSHBox(kid []byte) []byte {
	data := append([]byte{0x12, byte(len(kid))}, kid...)
	box := make([]byte, 32+len(data))
	binary.BigEndian.PutUint32(box[0:], uint32(len(box)))
	copy(box[4:], "pssh")
	copy(box[12:], widevineSystemID)
	binary.BigEndian.PutUint32(box[28:], uint32(len(data)))
	copy(box[32:], data)
	return box
}

// findWidevinePSSH returns the first complete Widevine pssh box in an MP4 segment.
func findWidevinePSSH(seg []byte) []byte {
	for off := 0; ; {
		i := bytes.Index(seg[off:], []byte("pssh"))
		if i < 0 {
			return nil
		}
		start := off + i - 4
		off += i + 4
		if start < 0 || start+32 > len(seg) {
			continue
		}
		size := int(binary.BigEndian.Uint32(seg[start:]))
		if size < 32 || start+size > len(seg) || !bytes.Equal(seg[start+12:start+28], widevineSystemID) {
			continue
		}
		return seg[start : start+size]
	}
}

// adaptationSetAround returns the <AdaptationSet> element that contains pos.
func adaptationSetAround(body []byte, pos int) []byte {
	start := bytes.LastIndex(body[:pos], []byte("<AdaptationSet"))
	if start < 0 {
		return nil
	}
	end := bytes.Index(body[pos:], []byte("</AdaptationSet>"))
	if end < 0 {
		return body[start:]
	}
	return body[start : pos+end]
}

// fetchInitPSSH reads the Widevine pssh from the init segment of the first
// Representation of the AdaptationSet, through this server's own /render.dash
// proxy at localBase (e.g. http://127.0.0.1:5001).
func fetchInitPSSH(body []byte, pos int, localBase string) []byte {
	base := baseURLTag.FindSubmatch(body)
	set := adaptationSetAround(body, pos)
	if base == nil || set == nil {
		return nil
	}
	ini := initAttr.FindSubmatch(set)
	if ini == nil {
		ini = initAttr.FindSubmatch(body)
	}
	rep := reprIDAttr.FindSubmatch(set)
	if ini == nil || rep == nil {
		return nil
	}
	path := strings.ReplaceAll(string(ini[1]), "$RepresentationID$", string(rep[1]))
	if strings.Contains(path, "$") || strings.Contains(path, "://") {
		return nil
	}
	resp, err := psshClient.Get(localBase + string(base[1]) + path)
	if err != nil {
		utils.Log.Printf("pssh: init segment: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		utils.Log.Printf("pssh: init segment HTTP %d", resp.StatusCode)
		return nil
	}
	seg, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return nil
	}
	return findWidevinePSSH(seg)
}

// injectWidevinePSSH adds a <cenc:pssh> to Widevine ContentProtection elements
// that have none. JioTV's MPDs carry the Widevine PSSH only in the init segment.
// Browsers (EME) read it from there, but Kodi's InputStream Adaptive takes the
// init data from the manifest only, so without it Kodi cannot open a DRM
// session ("A problem occurred with the DRM decryptor").
//
// The box is the init segment's own (it carries Jio's content ID, which the
// license server may need), cached per KID. initPSSH fetches it; when it
// returns nil a key-ID-only box is used instead. Call this after the BaseURL
// rewrite, so the init segment is fetched through the local proxy.
func injectWidevinePSSH(body []byte, initPSSH func(body []byte, pos int) []byte) []byte {
	locs := widevineCPEmpty.FindAllIndex(body, -1)
	if len(locs) == 0 {
		return body
	}

	var out []byte
	last := 0
	injected := false
	for _, loc := range locs {
		kids := defaultKIDAttr.FindAllSubmatch(body[:loc[0]], -1)
		if len(kids) == 0 {
			continue
		}
		kidHex := strings.ToLower(strings.ReplaceAll(string(kids[len(kids)-1][1]), "-", ""))
		kid, err := hex.DecodeString(kidHex)
		if err != nil || len(kid) != 16 {
			continue
		}

		var pssh string
		if v, ok := psshCache.Load(kidHex); ok {
			pssh = v.(string)
		} else if box := initPSSH(body, loc[0]); box != nil {
			pssh = base64.StdEncoding.EncodeToString(box)
			psshCache.Store(kidHex, pssh)
		} else {
			pssh = base64.StdEncoding.EncodeToString(widevinePSSHBox(kid))
		}

		tag := body[loc[0]:loc[1]]
		open := strings.TrimSpace(strings.TrimSuffix(string(tag), "/>")) + ">"
		out = append(out, body[last:loc[0]]...)
		out = append(out, open+"<cenc:pssh>"+pssh+"</cenc:pssh></ContentProtection>"...)
		last = loc[1]
		injected = true
	}
	if !injected {
		return body
	}
	out = append(out, body[last:]...)

	// Declare the cenc namespace if the MPD does not already.
	if loc := mpdOpenTag.FindIndex(out); loc != nil && !strings.Contains(string(out[loc[0]:loc[1]]), "xmlns:cenc=") {
		tag := strings.TrimSuffix(string(out[loc[0]:loc[1]]), ">")
		tag = strings.TrimSuffix(tag, "/") + ` xmlns:cenc="urn:mpeg:cenc:2013">`
		out = append(append(append([]byte{}, out[:loc[0]]...), tag...), out[loc[1]:]...)
	}
	return out
}
