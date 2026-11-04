// logo_tiles.js — about one in eight Jio logos is white on a transparent
// background (made for dark screens) and disappears on the white logo tile.
// Each logo is checked once when it loads, on a tiny 24x24 canvas; mostly
// white ones get the "logo-on-dark" class, which gives them a dark tile.
// Results are kept per session so a page reload does not flash.
(function () {
  "use strict";
  var KEY = "logoLight";
  var seen = {};
  try { seen = JSON.parse(sessionStorage.getItem(KEY) || "{}") || {}; } catch (e) {}
  var cv = document.createElement("canvas");
  cv.width = cv.height = 24;
  var cx = cv.getContext && cv.getContext("2d", { willReadFrequently: true });
  var saveTimer = null;

  function save() {
    if (saveTimer) return;
    saveTimer = setTimeout(function () {
      saveTimer = null;
      try { sessionStorage.setItem(KEY, JSON.stringify(seen)); } catch (e) {}
    }, 1000);
  }

  function isLight(img) {
    if (!cx) return false;
    try {
      cx.clearRect(0, 0, 24, 24);
      cx.drawImage(img, 0, 0, 24, 24);
      var d = cx.getImageData(0, 0, 24, 24).data, shown = 0, white = 0;
      for (var i = 0; i < d.length; i += 4) {
        if (d[i + 3] < 40) continue;
        shown++;
        if (0.2126 * d[i] + 0.7152 * d[i + 1] + 0.0722 * d[i + 2] > 215) white++;
      }
      // Logos with a solid background (few transparent pixels) are fine as is.
      if (shown > 576 * 0.9) return false;
      return shown > 0 && white / shown > 0.45;
    } catch (e) {
      return false; // other-origin logo: canvas is tainted
    }
  }

  function check(img) {
    var src = img.currentSrc || img.src;
    if (!src || src.indexOf("/jtvimage/") < 0) return;
    if (!(src in seen)) { seen[src] = isLight(img) ? 1 : 0; save(); }
    if (seen[src]) img.classList.add("logo-on-dark");
    else img.classList.remove("logo-on-dark");
  }

  function early(img) {
    var src = img.src;
    if (src && seen[src]) img.classList.add("logo-on-dark");
    if (img.complete && img.naturalWidth) check(img);
  }

  // load does not bubble, but a capture listener on document sees every image,
  // including lazy ones and the tiles tv_layouts.js builds later.
  document.addEventListener("load", function (e) {
    if (e.target && e.target.tagName === "IMG") check(e.target);
  }, true);

  function scan() { Array.prototype.forEach.call(document.images, early); }
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", scan);
  else scan();
})();
