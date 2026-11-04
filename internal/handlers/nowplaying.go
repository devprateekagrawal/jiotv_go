package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jiotv-go/jiotv_go/v3/pkg/epg"
	pkgUtils "github.com/jiotv-go/jiotv_go/v3/pkg/utils"
)

// NowPlayingHandler returns the show airing now on each channel, read from
// the local epg.xml.gz (needs epg = true). One small JSON for all channels,
// so the channel list can label every card without calling the JioTV API.
// Response: {"<channel_id>": {"t": title, "s": start, "e": end}} (unix seconds).
func NowPlayingHandler(c *fiber.Ctx) error {
	shows, err := epg.NowPlaying(pkgUtils.GetPathPrefix()+"epg.xml.gz", time.Now())
	if err != nil {
		// No EPG file yet: answer with an empty map so the page just shows no labels.
		c.Set("Cache-Control", "no-store")
		return c.JSON(fiber.Map{})
	}
	c.Set("Cache-Control", "max-age=60")
	return c.JSON(shows)
}
