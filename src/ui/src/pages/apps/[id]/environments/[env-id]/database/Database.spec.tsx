import type { Schema } from "./actions";
import nock from "nock";
import { describe, expect, beforeEach, it } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import Database from "./Database";

interface Props {
  schema?: Schema | null;
}

describe("~/pages/apps/[id]/environments/[env-id]/database/Database.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;

  interface MockFetchSchemaProps {
    response: { schema?: Schema | null };
    status?: number;
  }

  const endpoint = process.env.API_DOMAIN || "";

  const mockFetchSchema = ({ status = 200, response }: MockFetchSchemaProps) =>
    nock(endpoint)
      .get(`/schema?envId=${currentEnv.id}`)
      .reply(status, response);

  const mockCreateSchema = (status = 200) =>
    nock(endpoint)
      .post("/schema", { appId: currentApp.id, envId: currentEnv.id })
      .reply(status, { schema: `a${currentApp.id}e${currentEnv.id}` });

  const createWrapper = async ({ schema }: Props = {}) => {
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    const scope = mockFetchSchema({
      response: { schema },
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

  describe("when no schema exists", () => {
    beforeEach(async () => {
      await createWrapper({ schema: null });
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

  describe("when a schema exists", () => {
    beforeEach(async () => {
      await createWrapper({
        schema: {
          name: "a1e1",
          tables: [
            {
              name: "users",
              rows: 100,
              size: 8192,
            },
          ],
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

    it("should not display the attach button in empty state", async () => {
      expect(() => wrapper.getByText("Attach Database")).toThrow();
    });
  });

  describe("attaching a schema", () => {
    beforeEach(async () => {
      await createWrapper({ schema: null });
    });

    it("should make POST request and refresh when attach button is clicked", async () => {
      const createScope = mockCreateSchema(200);
      const refetchScope = mockFetchSchema({
        response: {
          schema: {
            name: `a${currentApp.id}e${currentEnv.id}`,
            tables: [],
          },
        },
        status: 200,
      });

      fireEvent.click(wrapper.getByText("Attach Database"));

      await waitFor(() => {
        expect(createScope.isDone()).toBe(true);
        expect(refetchScope.isDone()).toBe(true);
      });

      await waitFor(() => {
        expect(wrapper.getByText("Schema attached successfully")).toBeTruthy();
      });
    });
  });

  describe("error handling", () => {
    it("should display generic error for unknown errors", async () => {
      const scope = mockFetchSchema({
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
