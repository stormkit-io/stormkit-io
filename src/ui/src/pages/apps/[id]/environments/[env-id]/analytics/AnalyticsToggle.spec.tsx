import type { RenderResult } from "@testing-library/react";
import { render, fireEvent, waitFor, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  mockFetchAnalyticsStatus,
  mockEnableAnalytics,
  mockDisableAnalytics,
} from "~/testing/nocks/nock_analytics";
import AnalyticsToggle from "./AnalyticsToggle";

const appId = "100";
const envId = "200";

describe("~/pages/apps/[id]/environments/[env-id]/analytics/AnalyticsToggle.tsx", () => {
  let wrapper: RenderResult;

  const toggle = () => screen.getByRole("switch") as HTMLInputElement;

  it("reflects the disabled status from the api", async () => {
    const scope = mockFetchAnalyticsStatus({ appId, envId, enabled: false });

    wrapper = render(<AnalyticsToggle appId={appId} envId={envId} />);

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(toggle().checked).toBe(false);
    });
  });

  it("reflects the enabled status from the api", async () => {
    const scope = mockFetchAnalyticsStatus({ appId, envId, enabled: true });

    wrapper = render(<AnalyticsToggle appId={appId} envId={envId} />);

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(toggle().checked).toBe(true);
    });
  });

  it("enables tracking through the confirm modal", async () => {
    const statusScope = mockFetchAnalyticsStatus({
      appId,
      envId,
      enabled: false,
    });

    wrapper = render(<AnalyticsToggle appId={appId} envId={envId} />);

    await waitFor(() => {
      expect(statusScope.isDone()).toBe(true);
    });

    const enableScope = mockEnableAnalytics({ appId, envId });
    const refetchScope = mockFetchAnalyticsStatus({
      appId,
      envId,
      enabled: true,
    });

    fireEvent.click(toggle());

    expect(wrapper.getByText(/This will enable/)).toBeTruthy();

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(enableScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
      expect(toggle().checked).toBe(true);
    });
  });

  it("disables tracking through the confirm modal", async () => {
    const statusScope = mockFetchAnalyticsStatus({
      appId,
      envId,
      enabled: true,
    });

    wrapper = render(<AnalyticsToggle appId={appId} envId={envId} />);

    await waitFor(() => {
      expect(statusScope.isDone()).toBe(true);
    });

    const disableScope = mockDisableAnalytics({ appId, envId });
    const refetchScope = mockFetchAnalyticsStatus({
      appId,
      envId,
      enabled: false,
    });

    fireEvent.click(toggle());

    expect(wrapper.getByText(/This will disable/)).toBeTruthy();

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(disableScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
      expect(toggle().checked).toBe(false);
    });
  });
});
