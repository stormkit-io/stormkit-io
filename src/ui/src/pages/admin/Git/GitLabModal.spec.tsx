import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import GitLabModal from "./GitLabModal";
import { GitDetails } from "./types.d";

const apiDomain = process.env.API_DOMAIN || "";

describe("~/pages/admin/Git/GitLabModal.tsx", () => {
  let wrapper: RenderResult;
  const mockCloseModal = vi.fn();
  const mockOnSuccess = vi.fn();

  const findClientID = () =>
    wrapper.getByLabelText(/Client ID/) as HTMLInputElement;
  const findClientSecret = () =>
    wrapper.getByLabelText(/Client Secret/) as HTMLInputElement;
  const findRedirectURL = () =>
    wrapper.getByLabelText(/Redirect URL/) as HTMLInputElement;
  const findCancelButton = () =>
    wrapper.getByRole("button", { name: /Cancel/ }) as HTMLButtonElement;
  const findConfigureButton = () =>
    wrapper.getByRole("button", { name: /Configure/ }) as HTMLButtonElement;

  describe("without existing details", () => {
    beforeEach(() => {
      nock.cleanAll();
      vi.clearAllMocks();
      wrapper = render(
        <GitLabModal closeModal={mockCloseModal} onSuccess={mockOnSuccess} />
      );
    });

    afterEach(() => {
      nock.cleanAll();
    });

    it("should render modal with correct title and form fields", async () => {
      await waitFor(() => {
        wrapper.getByText("Configure GitLab");
        wrapper.getByText("Client ID");
        wrapper.getByText("Client Secret");
        wrapper.getByText("Redirect URL");
        wrapper.getByRole("button", { name: "Cancel" });
        wrapper.getByRole("button", { name: "Configure" });
      });
    });
  });

  describe("with existing details", () => {
    const existingDetails: GitDetails = {
      gitlab: {
        clientId: "existing-client-id",
        hasClientSecret: true,
        redirectUrl: "https://example.com/oauth/callback",
      },
    };

    beforeEach(() => {
      nock.cleanAll();
      vi.clearAllMocks();
      wrapper = render(
        <GitLabModal
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
      expect(findRedirectURL().value).toBe(
        "https://example.com/oauth/callback"
      );
    });
  });

  describe("form submission", () => {
    beforeEach(() => {
      nock.cleanAll();
      vi.clearAllMocks();
      wrapper = render(
        <GitLabModal closeModal={mockCloseModal} onSuccess={mockOnSuccess} />
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
      fireEvent.change(findRedirectURL(), {
        target: { value: "https://example.com/callback" },
      });

      const scope = nock(apiDomain)
        .post("/admin/git/configure", {
          provider: "gitlab",
          clientId: "new-client-id",
          clientSecret: "new-client-secret",
          redirectUrl: "https://example.com/callback",
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
