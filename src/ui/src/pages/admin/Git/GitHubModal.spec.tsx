import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import GitHubModal from "./GitHubModal";
import { GitDetails } from "./types.d";

const apiDomain = process.env.API_DOMAIN || "";

describe("~/pages/admin/Git/GitHubModal.tsx", () => {
  let wrapper: RenderResult;
  const mockCloseModal = vi.fn();
  const mockOnSuccess = vi.fn();

  const findClientSecret = () =>
    wrapper.getByLabelText(/Client secret/) as HTMLInputElement;
  const findPrivateKey = () =>
    wrapper.getByLabelText(/Private key/) as HTMLTextAreaElement;
  const findWebhookSecret = () =>
    wrapper.getByLabelText(/Webhook secret/) as HTMLInputElement;
  const findConfigureButton = () =>
    wrapper.getByRole("button", { name: /^Configure$/ }) as HTMLButtonElement;

  beforeEach(() => {
    nock.cleanAll();
    vi.clearAllMocks();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  describe("with an existing GitHub App", () => {
    const existingDetails: GitDetails = {
      github: {
        appId: "12345",
        account: "github-org",
        clientId: "existing-client-id",
        hasClientSecret: true,
        hasPrivateKey: true,
      },
    };

    beforeEach(() => {
      wrapper = render(
        <GitHubModal
          closeModal={mockCloseModal}
          onSuccess={mockOnSuccess}
          details={existingDetails}
        />
      );
    });

    it("should not require the stored client secret and private key", () => {
      expect(findClientSecret().required).toBe(false);
      expect(findPrivateKey().required).toBe(false);
      expect(findClientSecret().placeholder).toBe(
        "Leave empty to keep the current client secret"
      );
      expect(findPrivateKey().placeholder).toBe(
        "Leave empty to keep the current private key"
      );
    });

    it("should only warn about the webhook secret when none is stored", () => {
      expect(
        wrapper.queryByText(/Webhooks from GitHub are rejected/)
      ).toBeTruthy();

      wrapper.rerender(
        <GitHubModal
          closeModal={mockCloseModal}
          onSuccess={mockOnSuccess}
          details={{
            github: { ...existingDetails.github!, hasWebhookSecret: true },
          }}
        />
      );

      expect(
        wrapper.queryByText(/Webhooks from GitHub are rejected/)
      ).toBeNull();
      expect(findWebhookSecret().placeholder).toBe(
        "Leave empty to keep the current webhook secret"
      );
    });

    it("should require the secrets again when switching to another app", () => {
      fireEvent.change(wrapper.getByLabelText(/App ID/), {
        target: { value: "67890" },
      });

      expect(findClientSecret().required).toBe(true);
      expect(findPrivateKey().required).toBe(true);
      expect(findPrivateKey().placeholder).toBe(
        "The private key of your GitHub App"
      );
    });

    it("should submit only the webhook secret when the other secrets are left empty", async () => {
      fireEvent.change(findWebhookSecret(), {
        target: { value: "new-webhook-secret" },
      });

      fireEvent.change(findPrivateKey(), { target: { value: "\n" } });

      const scope = nock(apiDomain)
        .post("/admin/git/configure", {
          provider: "github",
          appId: "12345",
          account: "github-org",
          clientId: "existing-client-id",
          webhookSecret: "new-webhook-secret",
        })
        .reply(200, { ok: true });

      fireEvent.click(findConfigureButton());

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(mockOnSuccess).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe("with a GitHub App configured for the first time", () => {
    beforeEach(() => {
      wrapper = render(
        <GitHubModal closeModal={mockCloseModal} onSuccess={mockOnSuccess} />
      );

      fireEvent.click(wrapper.getByText("click here"));
    });

    it("should require the client secret and private key", () => {
      expect(findClientSecret().required).toBe(true);
      expect(findPrivateKey().required).toBe(true);
    });
  });
});
