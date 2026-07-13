import type { RenderResult } from "@testing-library/react";
import type { Mock } from "vitest";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { waitFor, fireEvent } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import { mockDeployNow } from "~/testing/nocks/nock_deployments";
import userEvent from "@testing-library/user-event";
import DeployModal from "./DeployModal";
import { renderWithRouter } from "~/testing/helpers";

interface Props {
  app: App;
  env: Environment;
}

const navigate = vi.fn();

vi.mock("react-router", async () => {
  const mod = await vi.importActual<typeof import("react-router")>(
    "react-router"
  );

  return {
    ...mod,
    useNavigate: () => navigate,
  };
});

describe("~/layouts/AppLayout/_components/DeployModal.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let toggleModal: Mock;
  const deploymentId = "15639164571";

  const createWrapper = ({ app, env }: Props) => {
    toggleModal = vi.fn();

    wrapper = renderWithRouter({
      el: () => (
        <DeployModal app={app} selected={env} toggleModal={toggleModal} />
      ),
    });
  };

  describe("when it is a bare app", () => {
    beforeEach(async () => {
      navigate.mockClear();

      currentApp = mockApp();
      currentApp.isBare = true;
      currentEnv = mockEnv({ app: currentApp });
      currentEnv.branch = "";
      currentEnv.build.buildCmd = "";
      currentEnv.build.distFolder = "";

      createWrapper({ app: currentApp, env: currentEnv });
    });

    it("should not display the branch, build command and output folder fields", () => {
      expect(() => wrapper.getByLabelText("Checkout branch")).toThrow();
    });

    it("should upload the zip files", async () => {
      const input = wrapper.getByTestId("my-dropzone");
      const file = new File(["hello world"], "hello.zip", {
        type: "application/zip",
      });

      const scope = mockDeployNow({
        appId: currentApp.id,
        envId: currentEnv.id!,
        files: [file],
        status: 200,
        response: { id: deploymentId },
      });

      await userEvent.upload(input, file);
      await fireEvent.click(wrapper.getByText("Deploy now"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(toggleModal).toHaveBeenCalledWith(false);
        expect(navigate).toHaveBeenCalledWith(
          `/apps/${currentApp.id}/environments/${currentEnv.id}/deployments/${deploymentId}`
        );
      });
    });
  });

  describe("when it is a repository app", () => {
    beforeEach(async () => {
      navigate.mockClear();

      currentApp = mockApp();
      currentEnv = mockEnv({ app: currentApp });
      currentEnv.branch = "";
      currentEnv.build.buildCmd = "";
      currentEnv.build.distFolder = "";

      createWrapper({ app: currentApp, env: currentEnv });
    });

    afterEach(() => {
      wrapper.unmount();
    });

    const deployConfig = {
      branch: "master",
      publish: true,
    };

    const executeDeployFlow = async () => {
      await userEvent.type(
        wrapper.getByLabelText("Checkout branch"),
        deployConfig.branch
      );

      await fireEvent.click(wrapper.getByText("Deploy now"));
    };

    it("mounts the modal properly", () => {
      expect(wrapper.getByText("Start a deployment")).toBeTruthy();
      expect(wrapper.getByText("Deploy now")).toBeTruthy();
      expect(
        wrapper.getByText("Update build configuration").getAttribute("href")
      ).toBe(`/apps/${currentApp.id}/environments/${currentEnv.id}`);
    });

    it("creates a new deployment", async () => {
      const scope = mockDeployNow({
        appId: currentApp.id,
        envId: currentEnv.id!,
        config: deployConfig,
        status: 200,
        response: { id: deploymentId },
      });

      await executeDeployFlow();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(toggleModal).toHaveBeenCalledWith(false);
        expect(navigate).toHaveBeenCalledWith(
          `/apps/${currentApp.id}/environments/${currentEnv.id}/deployments/${deploymentId}`
        );
      });
    });

    it("429 errors should display a payment error", async () => {
      const scope = mockDeployNow({
        appId: currentApp.id,
        envId: currentEnv.id!,
        config: deployConfig,
        status: 429,
        response: {
          error: "You have exceeded the maximum number of concurrent builds",
        },
      });

      await executeDeployFlow();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(navigate).not.toHaveBeenCalled();
        expect(
          wrapper.getByText(
            /You have exceeded the maximum number of concurrent builds/
          )
        ).toBeTruthy();
      });
    });

    it("other errors should display a generic error", async () => {
      const scope = mockDeployNow({
        appId: currentApp.id,
        envId: currentEnv.id!,
        config: deployConfig,
        status: 400,
        response: {
          error: "Something went wrong.",
        },
      });

      await executeDeployFlow();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(navigate).not.toHaveBeenCalled();
        expect(wrapper.getByText(/Something went wrong\./)).toBeTruthy();
      });
    });
  });
});
