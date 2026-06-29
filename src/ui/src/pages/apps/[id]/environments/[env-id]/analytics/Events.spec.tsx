import type { TimeSpan } from "./index.d";
import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { waitFor, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import mockDomain from "~/testing/data/mock_domain";
import { mockFetchEvents } from "~/testing/nocks/nock_analytics";
import Events from "./Events";

interface WrapperProps {
  ts?: TimeSpan;
  response?: { name: string; total: number; unique: number }[];
}

describe("~/pages/apps/[id]/environments/[env-id]/analytics/Events.tsx", () => {
  let wrapper: RenderResult;
  let scope: Scope;
  let currentEnv: Environment;

  const createWrapper = ({ ts = "30d", response }: WrapperProps) => {
    const domain = mockDomain();
    currentEnv = mockEnvironments({ app: mockApp() })[0];

    scope = mockFetchEvents({
      ts,
      envId: currentEnv.id,
      domainId: domain.id,
      response,
    });

    wrapper = render(
      <Events environment={currentEnv} domain={domain} ts={ts} />,
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

  it("shows an empty state with an integration hint when there are no events", async () => {
    createWrapper({ response: [] });

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText(/No events yet/)).toBeTruthy();
      expect(wrapper.getByText(/window.stormkit.track/)).toBeTruthy();
    });
  });
});
