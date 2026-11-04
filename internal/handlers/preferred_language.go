package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jiotv-go/jiotv_go/v3/internal/config"
)

// preferredLanguage returns the language whose per-language channel feeds
// should be shown (see television.PreferLanguageVariants). The "pl" query
// parameter overrides the config value, e.g. ?pl=off lists every feed.
func preferredLanguage(c *fiber.Ctx) string {
	if pl := c.Query("pl"); pl != "" {
		return pl
	}
	return config.Cfg.PreferredLanguage
}
