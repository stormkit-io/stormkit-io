import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, afterEach, it, vi } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import nock from "nock";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { RootContext } from "~/pages/Root.context";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import * as schemaNocks from "~/testing/nocks/nock_schema";
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
    redirectUrl: "https://example.com/callback",
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
        redirectUrl: "https://example.com/callback",
        providers,
      },
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
        expect(wrapper.getByText("Google")).toBeTruthy();
        expect(wrapper.getByText("X / Twitter (OAuth 2.0)")).toBeTruthy();
      });
    });

    it("should display correct status for enabled provider", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Enabled")).toBeTruthy();
      });
    });

    it("should display correct status for disabled provider", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Disabled")).toBeTruthy();
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
