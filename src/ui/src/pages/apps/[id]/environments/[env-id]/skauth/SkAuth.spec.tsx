import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, afterEach, it, vi } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import nock from "nock";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { RootContext } from "~/pages/Root.context";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import * as schemaNocks from "~/testing/nocks/nock_schema";
import { mockFetchDomains } from "~/testing/nocks/nock_domains";
import SkAuth from "./SkAuth";

const { mockFetchSchema } = schemaNocks;

const apiDomain = process.env.API_DOMAIN || "";

interface ProviderData {
  status: boolean;
  clientId?: string;
  clientSecret?: string;
}

interface MockFetchProvidersProps {
  envId: string;
  response?: {
    redirectUrl: string;
    authUrl?: string;
    providers: {
      [key: string]: ProviderData;
    };
  };
  status?: number;
}

const mockFetchProviders = ({
  envId,
  status = 200,
  response = {
    redirectUrl: "/_stormkit/auth/callback",
    authUrl: "https://api.example.com/auth",
    providers: {},
  },
}: MockFetchProvidersProps) => {
  return nock(apiDomain)
    .get(`/skauth/providers?envId=${envId}`)
    .reply(status, response);
};

interface Props {
  edition?: "development" | "self-hosted" | "cloud";
  hasSchema?: boolean;
  providers?: {
    [key: string]: {
      status: boolean;
      clientId?: string;
      clientSecret?: string;
    };
  };
}

describe("~/pages/apps/[id]/environments/[env-id]/skauth/SkAuth.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;

  const createWrapper = async ({
    edition = "self-hosted",
    hasSchema = false,
    providers = {},
  }: Props = {}) => {
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    const schemaScope = mockFetchSchema({
      envId: currentEnv.id!,
      response: hasSchema
        ? {
            schema: {
              name: "test_schema",
              tables: [],
            },
          }
        : { schema: null },
      status: 200,
    });

    const providersScope = mockFetchProviders({
      envId: currentEnv.id!,
      response: {
        redirectUrl: "/_stormkit/auth/callback",
        authUrl: "https://api.example.com/auth",
        providers,
      },
    });

    // The provider drawer (ProviderSettings) mounts alongside the providers
    // list and fetches the env's verified domains to build the callback URL.
    mockFetchDomains({
      appId: currentApp.id!,
      envId: currentEnv.id!,
      verified: true,
      status: 200,
      response: { domains: [] },
    });

    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "light",
          setMode: vi.fn(),
          details: {
            stormkit: {
              apiCommit: "",
              apiVersion: "v1.0.0",
              edition,
            },
          },
        }}
      >
        <EnvironmentContext.Provider value={{ environment: currentEnv }}>
          <SkAuth />
        </EnvironmentContext.Provider>
      </RootContext.Provider>,
    );

    await waitFor(() => {
      if (edition !== "cloud") {
        expect(schemaScope.isDone()).toBe(true);

        if (hasSchema) {
          expect(providersScope.isDone()).toBe(true);
        }
      }
    });
  };

  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  describe("when is cloud edition", () => {
    beforeEach(async () => {
      await createWrapper({ edition: "cloud" });
    });

    it("should display empty page with cloud message", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
          "https://www.stormkit.io/docs/features/database",
        );

        expect(
          wrapper.getByText(
            "The auth feature is currently only available for self-hosted installations.",
          ),
        ).toBeTruthy();
      });
    });

    it("should not show configure database button", async () => {
      await waitFor(() => {
        expect(() => wrapper.getByText("Configure database")).toThrow();
      });
    });
  });

  describe("when no schema exists", () => {
    beforeEach(async () => {
      await createWrapper({ hasSchema: false });
    });

    it("should display empty page with configure database button", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText(
            "You need to attach a database to enable authentication providers.",
          ),
        ).toBeTruthy();

        expect(wrapper.getByText("Configure database")).toBeTruthy();
      });
    });

    it("should have correct link to database page", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("Configure database").getAttribute("href"),
        ).toBe(
          `/apps/${currentEnv.appId}/environments/${currentEnv.id}/database`,
        );
      });
    });
  });

  describe("when schema exists with providers", () => {
    beforeEach(async () => {
      await createWrapper({
        hasSchema: true,
        providers: {
          email: {
            status: true,
          },
          google: {
            status: true,
            clientId: "google-client-id",
            clientSecret: "google-secret",
          },
          x: {
            status: false,
            clientId: "",
            clientSecret: "",
          },
        },
      });
    });

    it("should display authentication providers", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Email")).toBeTruthy();
        expect(wrapper.getByText("Magic Link")).toBeTruthy();
        expect(wrapper.getByText("Google")).toBeTruthy();
        expect(wrapper.getByText("X / Twitter (OAuth 2.0)")).toBeTruthy();
      });
    });

    it("should display correct status for enabled provider", async () => {
      await waitFor(() => {
        expect(wrapper.getAllByText("enabled").length).toBeGreaterThan(0);
      });
    });

    it("should display correct status for disabled provider", async () => {
      await waitFor(() => {
        expect(wrapper.getAllByText("disabled").length).toBeGreaterThan(0);
      });
    });

    it("should open drawer when clicking on provider", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Google")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Google"));

      await waitFor(() => {
        expect(wrapper.getByText("Google OAuth Settings")).toBeTruthy();
      });
    });

    it("should display Authorization URL in drawer", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Google")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Google"));

      await waitFor(() => {
        expect(wrapper.getByText("Authorization URL")).toBeTruthy();
        expect(
          wrapper.getByDisplayValue("https://api.example.com/auth/google"),
        ).toBeTruthy();
        expect(
          wrapper.getByText(/redirect users here to start sign-in/),
        ).toBeTruthy();
      });
    });
  });

  describe("allowed origins hint", () => {
    const enabledProviders = { google: { status: true } };
    const hintText = /A provider is enabled but no allowed origins are set/;

    beforeEach(() => {
      localStorage.clear();
    });

    afterEach(() => {
      localStorage.clear();
    });

    it("warns when a provider is enabled but no origins are set", async () => {
      await createWrapper({ hasSchema: true, providers: enabledProviders });

      await waitFor(() => {
        expect(wrapper.getByText(hintText)).toBeTruthy();
      });
    });

    it("dismisses the hint and remembers it in localStorage", async () => {
      await createWrapper({ hasSchema: true, providers: enabledProviders });

      await waitFor(() => {
        expect(wrapper.getByText(hintText)).toBeTruthy();
      });

      fireEvent.click(wrapper.getByLabelText("Close"));

      await waitFor(() => {
        expect(wrapper.queryByText(hintText)).toBeNull();
      });

      expect(
        localStorage.getItem(`skauth_origins_hint_dismissed_${currentEnv.id}`),
      ).toBe("1");
    });

    it("stays hidden when it was dismissed previously", async () => {
      localStorage.setItem(
        `skauth_origins_hint_dismissed_${mockEnv({ app: mockApp() }).id}`,
        "1",
      );

      await createWrapper({ hasSchema: true, providers: enabledProviders });

      await waitFor(() => {
        expect(wrapper.getByText("Allowed origins")).toBeTruthy();
      });

      expect(wrapper.queryByText(hintText)).toBeNull();
    });
  });

  describe("auth config form", () => {
    beforeEach(async () => {
      await createWrapper({ hasSchema: true });
    });

    it("should explain how success callback can access skauth", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText(
            "Relative URL to redirect to after successful authentication. At this URL, browser code can read the skauth item from localStorage.",
          ),
        ).toBeTruthy();
      });
    });

    it("should save config successfully", async () => {
      const scope = nock(apiDomain)
        .post("/skauth/config", {
          envId: currentEnv.id,
          successUrl: "",
          tokenTtl: 0,
          status: true,
          allowedOrigins: [],
          oauthServerEnabled: false,
          oauthResourcePath: "",
          oauthAllowLoopback: false,
          sessionStorage: "localStorage",
          cookieDomain: "",
          loginUrl: "",
        })
        .reply(200);

      fireEvent.submit(
        wrapper.getByRole("button", { name: "Save" }).closest("form")!,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("Settings saved successfully")).toBeTruthy();
      });
    });

    it("should submit allowed origins as a trimmed list", async () => {
      const scope = nock(apiDomain)
        .post("/skauth/config", {
          envId: currentEnv.id,
          successUrl: "",
          tokenTtl: 0,
          status: true,
          allowedOrigins: [
            "https://app.example.com",
            "https://dev.example.com",
          ],
          oauthServerEnabled: false,
          oauthResourcePath: "",
          oauthAllowLoopback: false,
          sessionStorage: "localStorage",
          cookieDomain: "",
          loginUrl: "",
        })
        .reply(200);

      const input = (await wrapper.findByLabelText(
        "Allowed origins",
      )) as HTMLTextAreaElement;
      fireEvent.change(input, {
        target: {
          value: "https://app.example.com\n  https://dev.example.com  \n\n",
        },
      });

      fireEvent.submit(
        wrapper.getByRole("button", { name: "Save" }).closest("form")!,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("Settings saved successfully")).toBeTruthy();
      });
    });

    it("should submit the OAuth toggle when enabled", async () => {
      const scope = nock(apiDomain)
        .post("/skauth/config", {
          envId: currentEnv.id,
          successUrl: "",
          tokenTtl: 0,
          status: true,
          allowedOrigins: [],
          oauthServerEnabled: true,
          oauthResourcePath: "",
          oauthAllowLoopback: false,
          sessionStorage: "cookie",
          cookieDomain: "",
          loginUrl: "",
        })
        .reply(200);

      fireEvent.click(
        await wrapper.findByLabelText("Store session in a cookie"),
      );

      fireEvent.click(
        await wrapper.findByLabelText("Enable OAuth server (MCP connectors)"),
      );

      fireEvent.submit(
        wrapper.getByRole("button", { name: "Save" }).closest("form")!,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });

    it("should submit the MCP path and loopback toggle", async () => {
      const scope = nock(apiDomain)
        .post("/skauth/config", {
          envId: currentEnv.id,
          successUrl: "",
          tokenTtl: 0,
          status: true,
          allowedOrigins: [],
          oauthServerEnabled: true,
          oauthResourcePath: "/mcp",
          oauthAllowLoopback: true,
          sessionStorage: "cookie",
          cookieDomain: "",
          loginUrl: "",
        })
        .reply(200);

      fireEvent.click(
        await wrapper.findByLabelText("Store session in a cookie"),
      );

      fireEvent.click(
        await wrapper.findByLabelText("Enable OAuth server (MCP connectors)"),
      );

      fireEvent.change(await wrapper.findByLabelText("MCP server path"), {
        target: { value: "/mcp" },
      });

      fireEvent.click(
        await wrapper.findByLabelText("Allow native / CLI clients"),
      );

      fireEvent.submit(
        wrapper.getByRole("button", { name: "Save" }).closest("form")!,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });

    it("should display error when save fails", async () => {
      nock(apiDomain).post("/skauth/config").reply(500);

      fireEvent.submit(
        wrapper.getByRole("button", { name: "Save" }).closest("form")!,
      );

      await waitFor(() => {
        expect(wrapper.getByText("Failed to save settings")).toBeTruthy();
      });
    });
  });

  describe("when fetch providers fails", () => {
    beforeEach(async () => {
      currentApp = mockApp();
      currentEnv = mockEnv({ app: currentApp });

      mockFetchSchema({
        envId: currentEnv.id!,
        response: {
          schema: {
            name: "test_schema",
            tables: [],
          },
        },
        status: 200,
      });

      nock(apiDomain)
        .get(`/skauth/providers?envId=${currentEnv.id}`)
        .reply(500);

      wrapper = render(
        <RootContext.Provider
          value={{
            mode: "light",
            setMode: vi.fn(),
            details: {
              stormkit: {
                apiCommit: "",
                apiVersion: "v1.0.0",
                edition: "self-hosted",
              },
            },
          }}
        >
          <EnvironmentContext.Provider value={{ environment: currentEnv }}>
            <SkAuth />
          </EnvironmentContext.Provider>
        </RootContext.Provider>,
      );
    });

    it("should display error message", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("Failed to fetch authentication providers"),
        ).toBeTruthy();
      });
    });
  });
});
