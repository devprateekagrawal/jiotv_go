# JioTV Go 📺

JioTV Go, an exciting project that allows you to stream Live TV channels on the web and IPTV. It's a web wrapper around the JioTV Android app, utilizing the same API to fetch and stream channels.

Ready to dive in? Download the latest binary for your operating system from [here](https://github.com/jiotv-go/jiotv_go/releases/latest), and explore the [documentation](https://jiotv_go.rabil.me/) to start your JioTV Go adventure! 🚀

_Give us 🌟 on GitHub if you like this project!_

We have video tutorials for [Windows](https://youtu.be/BnNTYTSvVBc), and [Android](https://youtu.be/ejiuml11g8o) users. Please watch them if you are unsure about the installation process.

## Features 🌟

- 📺 Stream Live TV channels, just like in the JioTV Android app.
- 🎬 M3U playlist support for IPTV.
- 🌐 Web interface for watching Live TV.
- 📅 EPG (Electronic Program Guide) support in compressed GZipped XML or JSON format.
- 🎥 Quality selection (Low, Medium, High) supported.
- ⚙️ Configurable port and host.
- 🔐 Authentication using Jio number with OTP.
- 👥 Support for multiple clients simultaneously.
- 🚀 Written in Go, ensuring it's fast, lightweight, and portable.
- 💻 Command-line interface for server management and self-update.
- 🔄 Background start and stop feature.

Get Started with JioTV Go by following the [Get Started](https://jiotv_go.rabil.me/get_started) guide.

## Table of Contents

<details close>
  <summary>Click to expand/collapse</summary>
  
- [JioTV Go 📺](#jiotv-go-)
  - [Features 🌟](#features-)
  - [Table of Contents](#table-of-contents)
  - [Documentation](#documentation)
  - [Router / browser notes (`tv-hindi` fork)](#router--browser-notes-tv-hindi-fork)
  - [Join the community on Telegram:](#join-the-community-on-telegram)
  - [Star History](#star-history)
  - [Contributors](#contributors)
  - [Let's Make JioTV Go Better Together! 🤝](#lets-make-jiotv-go-better-together-)
    - [**Report Bugs**](#report-bugs)
    - [**Ready to Contribute? Join the Journey! 🚀**](#ready-to-contribute-join-the-journey-)
  - [**License: Attribution 4.0 International (CC BY 4.0)**](#license-attribution-40-international-cc-by-40)
</details>

## Documentation

The complete documentation for JioTV Go is available at https://jiotv_go.rabil.me/ 📖

## Router / browser notes (`tv-hindi` fork)

This fork runs as a service on an OpenWrt router (linux/arm64, procd service `jiotvgo`). A Samsung TV browser watches it over https. The router-side setup and reinstall steps are kept outside this repo, in the router guide (section Y). This section records what was found while getting HD to play in a TV browser, and what was changed for it (2026-11-06).

### Build for the router

```bash
CGO_ENABLED=0 GOEXPERIMENT=jsonv2,greenteagc GOOS=linux GOARCH=arm64 \
  go build -o build/jiotv_go-linux-arm64 -trimpath -ldflags="-s -w" .
```

Run it with `JIOTV_PATH_PREFIX=/etc/jiotv_go/ JIOTV_LOG_PATH=/tmp/jiotv_go JIOTV_DRM=true jiotv_go --skip-update-check serve --host 0.0.0.0 --port 5001`. `JIOTV_DRM` defaults to true in `internal/config/config.go`. Set it to false to give IPTV apps that can't do Widevine an all-HLS playlist.

### Widevine needs a secure context

Browsers only enable EME (Widevine) on `https://` pages or `localhost`. On `http://<lan-ip>:5001` the DRM player's Widevine check fails, and the player falls back to HLS. For Colors HD that means 360p at most. Fix: put a TLS proxy in front. On the router that is `stunnel` with a self-signed cert (`:8443` → `127.0.0.1:5001`). The TV accepts the cert warning once, and DASH + Widevine then plays up to 1080p.

Behind such a proxy, `c.Protocol()` still reports `http`. So any absolute URL the server builds from it is wrong on the https port. Prefer relative URLs in pages.

### What Jio serves (checked 2026-11-06)

| Channel | HLS | DASH |
|---|---|---|
| Colors HD (144), Aaj Tak, Colors Cineplex | `high`/`medium` master is audio only; `auto` has video (max 360p) | Widevine, up to 1080p |
| SET HD (291), Sony SAB HD, Sony Max HD | 720p/1080p | clear (no `ContentProtection`), up to 1080p, though the API still returns a license URL |
| Star Plus HD | 404 | not checked |

### Changes in this fork

| Problem | Cause | Fix |
|---|---|---|
| Colors HD plays audio only over HLS | Jio's `high`/`medium` master playlists for some channels have no video variant | `RenderHandler` detects an audio-only master (`isAudioOnlyMaster`, tested in `audio_only_test.go`) and redirects to `/live/auto/<id>.m3u8` |
| SET HD (and other Sony channels) never plays in a browser: `render.mpd` 403, then the HLS fallback's `render.ts` 403 | The proxy forwarded the browser's `Origin` and `Referer` headers to the Jio/Sony CDN, which answers 403 when it sees them. curl and IPTV apps don't send them, so they worked | `internalUtils.StripBrowserHeaders` removes both headers. It is called in `ProxyRequest` (`.ts`, EPG, images), `MpdHandler` and the DASH segment handler |
| A clear DASH stream is forced through Widevine | The API sends a license URL for clear Sony streams. The player configured Widevine for them, and on http fell back to HLS | `player_drm.html` fetches the manifest first. If it has no `ContentProtection`, it drops the license URL and plays clear DASH. This also works on plain http |
| No channel logos on the https port | Logo URLs were built as `c.Protocol()://host/jtvimage/...` = `http://<ip>:8443/...` | `IndexHandler` uses relative `/jtvimage/...` |

Known leftover: when an MPD has no `<UTCTiming>`, `MpdHandler` adds `http://<host>/dashtime`, which is wrong behind an https proxy. The browser player sets `clockSyncUri: "/dashtime"` itself, so browser playback isn't affected.

### `/tv`: remote-friendly page for TV browsers

`https://<router>:8443/tv` is one page for watching TV with only a remote. There is a channel list first, then the player. Its parts:

* `GET /tv` renders `web/views/tv.html`.
* `GET /tv/stream/:id` (`internal/handlers/tv.go`) returns `{mpd, license, host, path, hls}`. It uses `getDrmMpd`. All URLs are relative, so they work behind the https proxy.
* The page loads `/channels` and `/nowplaying` (refreshed every minute). Channels sort Hindi first, then English, HD before SD. `TEST` channels are hidden.
* The tabs are Favourites, Serial, News, Movies, Sports, Bhakti, Bachche, Gaane and Sab. Category tabs show only Hindi and English channels.

Favourites, the last tab and the last channel are kept in `localStorage` (`tvFavs`, `tvTab`, `tvLast`). The default favourites are 144, 291, 471 and 173. Opening `/tv` starts the last channel; `/tv?home=1` skips that.

Keys:

| Where | Key | What it does |
|---|---|---|
| List | arrows | move between tiles |
| List | Up from the first row | goes to the tabs |
| List | OK | plays the channel |
| Player | Up/Down, CH+/CH− (427/428, PageUp/PageDown) | change channel within the tab |
| Player | OK | opens the info bar (show, favourite toggle); OK again toggles the favourite |
| Player | Back (Esc, Backspace, Tizen 10009, webOS 461, or the browser's own Back via `popstate`) | closes the bar, then goes back to the list |

* **Pointer mode.** If the TV browser moves a mouse pointer instead of sending arrow keys, tiles are clickable. A click on the player shows the bar with Pichla, Agla, Favourite and List buttons. The pointer hides after 3 s idle.
* **Key test.** `/tv?keys=1` shows on screen which key reached the page.

One Shaka player is reused for every channel. Each channel falls back in this order:

1. DASH. A clear manifest (no `ContentProtection`) plays without Widevine. An encrypted one plays with Widevine `SW_SECURE_CRYPTO`, `clockSyncUri /dashtime`, and host/path params on `render.dash` segments.
2. HLS `/live/high/<id>.m3u8`, through Shaka. It needs a secure context, because Shaka's AES-128 decryption uses Web Crypto.
3. The old `/player/<id>?q=auto&af=1` page, in an iframe.

Checked 2026-11-06 in a desktop browser:
* SET HD played as clear DASH at 720p on plain http.
* Colors HD on http fell through to the iframe player.
* On `localhost` (a secure context) Colors HD played over Shaka HLS, because that browser had no Widevine.

### /kodi/: channels as .strm files for Kodi

`/kodi/` is an HTTP directory with one folder per category (Hindi and English channels) and one `.strm` file per channel, holding the same `#KODIPROP` lines as the M3U. In Kodi add it as a video source: Videos > Files > Add videos > Browse > Add network location, protocol "Web server directory (HTTP)", server `192.168.1.1`, port `5001`, path `kodi/`.

Use it on Kodi builds that play DRM in the browser (kodi-tizen-wasm on Samsung TVs). They pick the browser player from the item's `#KODIPROP` lines, but a channel of the IPTV Simple PVR client gets those only after the player is chosen, so its Widevine channels fail in InputStream Adaptive ("A problem occurred with the DRM decryptor"). A `.strm` file carries them from the start. `?q=` on a `.strm` URL sets the quality (default `high`).

### /kodilog/: send Kodi's log to the server

A debug kodi.log is larger than paste.kodi.tv takes (2 MB). Add `http://<server>:5001/kodilog/` as a source in Kodi's File manager, then Add-ons > Install from zip file > that source > `script.jiotv.logsend.zip`, and run *JioTV: send log to router* from Add-ons > Program add-ons. It POSTs kodi.log and kodi.old.log to `/kodilog/upload`; the server keeps the newest 3 in `<temp dir>/jiotv_go/kodi-logs` (RAM on a router).

### Debugging tips

* Access log: `$JIOTV_LOG_PATH/jiotv_go.log` (UTC times). For a channel, watch `render.mpd`, `render.dash`, `render.m3u8`, `render.key` and `render.ts`. Many `render.dash` 200s mean DASH is playing. `render.ts` lines after `/player/<id>?...&af=1` mean the DRM player gave up and fell back to HLS.
* To reproduce browser-only failures with curl, add `-H "Origin: ..." -H "Referer: ..."`.
* Check whether a stream is really encrypted: look for `ContentProtection` in the MPD, or for `encv`/`tenc`/`pssh` boxes in the init segment.

## Join the community on Telegram:

- [Announcement Channel (`jiotv_go`)](https://telegram.me/jiotv_go)
- [Support Group (`jiotv_go_chat`)](https://telegram.me/jiotv_go_chat)

## Contributors

[![Contributors](https://contributors-img.web.app/image?repo=jiotv-go/jiotv_go)](https://github.com/jiotv-go/jiotv_go/graphs/contributors)

## Let's Make JioTV Go Better Together! 🤝

### **Report Bugs**

Found a pesky bug? No worries! Please help us improve JioTV Go by creating an issue [here](https://github.com/jiotv-go/jiotv_go/issues/new/choose). Be sure to include detailed steps to reproduce the bug, describe the expected behavior, and, if possible, attach screenshots. Your feedback is invaluable!

### **Ready to Contribute? Join the Journey! 🚀**

We wholeheartedly welcome your contributions. If you have ideas, fixes, or enhancements in mind, don't hesitate to create a pull request with your changes. For significant alterations, start by creating an issue to discuss your plans with us. Together, we can make JioTV Go even more incredible.

## **License: Attribution 4.0 International (CC BY 4.0)**

**Embrace the Spirit of Free Software!** JioTV Go is open-source and free to use. We're committed to keeping it accessible to everyone. If you come across any unauthorized attempts to sell this project, please report them to [me](mailto:mail@rabil.me) so we can take swift action. Your support is essential in safeguarding our project's values. 🙌📜💼
