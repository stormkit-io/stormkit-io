import type { Database as DB } from "./actions";
import nock from "nock";
import { describe, expect, beforeEach, it } from "vitest";
import { render, waitFor, type RenderResult } from "@testing-library/react";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import Database from "./Database";

interface Props {
  database?: DB;
}

describe("~/pages/apps/[id]/environments/[env-id]/database/Database.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;

  interface MockFetchDatabaseProps {
    response: { database?: DB };
    status?: number;
  }

  const endpoint = process.env.API_DOMAIN || "";

  const mockFetchDatabase = ({
    status = 200,
    response,
  }: MockFetchDatabaseProps) =>
    nock(endpoint)
      .get(`/database?envId=${currentEnv.id}`)
      .reply(status, response);

  const createWrapper = async ({ database }: Props = {}) => {
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    const scope = mockFetchDatabase({
      response: { database },
      status: 200,
    });

    wrapper = render(
      <EnvironmentContext.Provider value={{ environment: currentEnv }}>
        <Database />
      </EnvironmentContext.Provider>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  };

  describe("when no database is attached", () => {
    beforeEach(async () => {
      await createWrapper();
    });

    it("should display an empty page with an attach button", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Attach Database")).toBeTruthy();
        expect(
          wrapper.getByText("No database attached to this environment")
        ).toBeTruthy();
      });
    });

    it("should display a learn more button", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
          "https://www.stormkit.io/docs/features/database"
        );
      });
    });
  });

  describe("when a database is attached", () => {
    beforeEach(async () => {
      await createWrapper({
        database: {
          id: "db-123",
          name: "production-db",
          type: "external" as const,
          connectionString: "postgresql://user:pass@host:5432/dbname",
          createdAt: Date.now(),
        },
      });
    });

    it("should not display the empty page", async () => {
      await waitFor(() => {
        expect(() =>
          wrapper.getByText("No database attached to this environment")
        ).toThrow();
      });
    });

    it("should display database details placeholder", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Database details will go here")).toBeTruthy();
      });
    });

    it("should not display the attach button in empty state", async () => {
      expect(() => wrapper.getByText("Attach Database")).toThrow();
    });
  });

  describe("error handling", () => {
    it("should display generic error for unknown errors", async () => {
      const scope = mockFetchDatabase({
        response: {},
        status: 500,
      });

      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText("Unknown error while fetching database.")
        ).toBeTruthy();
      });
    });
  });
});
