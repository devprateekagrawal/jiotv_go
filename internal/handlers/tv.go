package handlers

import (
	"github.com/gofiber/fiber/v2"
	internalUtils "github.com/jiotv-go/jiotv_go/v3/internal/utils"
	"github.com/jiotv-go/jiotv_go/v3/pkg/television"
	"github.com/jiotv-go/jiotv_go/v3/pkg/utils"
)

// TVHandler renders /tv, a single-page, remote-friendly channel browser and
// player for TV browsers. The page loads /channels and /nowplaying itself and
// asks /tv/stream/:id for the stream of the channel it plays.
func TVHandler(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-cache")
	return c.Render("views/tv", fiber.Map{})
}

// TVStreamHandler returns the playback URLs for one channel as JSON:
//
//	{"mpd": "...", "license": "...", "host": "...", "path": "...", "hls": "..."}
//
// mpd is empty when DASH is off or unavailable; the page then plays hls.
// All URLs are relative, so they work behind an https proxy.
func TVStreamHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	c.Set("Cache-Control", "no-store")

	if isCustomChannel(id) {
		channel, exists := television.GetCustomChannelByID(id)
		if !exists {
			return internalUtils.NotFoundError(c, "channel not found")
		}
		return c.JSON(fiber.Map{"hls": channel.URL})
	}

	out := fiber.Map{"hls": utils.BuildHLSPlayURL("high", id)}
	if !EnableDRM {
		return c.JSON(out)
	}

	EnsureFreshCredentials()
	mpd, err := getDrmMpd(id, "auto")
	if err != nil && ForceRefreshCredentials() {
		mpd, err = getDrmMpd(id, "auto")
	}
	if err != nil || mpd == nil || mpd.PlayUrl == "" {
		utils.Log.Printf("TV stream %s: no DASH, using HLS: %v", id, err)
		return c.JSON(out)
	}

	out["mpd"] = mpd.PlayUrl
	out["license"] = mpd.LicenseUrl
	out["host"] = mpd.Tv_url_host
	out["path"] = mpd.Tv_url_path
	return c.JSON(out)
}
