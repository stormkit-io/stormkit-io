import type { TimeSpan } from "./index.d";
import { render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { trackFetches } from "~/testing/helpers";
import {
  mockFetchEventBreakdown,
  mockFetchEventProperties,
} from "~/testing/nocks/nock_analytics";
import EventBreakdown from "./EventBreakdown";

describe("~/pages/apps/[id]/environments/[env-id]/analytics/EventBreakdown.tsx", () => {
  const note = /Breakdowns are computed from raw events/;

  const renderBreakdown = (ts: TimeSpan) => {
    const scope = mockFetchEventProperties({
      ts,
      envId: "1",
      domainId: "2",
      event: "signup",
      response: ["plan"],
    });

    mockFetchEventBreakdown({
      ts,
      envId: "1",
      domainId: "2",
      event: "signup",
      property: "plan",
      response: [],
    });

    const wrapper = render(
      <EventBreakdown
        envId="1"
        domainId="2"
        event="signup"
        ts={ts}
        onBack={vi.fn()}
      />,
    );

    return { scope, wrapper };
  };

  // Breakdowns read raw events, which outlive neither the retention window
  // nor a one-year event total.
  it("notes that a one-year breakdown is limited to raw event retention", async () => {
    const { scope, wrapper } = renderBreakdown("365d");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(note)).toBeTruthy();
    });
  });

  describe("loading states", () => {
    let fetches: ReturnType<typeof trackFetches>;

    const props = { envId: "1", domainId: "2", event: "signup" };
    const row = (name: string) => ({ name, total: 1, unique: 1 });

    const renderAt = (ts: TimeSpan) =>
      render(<EventBreakdown {...props} ts={ts} onBack={vi.fn()} />);

    beforeEach(() => {
      fetches = trackFetches();
    });

    afterEach(() => {
      fetches.restore();
    });

    it("shows a failed properties request as an error", async () => {
      mockFetchEventProperties({ ...props, ts: "30d", status: 500 });

      const wrapper = renderAt("30d");
      await fetches.settle();

      expect(
        wrapper.getByText(
          "Something went wrong while fetching event properties.",
        ),
      ).toBeTruthy();
      expect(wrapper.queryByText(/no properties to group by/)).toBeNull();
    });

    it("shows a failed breakdown request as an error", async () => {
      mockFetchEventProperties({ ...props, ts: "30d", response: ["plan"] });
      mockFetchEventBreakdown({
        ...props,
        ts: "30d",
        property: "plan",
        status: 500,
      });

      const wrapper = renderAt("30d");
      await fetches.settle();

      expect(
        wrapper.getByText(
          "Something went wrong while fetching the event breakdown.",
        ),
      ).toBeTruthy();
    });

    it("says when the property has no values", async () => {
      mockFetchEventProperties({ ...props, ts: "30d", response: ["plan"] });
      mockFetchEventBreakdown({
        ...props,
        ts: "30d",
        property: "plan",
        response: [],
      });

      const wrapper = renderAt("30d");
      await fetches.settle();

      expect(
        wrapper.getByText("No plan values in the last 30 days."),
      ).toBeTruthy();
    });

    // Another span can have other properties, and grouping by one it lacks
    // would show nothing.
    it("groups by the first property when the chosen one is gone", async () => {
      mockFetchEventProperties({ ...props, ts: "30d", response: ["plan"] });
      mockFetchEventBreakdown({
        ...props,
        ts: "30d",
        property: "plan",
        response: [row("pro")],
      });

      const wrapper = renderAt("30d");
      await fetches.settle();

      expect(wrapper.getByText("pro")).toBeTruthy();

      mockFetchEventProperties({ ...props, ts: "7d", response: ["ref"] });
      mockFetchEventBreakdown({
        ...props,
        ts: "7d",
        property: "plan",
        response: [],
      });

      const byRef = mockFetchEventBreakdown({
        ...props,
        ts: "7d",
        property: "ref",
        response: [row("mobile")],
      });

      wrapper.rerender(<EventBreakdown {...props} ts="7d" onBack={vi.fn()} />);

      await fetches.settle();

      expect(byRef.isDone()).toBe(true);
      expect(wrapper.getByText("mobile")).toBeTruthy();
      expect(wrapper.queryByText("pro")).toBeNull();
    });

    // A slow response for the previous span must not replace the current one.
    it("ignores the breakdown of a span that is no longer selected", async () => {
      mockFetchEventProperties({ ...props, ts: "30d", response: ["plan"] });

      const slow = mockFetchEventBreakdown({
        ...props,
        ts: "30d",
        property: "plan",
        response: [row("slow_row")],
        delay: 100,
      });

      const wrapper = renderAt("30d");

      await waitFor(() => {
        expect(slow.isDone()).toBe(true);
      });

      mockFetchEventProperties({ ...props, ts: "7d", response: ["plan"] });
      mockFetchEventBreakdown({
        ...props,
        ts: "7d",
        property: "plan",
        response: [row("fast_row")],
      });

      wrapper.rerender(<EventBreakdown {...props} ts="7d" onBack={vi.fn()} />);

      await fetches.settle();

      expect(wrapper.getByText("fast_row")).toBeTruthy();
      expect(wrapper.queryByText("slow_row")).toBeNull();
    });

    it("ignores the properties of a span that is no longer selected", async () => {
      mockFetchEventProperties({
        ...props,
        ts: "365d",
        response: ["old"],
        delay: 100,
      });

      const wrapper = renderAt("365d");

      mockFetchEventProperties({ ...props, ts: "7d", response: ["plan"] });
      mockFetchEventBreakdown({
        ...props,
        ts: "7d",
        property: "plan",
        response: [row("pro")],
      });

      wrapper.rerender(<EventBreakdown {...props} ts="7d" onBack={vi.fn()} />);

      await fetches.settle();

      expect(wrapper.getByText("pro")).toBeTruthy();
      expect(wrapper.getByText("plan")).toBeTruthy();
    });
  });

  it("has no note for spans within the retention window", async () => {
    const { scope, wrapper } = renderBreakdown("90d");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    expect(wrapper.queryByText(note)).toBeNull();
  });
});
