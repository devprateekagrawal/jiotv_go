// back_home.js — loaded inside the player iframes. While the video has focus
// the remote's keys go to the iframe, not the page, so the page's Back
// handler never sees them. Here a Back key leaves the player: live channels
// go to the channel list, other pages (catchup) go one step back.
(function () {
  if (window.top === window) return;
  document.addEventListener("keydown", function (e) {
    var k = e.key || "", c = e.keyCode;
    var isBack = k === "Escape" || k === "BrowserBack" || k === "GoBack" || k === "Back" || c === 10009 || c === 461;
    if (!isBack) return;
    if (document.fullscreenElement || document.webkitFullscreenElement) return; // Esc leaves fullscreen first
    e.preventDefault();
    try {
      if (window.top.location.pathname.indexOf("/play/") === 0) window.top.location.href = "/";
      else window.top.history.back();
    } catch (_) {
      window.top.location.href = "/";
    }
  });
})();
