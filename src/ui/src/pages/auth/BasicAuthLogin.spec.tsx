import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, afterEach, it, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { mockAdminLogin } from "~/testing/nocks/nock_user";
import api from "~/utils/api/Api";
import BasicAuthLogin from "./BasicAuthLogin";

declare const global: {
  NavigateMock: any;
};

describe("pages/auth/BasicAuthLogin", () => {
  let wrapper: RenderResult;

  const createWrapper = () => {
    wrapper = render(<BasicAuthLogin />);
    return wrapper;
  };

  beforeEach(() => {
    createWrapper();
  });

  afterEach(() => {
    api.removeAuthToken();
  });

  it("should render the login form", () => {
    const email = wrapper.getByLabelText("Email");
    const password = wrapper.getByLabelText("Password");
    const submit = wrapper.getByRole("button", { name: "Login" });

    expect(email).toBeTruthy();
    expect(password).toBeTruthy();
    expect(submit).toBeTruthy();
  });

  it("should make a request to the api and reload the page on successfull requests", async () => {
    Object.defineProperty(window, "location", {
      value: {
        reload: vi.fn(),
      },
    });

    const email = wrapper.getByLabelText("Email");
    const password = wrapper.getByLabelText("Password");
    const submit = wrapper.getByRole("button", { name: "Login" });
    const scope = mockAdminLogin({
      payload: { email: "my-email@example.org", password: "my-password" },
      response: { sessionToken: "my-session-token" },
    });

    await userEvent.type(email, "my-email@example.org");
    await userEvent.type(password, "my-password");
    await userEvent.click(submit);

    await waitFor(() => {
      expect(scope.isDone()).toBeTruthy();
      expect(api.getAuthToken()).toBe("my-session-token");
      expect(window.location.reload).toHaveBeenCalled();
    });
  });

  it("should not submit the request if the email and/or password are not provided", async () => {
    Object.defineProperty(window, "location", {
      value: {
        reload: vi.fn(),
      },
    });

    const submit = wrapper.getByRole("button", { name: "Login" });

    await userEvent.click(submit);

    expect(submit).toBeTruthy();

    await waitFor(() => {
      wrapper.getByText("Email and password are required.");
    });

    expect(api.getAuthToken()).toBeUndefined();
    expect(window.location.reload).not.toHaveBeenCalled();
  });
});
