// tv_layouts.js — six TV styles for the channel list, picked from a Style
// popup and remembered per browser (localStorage "tvStyle"). "classic" keeps
// the original page untouched.
//
// The styles are built from the channel cards the server already rendered
// in #original-channels-grid, so the server's filters (preferred language,
// default categories) still apply. Opening a tile clicks the matching
// original card, so catchup mode and the subscription warning keep working.
(function () {
  "use strict";

  var STYLES = [
    ["classic", "Classic", "the original page"],
    ["1", "Lite Grid", "big tiles, lightest"],
    ["2", "Rows", "one row per category"],
    ["3", "Guide", "TV guide list"],
    ["4", "Side Rail", "categories on the left"],
    ["5", "List + Preview", "list with a big detail panel"],
    ["6", "Apple TV", "light, rounded cards"]
  ];
  var CAT = { 5: "Entertainment", 6: "Movies", 7: "Kids", 8: "Sports", 9: "Lifestyle", 10: "Infotainment", 12: "News",
    13: "Music", 15: "Devotional", 16: "Business", 17: "Educational", 18: "Shopping", 19: "JioDarshan" };
  var LOCK_SVG = '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path fill-rule="evenodd" d="M12 1.5a5.25 5.25 0 0 0-5.25 5.25v3a3 3 0 0 0-3 3v6.75a3 3 0 0 0 3 3h10.5a3 3 0 0 0 3-3v-6.75a3 3 0 0 0-3-3v-3c0-2.9-2.35-5.25-5.25-5.25Zm3.75 8.25v-3a3.75 3.75 0 1 0-7.5 0v3h7.5Z" clip-rule="evenodd"/></svg>';

  var CH = [], CATS = [], NP = {};
  var state = { cat: "All", q: "" };
  var L = "classic";
  var root, app;

  function store(k, v) { try { if (v === undefined) return localStorage.getItem(k); localStorage.setItem(k, v); } catch (e) {} return null; }
  function sstore(k, v) { try { if (v === undefined) return sessionStorage.getItem(k); sessionStorage.setItem(k, v); } catch (e) {} return null; }
  function esc(s) { return String(s == null ? "" : s).replace(/[&<>"]/g, function (m) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[m]; }); }

  // ---------- data from the server-rendered cards ----------
  function readChannels() {
    var favs = [];
    try { favs = JSON.parse(localStorage.getItem("favoriteChannels") || "[]") || []; } catch (e) {}
    CH = [].map.call(document.querySelectorAll("#original-channels-grid a.card[data-channel-id]"), function (a) {
      var img = a.querySelector("img");
      var cid = +a.getAttribute("data-category") || 0;
      return {
        id: a.getAttribute("data-channel-id"),
        n: a.getAttribute("data-channel-name") || (img && img.alt) || "",
        logo: img ? img.getAttribute("src") : "",
        c: CAT[cid] || "Other",
        hd: a.getAttribute("data-hd") === "true",
        lk: a.getAttribute("data-requires-subscription") === "true",
        fav: favs.indexOf(a.getAttribute("data-channel-id")) >= 0
      };
    });
    CATS = [];
    if (CH.some(function (c) { return c.fav; })) CATS.push("Favourites");
    CH.forEach(function (c) { if (CATS.indexOf(c.c) < 0) CATS.push(c.c); });
  }
  function inCat(c, k) { return k === "All" || (k === "Favourites" ? c.fav : c.c === k); }
  function F() {
    var q = state.q.toLowerCase().trim();
    if (!q) return CH;
    return CH.filter(function (c) { return c.n.toLowerCase().indexOf(q) >= 0 || show(c).t.toLowerCase().indexOf(q) >= 0; });
  }
  function list() { return F().filter(function (c) { return inCat(c, state.cat); }); }
  function show(c) { return NP[c.id] || { t: "", s: 0, e: 0 }; }
  function pct(c) { var p = show(c), now = Date.now() / 1000; return p.e > p.s ? Math.max(0, Math.min(100, (now - p.s) / (p.e - p.s) * 100)) : 0; }
  function left(c) { var p = show(c); if (!p.e) return ""; var m = Math.round((p.e - Date.now() / 1000) / 60); return m > 0 ? m + " min left" : "ending"; }

  // ---------- markup ----------
  function logo(c) { return '<img class="vlogo" loading="lazy" src="' + esc(c.logo) + '" alt="">'; }
  function tags(c) { return (c.hd ? '<span class="vhd">HD</span>' : "") + (c.lk ? '<span class="vlk" title="May need a separate subscription">' + LOCK_SVG + "</span>" : ""); }
  function bar(c) { return show(c).e ? '<div class="vprog"><i style="width:' + pct(c).toFixed(0) + '%"></i></div>' : ""; }
  function tile(c, cls) { return '<a class="' + cls + '" data-f data-id="' + esc(c.id) + '" href="/play/' + esc(c.id) + '">'; }
  function cats(cls, max) {
    var h = "";
    ["All"].concat(max ? CATS.slice(0, max) : CATS).forEach(function (k) {
      h += '<span class="' + cls + (k === state.cat ? " on" : "") + '" data-f tabindex="0" data-cat="' + esc(k) + '">' + esc(k) + "</span>";
    });
    return h;
  }
  var R = {
    1: function () {
      var h = '<div class="vscroll"><div class="vwrap"><div class="vchips">' + cats("vchip") + '</div><div class="vgrid">';
      list().forEach(function (c) {
        h += tile(c, "vt") + logo(c) + '<div class="nm vell">' + esc(c.n) + tags(c) + '</div><div class="sh vell">' + esc(show(c).t) + "</div>" + bar(c) + "</a>";
      });
      return h + "</div></div></div>";
    },
    2: function () {
      var h = '<div class="vscroll"><div class="vwrap">';
      CATS.forEach(function (k) {
        var rc = F().filter(function (c) { return inCat(c, k); });
        if (!rc.length) return;
        h += "<h2>" + esc(k) + '</h2><div class="vrow">';
        rc.forEach(function (c) {
          h += tile(c, "vt") + '<div class="top">' + logo(c) + '</div><div class="b"><div class="nm vell">' + esc(c.n) + tags(c) +
            '</div><div class="sh vell">' + esc(show(c).t) + "</div>" + bar(c) + "</div></a>";
        });
        h += "</div>";
      });
      return h + "</div></div>";
    },
    3: function () {
      var h = '<div class="vscroll"><div class="vwrap">';
      F().forEach(function (c) {
        h += tile(c, "vg") + logo(c) + '<div><div class="nm vell">' + esc(c.n) + tags(c) + '</div><div class="ct">' + esc(c.c) +
          '</div></div><div><div class="sh vell">' + esc(show(c).t) + "</div>" + bar(c) + '</div><div class="tl">' + left(c) + "</div></a>";
      });
      return h + "</div></div>";
    },
    4: function () {
      var h = '<div class="vwrap"><div class="vrail">' + cats("vri") + '</div><div class="vmain vscroll"><h2>' + esc(state.cat) + '</h2><div class="vgrid">';
      list().forEach(function (c) {
        h += tile(c, "vt") + logo(c) + '<div class="nm vell">' + esc(c.n) + tags(c) + '</div><div class="sh vell">' + esc(show(c).t) + "</div></a>";
      });
      return h + "</div></div></div>";
    },
    5: function () {
      var h = '<div class="vwrap"><div class="vlist vscroll">';
      F().forEach(function (c) {
        h += tile(c, "vli") + logo(c) + '<div><div class="nm vell">' + esc(c.n) + tags(c) + '</div><div class="sh vell">' + esc(show(c).t) + "</div></div></a>";
      });
      return h + '</div><div class="vpv" id="vpv"></div></div>';
    },
    6: function () {
      var h = '<div class="vscroll"><div class="vwrap"><div class="vtabs">' + cats("vtab", 8) + "</div><h1>" +
        (state.cat === "All" ? "Live TV" : esc(state.cat)) + '</h1><div class="vgrid">';
      list().forEach(function (c) {
        h += tile(c, "vt") + '<div class="vcard">' + logo(c) + '</div><div class="nm vell">' + esc(c.n) + tags(c) + '</div><div class="sh vell">' + esc(show(c).t) + "</div>" + bar(c) + "</a>";
      });
      return h + "</div></div></div>";
    }
  };
  function byId(id) { for (var i = 0; i < CH.length; i++) if (CH[i].id === id) return CH[i]; return null; }
  function preview(id) {
    var pv = document.getElementById("vpv"), c = byId(id);
    if (!pv || !c) return;
    pv.innerHTML = logo(c) + "<h1>" + esc(c.n) + '</h1><div class="ct">' + esc(c.c) + (c.hd ? " · HD" : "") + (c.lk ? " · May need a subscription" : "") +
      '</div><div class="now">NOW PLAYING</div><div class="sh">' + esc(show(c).t || "—") + "</div>" + bar(c) + '<div class="tl">' + left(c) + '</div><div class="ok">OK to watch</div>';
  }

  // ---------- render ----------
  function place() {
    var nav = document.querySelector("nav.navbar");
    root.style.top = (nav ? Math.max(0, nav.getBoundingClientRect().bottom) : 0) + "px";
  }
  function render() {
    app.innerHTML = R[L]();
    var n = app.querySelectorAll("a[data-id]").length;
    document.getElementById("vcnt").textContent = n + " channels";
    if (!n) app.insertAdjacentHTML("afterbegin", '<div class="vnone">No channel matches “' + esc(state.q) + "”</div>");
  }
  function focusTile(id) {
    var el = (id && app.querySelector('a[data-id="' + id + '"]')) || app.querySelector("a[data-id]");
    if (el) { el.focus(); if (el.scrollIntoView) el.scrollIntoView({ block: "nearest", inline: "nearest" }); if (L === "5") preview(el.getAttribute("data-id")); }
  }
  function apply(style) {
    L = style;
    store("tvStyle", style);
    if (style === "classic") {
      document.body.classList.remove("tvl-on");
      if (root) root.style.display = "none";
      return;
    }
    if (!root) build();
    root.className = "L" + style;
    root.style.display = "";
    document.body.classList.add("tvl-on");
    place();
    render();
    focusTile(sstore("tvLastChannel"));
  }

  function build() {
    root = document.createElement("div");
    root.id = "tvl";
    root.innerHTML = '<div id="vsb"><input id="vq" data-f type="search" placeholder="Search channels (e.g. Aaj Tak, Sony, Star Sports)" autocomplete="off">' +
      '<span id="vcnt"></span><button type="button" class="vstyle-btn" data-f data-style-open>Style</button></div><div id="vapp"></div>';
    document.body.appendChild(root);
    app = document.getElementById("vapp");
    document.getElementById("vq").addEventListener("input", function (e) { state.q = e.target.value; render(); });
    window.addEventListener("resize", place);
  }

  // ---------- style popup ----------
  var pop;
  function openPicker() {
    if (!pop) {
      pop = document.createElement("div");
      pop.id = "vstyle";
      document.body.appendChild(pop);
      pop.addEventListener("click", function (e) {
        var b = e.target.closest("button[data-style]");
        if (b) { closePicker(); apply(b.getAttribute("data-style")); return; }
        if (e.target === pop) closePicker();
      });
    }
    var h = '<div class="box" role="dialog" aria-label="Channel list style"><h3>Channel list style</h3><p>Saved on this TV. Change it any time.</p>';
    STYLES.forEach(function (s) {
      h += '<button type="button" data-style="' + s[0] + '"' + (s[0] === L ? ' class="on"' : "") + ">" + esc(s[1]) + "<small>" + esc(s[2]) + "</small></button>";
    });
    pop.innerHTML = h + "</div>";
    pop.classList.add("open");
    (pop.querySelector("button.on") || pop.querySelector("button")).focus();
  }
  function closePicker() {
    if (!pop) return;
    pop.classList.remove("open");
    if (L === "classic") { var nb = document.getElementById("tv-style-nav"); if (nb) nb.focus(); }
    else focusTile(sstore("tvLastChannel"));
  }
  function pickerOpen() { return pop && pop.classList.contains("open"); }

  // ---------- D-pad navigation for the TV styles ----------
  function candidates() {
    var els = [].slice.call(root.querySelectorAll("[data-f]"));
    [].forEach.call(document.querySelectorAll("nav.navbar a[href], nav.navbar button, nav.navbar input"), function (el) {
      if (el.offsetParent !== null) els.push(el);
    });
    return els;
  }
  function move(dir) {
    var cur = document.activeElement, all = candidates();
    if (!cur || all.indexOf(cur) < 0) { focusTile(sstore("tvLastChannel")); return; }
    var a = cur.getBoundingClientRect(), ax = a.left + a.width / 2, ay = a.top + a.height / 2, best = null, bs = 1e9;
    all.forEach(function (el) {
      if (el === cur) return;
      var b = el.getBoundingClientRect();
      if (!b.width) return;
      var dx = b.left + b.width / 2 - ax, dy = b.top + b.height / 2 - ay, p, s;
      if (dir === "L") { if (b.right > a.left + 1) return; p = -dx; s = Math.abs(dy); }
      if (dir === "R") { if (b.left < a.right - 1) return; p = dx; s = Math.abs(dy); }
      if (dir === "U") { if (b.bottom > a.top + 1) return; p = -dy; s = Math.abs(dx); }
      if (dir === "D") { if (b.top < a.bottom - 1) return; p = dy; s = Math.abs(dx); }
      var sc = p + s * 3;
      if (sc < bs) { bs = sc; best = el; }
    });
    if (best) {
      try { best.focus({ preventScroll: true }); } catch (e) { best.focus(); }
      if (best.scrollIntoView) best.scrollIntoView({ block: "nearest", inline: "nearest" });
    }
  }
  function pickCat(k) {
    state.cat = k;
    render();
    var again = root.querySelector('[data-cat="' + k + '"]');
    if (again) again.focus();
  }
  function openTile(id) {
    sstore("tvLastChannel", id);
    var orig = document.querySelector('#original-channels-grid a.card[data-channel-id="' + id + '"]');
    if (orig) orig.click(); else window.location.href = "/play/" + id;
  }

  var DIRS = { ArrowLeft: "L", ArrowRight: "R", ArrowUp: "U", ArrowDown: "D", Left: "L", Right: "R", Up: "U", Down: "D" };
  var CODES = { 37: "L", 39: "R", 38: "U", 40: "D" };
  window.addEventListener("keydown", function (e) {
    var k = e.key || "", code = e.keyCode, d = DIRS[k] || CODES[code];
    var isEnter = k === "Enter" || code === 13;
    var isBack = k === "Escape" || k === "BrowserBack" || k === "GoBack" || k === "Back" || code === 10009 || code === 461;
    if (pickerOpen()) {
      if (d === "U" || d === "D") {
        e.preventDefault(); e.stopImmediatePropagation();
        var bs = [].slice.call(pop.querySelectorAll("button")), i = bs.indexOf(document.activeElement);
        bs[Math.max(0, Math.min(bs.length - 1, i + (d === "D" ? 1 : -1)))].focus();
      } else if (isBack) { e.preventDefault(); e.stopImmediatePropagation(); closePicker(); }
      return;
    }
    if (L === "classic" || !root) return;
    if (document.querySelector("dialog[open], .modal-open")) return; // login / subscription popups use the page's own keys
    var el = document.activeElement, inQ = el && el.id === "vq";
    if (inQ) {
      if (d === "L" || d === "R") return; // move the text cursor
      if (isEnter || d === "D") { e.preventDefault(); e.stopImmediatePropagation(); focusTile(); return; }
      if (isBack && state.q) { e.preventDefault(); e.stopImmediatePropagation(); el.value = ""; state.q = ""; render(); return; }
    }
    if (d) { e.preventDefault(); e.stopImmediatePropagation(); move(d); return; }
    if (isEnter && el) {
      if (el.getAttribute("data-cat")) { e.preventDefault(); e.stopImmediatePropagation(); pickCat(el.getAttribute("data-cat")); }
      else if (el.getAttribute("data-id")) { e.preventDefault(); e.stopImmediatePropagation(); openTile(el.getAttribute("data-id")); }
      else if (el.hasAttribute("data-style-open")) { e.preventDefault(); e.stopImmediatePropagation(); openPicker(); }
      return;
    }
    // Typing a letter anywhere jumps into the search box.
    if (!inQ && k.length === 1 && /[a-z0-9]/i.test(k) && !e.ctrlKey && !e.altKey && !e.metaKey) {
      document.getElementById("vq").focus();
    }
  }, true);

  document.addEventListener("click", function (e) {
    if (!root || L === "classic" || !e.target.closest) return;
    var el = e.target.closest("#tvl [data-f]");
    if (!el) return;
    if (el.getAttribute("data-id")) { e.preventDefault(); openTile(el.getAttribute("data-id")); }
    else if (el.getAttribute("data-cat")) { e.preventDefault(); pickCat(el.getAttribute("data-cat")); }
    else if (el.hasAttribute("data-style-open")) { e.preventDefault(); openPicker(); }
  });
  document.addEventListener("focusin", function (e) {
    if (!root || L === "classic") return;
    var el = e.target, id = el.getAttribute && el.getAttribute("data-id");
    if (id && L === "5") preview(id);
    if (L === "4" && el.getAttribute && el.getAttribute("data-cat") && el.getAttribute("data-cat") !== state.cat) pickCat(el.getAttribute("data-cat"));
  });

  // Now-playing data comes from nowplaying_cards.js (one /nowplaying request).
  document.addEventListener("jtv-nowplaying", function (e) {
    NP = e.detail || {};
    if (!root || L === "classic") return;
    var focused = document.activeElement && document.activeElement.getAttribute && document.activeElement.getAttribute("data-id");
    if (document.activeElement && document.activeElement.id === "vq") { render(); return; }
    render();
    if (focused) focusTile(focused);
  });

  function navButton() {
    var end = document.querySelector("nav.navbar .navbar-end");
    if (!end || document.getElementById("tv-style-nav")) return;
    var b = document.createElement("button");
    b.id = "tv-style-nav";
    b.type = "button";
    b.className = "btn btn-ghost btn-md";
    b.setAttribute("aria-label", "Change channel list style");
    b.textContent = "Style";
    b.addEventListener("click", openPicker);
    end.insertBefore(b, end.firstChild);
  }

  function start() {
    if (!document.getElementById("original-channels-grid")) return;
    readChannels();
    navButton();
    if (window.JTV_NP) NP = window.JTV_NP;
    var saved = store("tvStyle");
    apply(saved && (saved === "classic" || R[saved]) ? saved : "classic");
  }
  window.openTvStylePicker = function () { openPicker(); };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start); else start();
})();
