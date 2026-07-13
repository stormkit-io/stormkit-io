import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import ConnectMoreRepos from "./ConnectMoreRepos";

describe("~/pages/apps/new/github/ConnectMoreRepos.tsx", () => {
  let wrapper: RenderResult;
  let setRefreshToken: Mock;
  let setInstallationId: Mock;

  const createWrapper = () => {
    wrapper = render(
      <ConnectMoreRepos
        setInstallationId={setInstallationId}
        setRefreshToken={setRefreshToken}
        openPopupURL="my-url"
      />
    );
  };

  beforeEach(() => {
    setRefreshToken = vi.fn();
    setInstallationId = vi.fn();

    Object.defineProperty(window, "location", {
      value: {
        reload: vi.fn(),
      },
    });

    Object.defineProperty(window, "open", {
      value: vi.fn(),
    });

    createWrapper();
  });

  it("should render the button", async () => {
    expect(wrapper.getByText("Connect more repositories")).toBeTruthy();
  });

  it("clicking connect more should open a popup so that the user can configure permissions", () => {
    const button = wrapper.getByText("Connect more repositories");
    fireEvent.click(button);
    expect(window.open).toHaveBeenCalledWith(
      "my-url",
      "Add repository",
      "toolbar=no,location=no,status=no,menubar=no,scrollbars=yes,resizable=yes,width=1000,height=600,left=100,top=100"
    );
  });
});
