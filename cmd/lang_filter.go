package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jiotv-go/jiotv_go/v3/internal/config"
)

// HindiEnglishLanguages are the language IDs that --lan_filter=1 keeps
// on the channel list: 1 = Hindi, 6 = English.
var HindiEnglishLanguages = []int{1, 6}

// ApplyLangFilter sets which languages the channel list shows, from the
// serve command's --lan_filter flag, overriding default_languages in the
// config file and JIOTV_DEFAULT_LANGUAGES:
//
//	""  flag not given: keep what the config says
//	"0" all channels
//	"1" Hindi + English only
//	"1,6,5" (any other list) exactly those language IDs
func ApplyLangFilter(value string) error {
	value = strings.TrimSpace(value)
	switch value {
	case "":
		return nil
	case "0":
		config.Cfg.DefaultLanguages = nil
		return nil
	case "1":
		config.Cfg.DefaultLanguages = append([]int(nil), HindiEnglishLanguages...)
		return nil
	}
	var ids []int
	for _, part := range strings.Split(value, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id < 0 {
			return fmt.Errorf("invalid --lan_filter %q: use 0 (all), 1 (Hindi + English) or language IDs like 1,6", value)
		}
		ids = append(ids, id)
	}
	config.Cfg.DefaultLanguages = ids
	return nil
}
