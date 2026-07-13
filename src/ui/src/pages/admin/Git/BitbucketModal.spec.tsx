import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import BitbucketModal from "./BitbucketModal";
import { GitDetails } from "./types.d";

const apiDomain = process.env.API_DOMAIN || "";

describe("~/pages/admin/Git/BitbucketModal.tsx", () => {
  let wrapper: RenderResult;
  const mockCloseModal = vi.fn();
  const mockOnSuccess = vi.fn();

  const findClientID = () =>
    wrapper.getByLabelText(/Client ID/) as HTMLInputElement;
  const findClientSecret = () =>
    wrapper.getByLabelText(/Client secret/) as HTMLInputElement;
  const findDeployKey = () =>
    wrapper.getByLabelText(/Deploy key/) as HTMLInputElement;
  const findCancelButton = () =>
    wrapper.getByRole("button", { name: /Cancel/ }) as HTMLButtonElement;
  const findConfigureButton = () =>
    wrapper.getByRole("button", { name: /Configure/ }) as HTMLButtonElement;

  describe("without existing details", () => {
    beforeEach(() => {
      nock.cleanAll();
      vi.clearAllMocks();
      wrapper = render(
        <BitbucketModal closeModal={mockCloseModal} onSuccess={mockOnSuccess} />
      );
    });

    afterEach(() => {
      nock.cleanAll();
    });

    it("should render modal with correct title and form fields", async () => {
      await waitFor(() => {
        wrapper.getByText("Configure Bitbucket");
        wrapper.getByText("Client ID");
        wrapper.getByText("Client secret");
        wrapper.getByText("Deploy key");
        wrapper.getByRole("button", { name: "Cancel" });
        wrapper.getByRole("button", { name: "Configure" });
      });
    });
  });

  describe("with existing details", () => {
    const existingDetails: GitDetails = {
      bitbucket: {
        clientId: "existing-client-id",
        hasDeployKey: true,
        hasClientSecret: true,
      },
    };

    beforeEach(() => {
      nock.cleanAll();
      vi.clearAllMocks();
      wrapper = render(
        <BitbucketModal
          closeModal={mockCloseModal}
          onSuccess={mockOnSuccess}
          details={existingDetails}
        />
      );
    });

    afterEach(() => {
      nock.cleanAll();
    });

    it("should populate fields with existing details", () => {
      expect(findClientID().value).toBe("existing-client-id");
      expect(findClientSecret().placeholder).toBe("***************");
      expect(findDeployKey().placeholder).toBe("Enter deploy key");
    });
  });

  describe("form submission", () => {
    beforeEach(() => {
      nock.cleanAll();
      vi.clearAllMocks();
      wrapper = render(
        <BitbucketModal closeModal={mockCloseModal} onSuccess={mockOnSuccess} />
      );
    });

    afterEach(() => {
      nock.cleanAll();
    });

    it("should call closeModal when cancel button is clicked", () => {
      fireEvent.click(findCancelButton());
      expect(mockCloseModal).toHaveBeenCalledTimes(1);
    });

    it("should successfully submit configuration", async () => {
      fireEvent.change(findClientID(), { target: { value: "new-client-id" } });
      fireEvent.change(findClientSecret(), {
        target: { value: "new-client-secret" },
      });
      fireEvent.change(findDeployKey(), {
        target: { value: "new-deploy-key" },
      });

      const scope = nock(apiDomain)
        .post("/admin/git/configure", {
          provider: "bitbucket",
          clientId: "new-client-id",
          clientSecret: "new-client-secret",
          deployKey: "new-deploy-key",
        })
        .reply(200, { success: true });

      fireEvent.click(findConfigureButton());

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(mockOnSuccess).toHaveBeenCalledTimes(1);
      });
    });
  });
});
