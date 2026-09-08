import type { RenderResult } from "@testing-library/react";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import mockDeployments from "~/testing/data/mock_deployments_v2";
import {
  mockFetchPublishState,
  mockPublishDeployments,
} from "~/testing/nocks/nock_deployments_v2";
import PublishModal from "./PublishModal";

// Long enough to observe the state between polls, short enough that the tests
// do not spend real seconds waiting for one.
const pollInterval = 200;

interface Props {
  onClose?: () => void;
  onUpdate?: () => void;
  deployment?: DeploymentV2;
}

describe("~/shared/deployments/PublishModal", () => {
  let wrapper: RenderResult;
  let currentDepl: DeploymentV2;

  // The modal polls on a real clock for as long as a publish takes. Faking it
  // keeps these tests instant, and keeps them from holding the event loop long
  // enough to upset timing-sensitive specs running alongside them.

  const createWrapper = ({
    onClose = vi.fn(),
    onUpdate = vi.fn(),
    deployment,
  }: Props = {}) => {
    currentDepl = deployment || mockDeployments()[0];

    wrapper = render(
      <PublishModal
        onClose={onClose}
        onUpdate={onUpdate}
        deployment={currentDepl}
        pollInterval={pollInterval}
      />,
    );
  };

  it("should render the title properly", () => {
    createWrapper();
    expect(wrapper.getByText("Publish deployment")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Stormkit starts the deployment first and moves traffic to it once it answers. Until then the current deployment keeps serving.",
      ),
    ).toBeTruthy();
  });

  it("should render the deployment properly", () => {
    createWrapper();
    expect(wrapper.getByText(/chore: update packages/)).toBeTruthy();
    expect(wrapper.getByText(/by/)).toBeTruthy();
    expect(wrapper.getByText(/Joe Doe/)).toBeTruthy();
  });

  it("should render the publish button properly", () => {
    createWrapper();
    expect(wrapper.getByText(/Publish to/)).toBeTruthy();
    expect(wrapper.getByText(/stormkit-io\/sample-project/)).toBeTruthy();
    expect(wrapper.getByText(/production/)).toBeTruthy();
  });

  it("waits for the deployment to answer before reporting success", async () => {
    createWrapper();

    const scope = mockPublishDeployments({
      appId: currentDepl.appId,
      envId: currentDepl.envId,
      publish: [{ deploymentId: currentDepl.id }],
    });

    // Registered before the click: the poll runs on a real clock, so a slow
    // machine can reach the first one before the test gets here.
    mockFetchPublishState({ deploymentId: currentDepl.id, published: true });

    fireEvent.click(wrapper.getByText(/Publish to/));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    // The publish call has returned, but the environment has not moved yet.
    expect(wrapper.getByText(/Deployment is being warmed up/)).toBeTruthy();

    await waitFor(
      () => {
        expect(
          wrapper.getByText("The environment is now serving this deployment."),
        ).toBeTruthy();
      },
      { timeout: 6000 },
    );
  }, 15000);

  // The page stops polling once a build succeeds, so the modal has to tell it
  // when the warm-up becomes visible — otherwise nothing brings the page back
  // to look again and the publishing badge never appears.
  it("tells the page when the warm-up becomes visible", async () => {
    const onUpdate = vi.fn();

    createWrapper({ onUpdate });

    mockPublishDeployments({
      appId: currentDepl.appId,
      envId: currentDepl.envId,
      publish: [{ deploymentId: currentDepl.id }],
    });

    mockFetchPublishState({ deploymentId: currentDepl.id, isWarmingUp: true });

    fireEvent.click(wrapper.getByText(/Publish to/));

    await waitFor(() => {
      // Once for the publish request, once for the warm-up becoming visible.
      expect(onUpdate).toHaveBeenCalledTimes(2);
    });
  });

  it("reports a deployment that never came up", async () => {
    createWrapper();

    const scope = mockPublishDeployments({
      appId: currentDepl.appId,
      envId: currentDepl.envId,
      publish: [{ deploymentId: currentDepl.id }],
    });

    // Warming up, then stopped without the environment moving. Both are
    // registered before the click because the poll runs on a real clock.
    mockFetchPublishState({ deploymentId: currentDepl.id, isWarmingUp: true });
    mockFetchPublishState({ deploymentId: currentDepl.id });

    fireEvent.click(wrapper.getByText(/Publish to/));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    await waitFor(
      () => {
        expect(wrapper.getByText(/did not answer/)).toBeTruthy();
      },
      { timeout: 12000 },
    );
  }, 20000);
});
