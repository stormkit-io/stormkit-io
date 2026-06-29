import type { TimeSpan } from "./index.d";
import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { waitFor, render, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockDomain from "~/testing/data/mock_domain";
import {
  mockFetchEvents,
  mockFetchEventProperties,
  mockFetchEventBreakdown,
} from "~/testing/nocks/nock_analytics";
import Events from "./Events";

interface WrapperProps {
  ts?: TimeSpan;
  response?: { name: string; total: number; unique: number }[];
}

describe("~/pages/apps/[id]/environments/[env-id]/analytics/Events.tsx", () => {
  let wrapper: RenderResult;
  let scope: Scope;
  let currentEnv: Environment;
  let currentDomain: Domain;

  const createWrapper = ({ ts = "30d", response }: WrapperProps) => {
    currentDomain = mockDomain();
    currentEnv = mockEnvironments({ app: mockApp() })[0];

    scope = mockFetchEvents({
      ts,
      envId: currentEnv.id,
      domainId: currentDomain.id,
      response,
    });

    wrapper = render(
      <Events environment={currentEnv} domain={currentDomain} ts={ts} />,
    );
  };

  it("renders event counts from the api", async () => {
    createWrapper({
      response: [
        { name: "trip_creation", total: 1234, unique: 987 },
        { name: "product_insertion", total: 456, unique: 321 },
      ],
    });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("Events")).toBeTruthy();
      expect(wrapper.getByText("trip_creation")).toBeTruthy();
      expect(wrapper.getByText("1,234")).toBeTruthy();
      expect(wrapper.getByText(/987 unique/)).toBeTruthy();
      expect(wrapper.getByText("product_insertion")).toBeTruthy();
    });
  });

  it("shows an empty state with a link to the help drawer", async () => {
    createWrapper({ response: [] });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(/No events yet/)).toBeTruthy();
      expect(wrapper.getByText("Learn how to send events")).toBeTruthy();
    });
  });

  it("opens the help drawer with client and server examples", async () => {
    createWrapper({
      response: [{ name: "trip_creation", total: 1, unique: 1 }],
    });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    fireEvent.click(wrapper.getByText("How to track"));

    // The drawer renders in a portal, so query the whole document.
    expect(screen.getByText("Track events")).toBeTruthy();
    expect(screen.getByDisplayValue(/window.stormkit.track/)).toBeTruthy();
    expect(screen.getByDisplayValue(/_stormkit\/collect/)).toBeTruthy();
  });

  it("drills into an event and groups by a property", async () => {
    createWrapper({
      response: [{ name: "trip_creation", total: 4, unique: 2 }],
    });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    const propsScope = mockFetchEventProperties({
      ts: "30d",
      envId: currentEnv.id,
      domainId: currentDomain.id,
      event: "trip_creation",
      response: ["ref"],
    });

    const breakdownScope = mockFetchEventBreakdown({
      ts: "30d",
      envId: currentEnv.id,
      domainId: currentDomain.id,
      event: "trip_creation",
      property: "ref",
      response: [
        { name: "mobile", total: 2, unique: 1 },
        { name: "web", total: 2, unique: 1 },
      ],
    });

    fireEvent.click(wrapper.getByLabelText("Group trip_creation"));

    await waitFor(() => {
      expect(propsScope.isDone()).toBe(true);
      expect(breakdownScope.isDone()).toBe(true);
      expect(wrapper.getByText("Group by")).toBeTruthy();
      expect(wrapper.getByText("mobile")).toBeTruthy();
      expect(wrapper.getByText("web")).toBeTruthy();
    });
  });
});
