(function () {
  var current = document.currentScript;
  var rid = (current && current.getAttribute("data-sk-rid")) || "";
  var endpoint = "/_stormkit/collect";

  function send(payload) {
    try {
      var body = JSON.stringify(payload);

      if (navigator.sendBeacon) {
        navigator.sendBeacon(endpoint, body);
      } else {
        fetch(endpoint, {
          method: "POST",
          body: body,
          keepalive: true,
          headers: { "Content-Type": "application/json" },
        });
      }
    } catch (e) {}
  }

  function path() {
    // Match the server, which records the query-less path (req.OriginalPath),
    // so client and server rows group together by request_path.
    return location.pathname;
  }

  function pageview() {
    send({
      pageviews: [{ path: path(), referrer: document.referrer, requestId: rid }],
    });
  }

  var sk = (window.stormkit = window.stormkit || {});

  sk.track = function (name, properties) {
    if (!name) {
      return;
    }

    send({
      events: [
        {
          name: String(name),
          path: path(),
          requestId: rid,
          metadata: properties || undefined,
        },
      ],
    });
  };

  // Report SPA route changes only. The initial pageview is recorded server-side,
  // so we patch pushState and listen for popstate without firing on load. We do
  // NOT wrap replaceState: routers call it for URL normalization and hydration
  // keys, which would double-count the landing page the server already recorded.
  function wrap(fn) {
    return function () {
      var result = fn.apply(this, arguments);
      pageview();
      return result;
    };
  }

  history.pushState = wrap(history.pushState);
  window.addEventListener("popstate", pageview);
})();
