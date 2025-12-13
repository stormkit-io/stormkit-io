import type { Schema } from "./actions";
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
import {
  mockFetchSchema,
  mockCreateSchema,
  mockUpdateSchemaConfig,
} from "~/testing/nocks/nock_schema";
import Database from "./Database";

interface Props {
  schema?: Schema | null;
}

describe("~/pages/apps/[id]/environments/[env-id]/database/Database.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;

  const createWrapper = async ({ schema = null }: Props = {}) => {
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    const scope = mockFetchSchema({
      envId: currentEnv.id!,
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
      const createScope = mockCreateSchema({
        payload: {
          appId: currentApp.id,
          envId: currentEnv.id!,
        },
        status: 200,
      });

      const refetchScope = mockFetchSchema({
        envId: currentEnv.id!,
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
      currentApp = mockApp();
      currentEnv = mockEnv({ app: currentApp });

      const scope = mockFetchSchema({
        envId: currentEnv.id!,
        response: { schema: null },
        status: 500,
      });

      wrapper = render(
        <EnvironmentContext.Provider value={{ environment: currentEnv }}>
          <Database />
        </EnvironmentContext.Provider>
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText("Unknown error while fetching database.")
        ).toBeTruthy();
      });
    });
  });

  describe("schema migrations configuration", () => {
    describe.each`
      migrationsEnabled
      ${true}
      ${false}
    `(
      "when migrations are enabled: $migrationsEnabled",
      ({ migrationsEnabled }) => {
        beforeEach(async () => {
          await createWrapper({
            schema: {
              name: "a1e1",
              tables: [],
              migrationsEnabled,
              migrationsPath: "/db/migrations",
            },
          });
        });

        it("should display form elements appropriately", async () => {
          await waitFor(() => {
            const pathInput = wrapper.getByLabelText(
              "Migrations path"
            ) as HTMLInputElement;

            const switchInput = wrapper.getByRole("switch", {
              name: /enable schema migrations/i,
            }) as HTMLInputElement;

            expect(switchInput.checked).toBe(migrationsEnabled);
            expect(pathInput.value).toBe("/db/migrations");
            expect(wrapper.getByText("Save")).toBeTruthy();
          });
        });

        it("should update configuration when form is submitted", async () => {
          await waitFor(() => {
            expect(wrapper.getByLabelText("Migrations path")).toBeTruthy();
          });

          const pathInput = wrapper.getByLabelText(
            "Migrations path"
          ) as HTMLInputElement;

          fireEvent.change(pathInput, { target: { value: "/app/migrations" } });

          const scope = mockUpdateSchemaConfig({
            payload: {
              appId: currentApp.id,
              envId: currentEnv.id!,
              migrationsEnabled,
              migrationsPath: "/app/migrations",
            },
          });

          fireEvent.click(wrapper.getByText("Save"));

          await waitFor(() => {
            expect(scope.isDone()).toBe(true);
            expect(
              wrapper.getByText("Schema updated successfully")
            ).toBeTruthy();
          });
        });

        it("should toggle migrations when switch is clicked", async () => {
          await waitFor(() => {
            expect(
              wrapper.getByRole("switch", { name: /enable schema migrations/i })
            ).toBeTruthy();
          });

          const switchInput = wrapper.getByRole("switch", {
            name: /enable schema migrations/i,
          }) as HTMLInputElement;

          fireEvent.click(switchInput);

          const scope = mockUpdateSchemaConfig({
            payload: {
              appId: currentApp.id,
              envId: currentEnv.id!,
              migrationsEnabled: !migrationsEnabled,
              migrationsPath: "/db/migrations",
            },
          });

          fireEvent.click(wrapper.getByText("Save"));

          await waitFor(() => {
            expect(scope.isDone()).toBe(true);
            expect(
              wrapper.getByText("Schema updated successfully")
            ).toBeTruthy();
          });
        });
      }
    );

    describe("error handling for migrations configuration", () => {
      beforeEach(async () => {
        await createWrapper({
          schema: {
            name: "a1e1",
            tables: [],
            migrationsEnabled: true,
            migrationsPath: "/db/migrations",
          },
        });
      });

      it("should display error when update fails", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Save")).toBeTruthy();
        });

        const scope = mockUpdateSchemaConfig({
          payload: {
            appId: currentApp.id,
            envId: currentEnv.id!,
            migrationsEnabled: true,
            migrationsPath: "/db/migrations",
          },
          status: 500,
        });

        const saveButton = wrapper.getByText("Save");
        fireEvent.click(saveButton);

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
          expect(
            wrapper.getByText(
              "Unknown error while updating schema. Please try again."
            )
          ).toBeTruthy();
        });
      });
    });
  });
});
