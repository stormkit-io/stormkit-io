import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { render, RenderResult } from "@testing-library/react";
import mockDeployments from "~/testing/data/mock_deployments_v2";
import CommitInfo, { toHumanTime } from "./CommitInfo";

interface Props {
  deployment: DeploymentV2;
  showStatus?: boolean;
  showProject?: boolean;
  clickable?: boolean;
}

describe("~/shared/deployments/CommitInfo.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({
    deployment,
    showProject,
    clickable = true,
  }: Props) => {
    wrapper = render(
      <MemoryRouter>
        <CommitInfo
          deployment={deployment}
          showProject={showProject}
          clickable={clickable}
        />
      </MemoryRouter>
    );
  };

  it("toHumanTime", () => {
    expect(toHumanTime(60)).toBe("1m 0s");
    expect(toHumanTime(61)).toBe("1m 1s");
    expect(toHumanTime(120)).toBe("2m 0s");
    expect(toHumanTime(121)).toBe("2m 1s");
  });

  it("should display information properly", () => {
    const deployment = mockDeployments()[0];
    deployment.detailsUrl = "/my-test/url";
    createWrapper({ deployment, showProject: true });
    expect(wrapper.getByText("chore: update packages")).toBeTruthy();
    expect(wrapper.getByText("by Joe Doe")).toBeTruthy();
    expect(wrapper.getByText("published")).toBeTruthy();

    expect(
      wrapper.getByText("chore: update packages").getAttribute("href")
    ).toBe("/my-test/url");

    expect(wrapper.getByText("sample-project").getAttribute("href")).toBe(
      "/apps/1/environments/1/deployments"
    );
  });

  it("should not contain a link to deployments page when it's not clickable", () => {
    createWrapper({ deployment: mockDeployments()[0], clickable: false });

    expect(
      wrapper.getByText("chore: update packages").getAttribute("href")
    ).toBe(null);
  });

  it("should not display project when the showProject property is false", () => {
    createWrapper({ deployment: mockDeployments()[0], showProject: false });

    expect(() => wrapper.getByText("sample-project")).toThrow();
  });

  it("should display the default message when the commit message is empty", () => {
    const deployment = mockDeployments()[0];
    deployment.commit.message = "";
    deployment.status = "success";
    deployment.repo = "";

    createWrapper({ deployment });

    expect(wrapper.getByText("Uploaded zip file")).toBeTruthy();
  });
});
