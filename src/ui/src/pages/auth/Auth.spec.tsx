import { describe, expect, beforeEach, it, vi } from "vitest";
import { mockUser } from "~/testing/data";
import { waitFor, RenderResult } from "@testing-library/react";
import { AuthContext, AuthContextProps } from "./Auth.context";
import { renderWithRouter } from "~/testing/helpers";
import Auth from "./Auth";

declare const global: {
  NavigateMock: any;
};

interface Props {
  context?: AuthContextProps;
  path?: string;
}

describe("~/pages/auth/Auth.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const createWrapper = ({ context, path }: Props = {}) => {
    wrapper = renderWithRouter({
      initialEntries: [path || "/"],
      el: () => (
        <AuthContext.Provider value={{ user: mockUser(), ...context }}>
          <Auth />
        </AuthContext.Provider>
      ),
    });
  };

  describe("when user is logged in and redirect is provided", () => {
    beforeEach(() => {
      createWrapper({ path: "/?redirect=/apps" });
    });

    it("redirects user to the given page", async () => {
      await waitFor(() => {
        expect(global.NavigateMock).toHaveBeenCalledWith("/apps");
      });
    });
  });

  describe("when user is logged in and redirect is not provided", () => {
    beforeEach(() => {
      createWrapper();
    });

    it("redirects user to the home page", async () => {
      await waitFor(() => {
        expect(global.NavigateMock).toHaveBeenCalledWith("/");
      });
    });
  });

  describe("when user is not logged in and has providers ready", () => {
    let loginOauthSpy;

    beforeEach(() => {
      loginOauthSpy = vi.fn();

      createWrapper({
        context: {
          user: undefined,
          loginOauth: loginOauthSpy,
          providers: { gitlab: true, bitbucket: true, github: true },
        },
      });
    });

    it("displays some text", async () => {
      await waitFor(() => {
        expect(wrapper.getByText(/TLS/)).toBeTruthy();
        expect(wrapper.getByText(/automated TLS/)).toBeTruthy();
        expect(wrapper.getByText(/Serverless functions/)).toBeTruthy();
      });
    });

    it("displays three buttons", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("GitHub")).toBeTruthy();
        expect(wrapper.getByText("Bitbucket")).toBeTruthy();
        expect(wrapper.getByText("GitLab")).toBeTruthy();
      });
    });
  });

  describe("when user is not logged in and some providers are not configured", () => {
    beforeEach(() => {
      createWrapper({
        context: {
          user: undefined,
          loginOauth: vi.fn(),
          providers: { gitlab: true, bitbucket: false, github: false },
        },
      });
    });

    it("displays one button", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("GitLab")).toBeTruthy();
      });

      expect(() => wrapper.getByText("GitHub")).toThrow();
      expect(() => wrapper.getByText("Bitbucket")).toThrow();
    });
  });

  describe("when the user is self-hosted and is visiting for the first time", () => {
    beforeEach(() => {
      createWrapper({
        context: {
          user: undefined,
          providers: { github: false, gitlab: false, bitbucket: false },
        },
      });
    });

    // The rest is tested in BasicAuthRegister.spec.tsx
    it("displays the welcome text", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Welcome to Stormkit")).toBeTruthy();
      });
    });
  });

  describe("when the user is self-hosted and is already registered", () => {
    beforeEach(() => {
      createWrapper({
        context: {
          user: undefined,
          providers: {
            gitlab: false,
            bitbucket: false,
            github: false,
            basicAuth: "enabled",
          },
        },
      });
    });

    // The rest is tested in BasicAuthRegister.spec.tsx
    it("displays the welcome text", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("Login to your administrator account")
        ).toBeTruthy();
      });
    });
  });
});
