import type { TimeSpan } from "./index.d";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
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

  it("has no note for spans within the retention window", async () => {
    const { scope, wrapper } = renderBreakdown("90d");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    expect(wrapper.queryByText(note)).toBeNull();
  });
});
