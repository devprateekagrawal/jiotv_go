package television

import (
	"regexp"
	"strings"
)

// variantLanguageWords are the language names JioTV puts inside channel names
// to mark per-language feeds of the same channel, e.g. "Discovery Hindi",
// "Discovery Tamil", "Pogo Telugu".
var variantLanguageWords = []string{
	"Hindi", "English", "Tamil", "Telugu", "Kannada", "Malayalam", "Marathi",
	"Bengali", "Bangla", "Gujarati", "Punjabi", "Odia", "Oriya", "Bhojpuri",
}

var variantLanguageRe = regexp.MustCompile(`(?i)\b(` + strings.Join(variantLanguageWords, "|") + `)\b`)

var spaceRe = regexp.MustCompile(`\s+`)

// variantBaseName returns the channel name without its language word,
// lower-cased, plus the language word found ("" if none).
func variantBaseName(name string) (string, string) {
	lang := ""
	if m := variantLanguageRe.FindString(name); m != "" {
		lang = strings.ToLower(m)
	}
	base := variantLanguageRe.ReplaceAllString(name, " ")
	base = strings.ToLower(strings.TrimSpace(spaceRe.ReplaceAllString(base, " ")))
	return base, lang
}

// IsPreferredLanguageOff reports whether the preferred-language feature is disabled.
func IsPreferredLanguageOff(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "", "off", "none", "false", "0", "all":
		return true
	}
	return false
}

// PreferLanguageVariants keeps only the preferred-language feed of channels
// that JioTV carries as separate per-language feeds.
//
// JioTV streams carry a single audio language each, so "Hindi audio" means
// picking the "... Hindi" channel. Channels are grouped by their name with
// the language word removed ("Discovery Hindi", "Discovery Tamil" and
// "Discovery English" form one group). When a group has a feed whose name
// contains the preferred language, the group's other feeds are dropped.
// Groups without such a feed, and channels with no siblings, are kept as is,
// so no channel disappears unless its preferred-language version is listed.
func PreferLanguageVariants(channels []Channel, lang string) []Channel {
	if IsPreferredLanguageOff(lang) || len(channels) == 0 {
		return channels
	}
	want := strings.ToLower(strings.TrimSpace(lang))

	hasPreferred := make(map[string]bool)
	for _, ch := range channels {
		base, l := variantBaseName(ch.Name)
		if l == want {
			hasPreferred[base] = true
		}
	}
	if len(hasPreferred) == 0 {
		return channels
	}

	out := make([]Channel, 0, len(channels))
	for _, ch := range channels {
		base, l := variantBaseName(ch.Name)
		if hasPreferred[base] && l != want {
			continue
		}
		out = append(out, ch)
	}
	return out
}
