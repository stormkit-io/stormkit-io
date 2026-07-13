import type { RenderResult } from "@testing-library/react";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import mockDeployments from "~/testing/data/mock_deployments_v2";
import { mockPublishDeployments } from "~/testing/nocks/nock_deployments_v2";
import PublishModal from "./PublishModal";

interface Props {
  onClose?: () => void;
  onUpdate?: () => void;
  deployment?: DeploymentV2;
}

describe("~/shared/deployments/PublishModal", () => {
  let wrapper: RenderResult;
  let currentDepl: DeploymentV2;

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
      />
    );
  };

  it("should render the title properly", () => {
    createWrapper();
    expect(wrapper.getByText("Publish deployment")).toBeTruthy();
    expect(
      wrapper.getByText(
        "A published deployment will be promoted to the environment endpoint."
      )
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

  it("publish gradually", async () => {
    const scope = mockPublishDeployments({
      appId: currentDepl.appId,
      envId: currentDepl.envId,
      publish: [{ percentage: 100, deploymentId: currentDepl.id }],
    });

    const onUpdate = vi.fn();

    createWrapper({ onUpdate });

    fireEvent.click(wrapper.getByText(/Publish to/));

    await waitFor(() => {
      expect(onUpdate).toHaveBeenCalled();
      expect(scope.isDone()).toBe(true);
    });
  });
});
