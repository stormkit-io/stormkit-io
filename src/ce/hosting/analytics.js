(function () {
  // Guard against a second load (managed snippet plus a user copy, or a
  // duplicate injection): without it pushState would be wrapped more than once
  // and every SPA navigation would be counted multiple times.
  if (window.__skAnalytics) {
    return;
  }

  window.__skAnalytics = true;

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

  // The server already recorded the landing, so it is the starting point for
  // de-duplicating route changes.
  var lastPath = path();

  function now() {
    return window.performance && performance.now
      ? performance.now()
      : Date.now();
  }

  function newViewID() {
    try {
      return crypto.randomUUID();
    } catch (e) {
      return Math.random().toString(36).slice(2) + Date.now().toString(36);
    }
  }

  function isVisible() {
    return document.visibilityState === "visible" && !document.prerendering;
  }

  // The page view whose foreground time is being measured. Time only counts
  // while the page is visible, and is reported when the page is hidden or
  // left, so one view can report several times under the same id.
  var view = { path: lastPath, id: newViewID(), ms: 0, since: 0 };

  function resume() {
    if (!view.since && isVisible()) {
      view.since = now();
    }
  }

  function pause() {
    if (view.since) {
      view.ms += now() - view.since;
      view.since = 0;
    }
  }

  function reportEngagement() {
    pause();

    var seconds = Math.round(view.ms / 1000);
    view.ms = 0;

    if (seconds > 0) {
      send({
        engagements: [{ path: view.path, viewId: view.id, seconds: seconds }],
      });
    }
  }

  function startView(p) {
    reportEngagement();
    view = { path: p, id: newViewID(), ms: 0, since: 0 };
    resume();
  }

  resume();

  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "hidden") {
      reportEngagement();
    } else {
      resume();
    }
  });

  window.addEventListener("pagehide", reportEngagement);

  function pageview() {
    var next = path();

    // Routers push the same path when only the query or hash changes (filters,
    // pagination), which is not a new page.
    if (next === lastPath) {
      return;
    }

    // document.referrer never changes after the landing, so a route change is
    // credited to the page the visitor navigated from.
    var referrer = location.origin + lastPath;
    lastPath = next;
    startView(next);

    send({
      pageviews: [{ path: next, referrer: referrer, requestId: rid }],
    });
  }

  // The landing never reached the server when it was prerendered, served from
  // a prefetch, or served from the browser cache (link prefetch included), so
  // the server could not count it. Record it here once the page is shown.
  function landing() {
    send({
      pageviews: [
        {
          path: path(),
          referrer: document.referrer,
          requestId: rid,
          search: location.search,
        },
      ],
    });
  }

  function missedByServer() {
    try {
      var nav = performance.getEntriesByType("navigation")[0];

      if (!nav) {
        return false;
      }

      if (nav.activationStart > 0) {
        return true;
      }

      if (typeof nav.deliveryType === "string") {
        return (
          nav.deliveryType === "cache" ||
          nav.deliveryType === "navigational-prefetch"
        );
      }

      // Browsers without deliveryType (Firefox, Safari) report a cache hit as a
      // body that arrived without any bytes on the wire.
      return nav.transferSize === 0 && nav.decodedBodySize > 0;
    } catch (e) {
      return false;
    }
  }

  if (document.prerendering) {
    document.addEventListener(
      "prerenderingchange",
      function () {
        landing();
        resume();
      },
      { once: true },
    );
  } else if (missedByServer()) {
    landing();
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
  // so we patch pushState and listen for popstate without firing on load.
  function wrap(fn) {
    return function () {
      var result = fn.apply(this, arguments);
      pageview();
      return result;
    };
  }

  // replaceState is not a page view: routers call it for URL normalization,
  // hydration keys and redirects. It still moves the visitor, so later route
  // changes are compared to (and credited to) the replaced path.
  var replaceState = history.replaceState;

  history.replaceState = function () {
    var result = replaceState.apply(this, arguments);
    lastPath = path();
    view.path = lastPath;
    return result;
  };

  history.pushState = wrap(history.pushState);
  window.addEventListener("popstate", pageview);
})();
