import fs from "node:fs";
import path from "node:path";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// The hosting script lives in the Go tree; it is tested here because this is
// where the JS test runner is.
const scriptPath = path.resolve(__dirname, "../../../ce/hosting/analytics.js");
const script = fs.readFileSync(scriptPath, "utf8");

interface Pageview {
  path: string;
  referrer: string;
}

describe("hosting/analytics.js", () => {
  const originalPushState = history.pushState;
  const originalReplaceState = history.replaceState;
  const originalAddEventListener = window.addEventListener;
  const listeners: [string, EventListenerOrEventListenerObject][] = [];
  let beacons: Pageview[];

  const load = () => {
    new Function(script)();
  };

  beforeEach(() => {
    beacons = [];
    history.replaceState(null, "", "/blog");

    // Track listeners the script adds so each test starts without the
    // previous test's popstate handler.
    window.addEventListener = function (
      this: Window,
      type: string,
      listener: EventListenerOrEventListenerObject,
      options?: boolean | AddEventListenerOptions,
    ) {
      listeners.push([type, listener]);
      originalAddEventListener.call(this, type, listener, options);
    } as typeof window.addEventListener;

    Object.defineProperty(navigator, "sendBeacon", {
      configurable: true,
      value: (_url: string, body: string) => {
        beacons.push(...JSON.parse(body).pageviews);
        return true;
      },
    });
  });

  afterEach(() => {
    listeners.forEach(([type, listener]) =>
      window.removeEventListener(type, listener),
    );

    listeners.length = 0;
    window.addEventListener = originalAddEventListener;
    history.pushState = originalPushState;
    history.replaceState = originalReplaceState;
    delete (window as any).__skAnalytics;
    delete (document as any).prerendering;
    vi.restoreAllMocks();
  });

  it("does not record the landing, which the server already counted", () => {
    load();
    expect(beacons).toEqual([]);
  });

  it("credits a route change to the previous page, not the landing referrer", () => {
    load();
    history.pushState(null, "", "/blog/post-1");
    history.pushState(null, "", "/about");

    expect(beacons).toEqual([
      {
        path: "/blog/post-1",
        referrer: `${location.origin}/blog`,
        requestId: "",
      },
      {
        path: "/about",
        referrer: `${location.origin}/blog/post-1`,
        requestId: "",
      },
    ]);
  });

  it("skips a route change that only changes the query or hash", () => {
    load();
    history.pushState(null, "", "/blog?page=2");
    history.pushState(null, "", "/blog#comments");

    expect(beacons).toEqual([]);
  });

  // Moves the URL the way the browser does on back/forward, without going
  // through the script's patched history methods.
  const browserNavigate = (url: string) => {
    originalReplaceState.call(history, null, "", url);
    window.dispatchEvent(new PopStateEvent("popstate"));
  };

  it("de-duplicates back and forward navigation the same way", () => {
    load();
    history.pushState(null, "", "/about");
    browserNavigate("/about?tab=team");

    expect(beacons.map(b => b.path)).toEqual(["/about"]);

    browserNavigate("/blog");

    expect(beacons).toEqual([
      { path: "/about", referrer: `${location.origin}/blog`, requestId: "" },
      { path: "/blog", referrer: `${location.origin}/about`, requestId: "" },
    ]);
  });

  it("does not count a replace navigation but credits the next one to it", () => {
    load();
    history.replaceState(null, "", "/en");
    expect(beacons).toEqual([]);

    // Returning to the page the redirect replaced is a real route change.
    history.pushState(null, "", "/blog");
    history.pushState(null, "", "/pricing");

    expect(beacons).toEqual([
      { path: "/blog", referrer: `${location.origin}/en`, requestId: "" },
      { path: "/pricing", referrer: `${location.origin}/blog`, requestId: "" },
    ]);
  });

  it("records the landing once a prerendered page is shown", () => {
    Object.defineProperty(document, "prerendering", {
      configurable: true,
      value: true,
    });

    load();
    expect(beacons).toEqual([]);

    document.dispatchEvent(new Event("prerenderingchange"));
    expect(beacons.map(b => b.path)).toEqual(["/blog"]);
  });

  const navigationEntry = (entry: Record<string, unknown>) => {
    vi.spyOn(performance, "getEntriesByType").mockReturnValue([entry as any]);
  };

  it("records the landing of a page served from a prefetch", () => {
    navigationEntry({
      deliveryType: "navigational-prefetch",
      activationStart: 0,
    });

    load();
    expect(beacons.map(b => b.path)).toEqual(["/blog"]);
  });

  it("records the landing of a page served from the browser cache", () => {
    navigationEntry({ deliveryType: "cache", activationStart: 0 });

    load();
    expect(beacons.map(b => b.path)).toEqual(["/blog"]);
  });

  it("does not record a landing that reached the server", () => {
    navigationEntry({ deliveryType: "", activationStart: 0, transferSize: 0 });

    load();
    expect(beacons).toEqual([]);
  });

  it("falls back to transfer size where deliveryType is not supported", () => {
    navigationEntry({ transferSize: 0, decodedBodySize: 2048 });

    load();
    expect(beacons.map(b => b.path)).toEqual(["/blog"]);
  });

  it("does not record a network landing where deliveryType is not supported", () => {
    navigationEntry({ transferSize: 3000, decodedBodySize: 2048 });

    load();
    expect(beacons).toEqual([]);
  });
});
