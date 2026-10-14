import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import ForgejoModal from "./ForgejoModal";
import { GitDetails } from "./types.d";

const apiDomain = process.env.API_DOMAIN || "";

describe("~/pages/admin/Git/ForgejoModal.tsx", () => {
  let wrapper: RenderResult;
  const mockCloseModal = vi.fn();
  const mockOnSuccess = vi.fn();

  const findInput = (label: RegExp) =>
    wrapper.getByLabelText(label) as HTMLInputElement;

  const submit = () =>
    fireEvent.click(wrapper.getByRole("button", { name: /Configure/ }));

  const createWrapper = (details?: GitDetails) => {
    wrapper = render(
      <ForgejoModal
        closeModal={mockCloseModal}
        onSuccess={mockOnSuccess}
        details={details}
      />
    );
  };

  beforeEach(() => {
    nock.cleanAll();
    vi.clearAllMocks();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it("should save the configuration", async () => {
    createWrapper();

    const scope = nock(apiDomain)
      .post("/admin/git/configure", {
        provider: "forgejo",
        baseUrl: "https://git.example.org",
        clientId: "client-id",
        clientSecret: "client-secret",
        redirectUrl: "",
      })
      .reply(200, { ok: true });

    fireEvent.change(findInput(/Forgejo URL/), {
      target: { value: "https://git.example.org" },
    });
    fireEvent.change(findInput(/Client ID/), { target: { value: "client-id" } });
    fireEvent.change(findInput(/Client Secret/), {
      target: { value: "client-secret" },
    });

    submit();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(mockOnSuccess).toHaveBeenCalledWith(
        "Forgejo configuration saved successfully"
      );
    });
  });

  it("should keep the stored secret when left blank", async () => {
    createWrapper({
      forgejo: {
        baseUrl: "https://git.example.org",
        clientId: "client-id",
        hasClientSecret: true,
        redirectUrl: "https://api.example.org/auth/forgejo/callback",
      },
    });

    expect(findInput(/Forgejo URL/).value).toBe("https://git.example.org");
    expect(findInput(/Client ID/).value).toBe("client-id");

    const scope = nock(apiDomain)
      .post("/admin/git/configure", {
        provider: "forgejo",
        baseUrl: "https://git.example.org",
        clientId: "client-id",
        clientSecret: "",
        redirectUrl: "https://api.example.org/auth/forgejo/callback",
      })
      .reply(200, { ok: true });

    submit();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("should show the error returned by the API", async () => {
    createWrapper();

    nock(apiDomain)
      .post("/admin/git/configure")
      .reply(400, { error: "Forgejo URL must be an absolute http(s) URL" });

    fireEvent.change(findInput(/Forgejo URL/), {
      target: { value: "git.example.org" },
    });
    fireEvent.change(findInput(/Client ID/), { target: { value: "client-id" } });
    fireEvent.change(findInput(/Client Secret/), {
      target: { value: "client-secret" },
    });

    submit();

    await waitFor(() => {
      expect(
        wrapper.getByText("Forgejo URL must be an absolute http(s) URL")
      ).toBeTruthy();
    });
  });
});
