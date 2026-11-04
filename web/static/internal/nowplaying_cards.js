// nowplaying_cards.js — show the programme airing now on each channel card.
// One request to /nowplaying (built from the local EPG file) for all cards,
// refreshed every 5 minutes. Does nothing when EPG is off.
(function () {
  const REFRESH_MS = 5 * 60 * 1000;

  function fmt(ts) {
    const d = new Date(ts * 1000);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }

  function slotFor(card) {
    let el = card.querySelector(".np-title");
    if (!el) {
      const body = card.querySelector(":scope > div");
      if (!body) return null;
      el = document.createElement("span");
      el.className = "np-title";
      const bar = document.createElement("span");
      bar.className = "np-bar";
      bar.hidden = true;
      bar.appendChild(document.createElement("i"));
      body.appendChild(el);
      body.appendChild(bar);
    }
    return el;
  }

  async function refresh() {
    let shows;
    try {
      const res = await fetch("/nowplaying", { cache: "no-store" });
      if (!res.ok) return;
      shows = await res.json();
    } catch (_) {
      return;
    }
    if (!shows || typeof shows !== "object") return;
    const now = Date.now() / 1000;
    document.querySelectorAll("a.card[data-channel-id]").forEach((card) => {
      const show = shows[card.dataset.channelId];
      const el = slotFor(card);
      if (!el) return;
      const bar = el.nextElementSibling;
      if (!show || !show.t) {
        el.textContent = "";
        if (bar) bar.hidden = true;
        return;
      }
      el.textContent = show.t;
      el.title = show.t + " (" + fmt(show.s) + " - " + fmt(show.e) + ")";
      if (bar && show.e > show.s) {
        const pct = Math.max(0, Math.min(100, ((now - show.s) / (show.e - show.s)) * 100));
        bar.firstChild.style.width = pct.toFixed(1) + "%";
        bar.hidden = false;
      }
    });
  }

  function start() {
    refresh();
    setInterval(refresh, REFRESH_MS);
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
