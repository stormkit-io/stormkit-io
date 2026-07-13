import { MemoryRouter } from "react-router";
import { render, RenderResult } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockDeployments from "~/testing/data/mock_deployments_v2";
import AppChip from "./AppChip";

interface Props {
  deployment: DeploymentV2;
}

describe("~/shared/deployments/AppChip.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ deployment }: Props) => {
    wrapper = render(
      <MemoryRouter>
        <AppChip repo={deployment.repo} envName={deployment.envName} />
      </MemoryRouter>
    );
  };

  it("should display information properly", () => {
    const deployment = mockDeployments()[0];
    deployment.detailsUrl = "/my-test/url";
    createWrapper({ deployment });
    expect(wrapper.getByText(/stormkit-io\/sample-project/)).toBeTruthy();
    expect(wrapper.getByText(/production/)).toBeTruthy();
  });
});
