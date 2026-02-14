import type { Schema } from "./actions";
import { describe, expect, beforeEach, it, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import { RootContext } from "~/pages/Root.context";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import * as nocks from "~/testing/nocks/nock_schema";
import Database from "./Database";

const {
  mockFetchSchema,
  mockCreateSchema,
  mockUpdateSchema,
  mockDeleteSchema,
} = nocks;

interface Props {
  schema?: Schema | null;
  edition?: "development" | "self-hosted" | "cloud";
}

describe("~/pages/apps/[id]/environments/[env-id]/database/Database.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;

  const createWrapper = async ({
    schema = null,
    edition = "self-hosted",
  }: Props = {}) => {
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    const scope = mockFetchSchema({
      envId: currentEnv.id!,
      response: { schema },
      status: 200,
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
          <Database />
        </EnvironmentContext.Provider>
      </RootContext.Provider>,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(edition !== "cloud");
    });
  };

  describe("when is cloud edition", () => {
    beforeEach(async () => {
      await createWrapper({ edition: "cloud", schema: null });
    });

    it("should display an empty page with a description and learn more link", async () => {
      await waitFor(() => {
        expect(() => wrapper.getByText("Attach Database")).toThrow();
        expect(
          wrapper.getByText(
            "The database feature is currently only available for self-hosted installations.",
          ),
        ).toBeTruthy();
        expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
          "https://www.stormkit.io/docs/features/database",
        );
      });
    });
  });

  describe("when no schema exists", () => {
    beforeEach(async () => {
      await createWrapper({ schema: null });
    });

    it("should display an empty page with an attach button", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Attach Database")).toBeTruthy();
        expect(
          wrapper.getByText("No database attached to this environment"),
        ).toBeTruthy();
      });
    });

    it("should display a learn more button", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
          "https://www.stormkit.io/docs/features/database",
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
          wrapper.getByText("No database attached to this environment"),
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

      const button = await waitFor(() => wrapper.getByText("Attach Database"));
      fireEvent.click(button);

      await waitFor(() => {
        expect(createScope.isDone()).toBe(true);
        expect(refetchScope.isDone()).toBe(true);
      });

      await waitFor(() => {
        expect(wrapper.getByText("Schema attached successfully")).toBeTruthy();
      });
    });

    it("should display error when attach fails", async () => {
      const createScope = mockCreateSchema({
        payload: {
          appId: currentApp.id,
          envId: currentEnv.id!,
        },
        status: 500,
      });

      const button = await waitFor(() => wrapper.getByText("Attach Database"));

      fireEvent.click(button);

      await waitFor(() => {
        expect(createScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "Unknown error while attaching schema. Please try again.",
          ),
        ).toBeTruthy();
      });
    });
  });

  describe("deleting a schema", () => {
    beforeEach(async () => {
      await createWrapper({
        schema: {
          name: "a1e1",
          tables: [],
          migrationsEnabled: false,
          migrationsFolder: "/migrations",
        },
      });
    });

    it("should close modal when cancel is clicked", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Delete")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Delete Database Schema")).toBeTruthy();
      });

      const cancelButton = wrapper.getByText("Cancel");
      fireEvent.click(cancelButton);

      await waitFor(() => {
        expect(() => wrapper.getByText("Delete Database Schema")).toThrow();
      });
    });

    it("should make DELETE request and refresh when deletion is confirmed", async () => {
      const deleteScope = mockDeleteSchema({
        payload: {
          appId: currentApp.id,
          envId: currentEnv.id!,
        },
        status: 200,
      });

      const refetchScope = mockFetchSchema({
        envId: currentEnv.id!,
        response: { schema: null },
        status: 200,
      });

      await waitFor(() => {
        expect(wrapper.getByText("Delete")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Delete Database Schema")).toBeTruthy();
        expect(
          wrapper.getByText(
            /You are about to permanently delete this environment's database schema/i,
          ),
        ).toBeTruthy();
      });

      const confirmButton = wrapper.getByText("Yes, continue");
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(deleteScope.isDone()).toBe(true);
        expect(refetchScope.isDone()).toBe(true);
      });

      await waitFor(() => {
        expect(wrapper.getByText("Schema deleted successfully")).toBeTruthy();
      });
    });

    it("should display permission error when delete returns 403", async () => {
      const deleteScope = mockDeleteSchema({
        payload: {
          appId: currentApp.id,
          envId: currentEnv.id!,
        },
        status: 403,
      });

      await waitFor(() => {
        expect(wrapper.getByText("Delete")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Delete Database Schema")).toBeTruthy();
      });

      const confirmButton = wrapper.getByText("Yes, continue");
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(deleteScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            /You don't have permission to delete this database schema/i,
          ),
        ).toBeTruthy();
      });
    });

    it("should display generic error when delete fails", async () => {
      const deleteScope = mockDeleteSchema({
        payload: {
          appId: currentApp.id,
          envId: currentEnv.id!,
        },
        status: 500,
      });

      await waitFor(() => {
        expect(wrapper.getByText("Delete")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Delete Database Schema")).toBeTruthy();
      });

      const confirmButton = wrapper.getByText("Yes, continue");
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(deleteScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "Failed to delete database schema. Please try again.",
          ),
        ).toBeTruthy();
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
        </EnvironmentContext.Provider>,
      );

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText("Unknown error while fetching database."),
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
              migrationsFolder: "/db/migrations",
            },
          });
        });

        it("should display form elements appropriately", async () => {
          await waitFor(() => {
            const pathInput = wrapper.getByLabelText(
              "Migrations path",
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
            "Migrations path",
          ) as HTMLInputElement;

          fireEvent.change(pathInput, { target: { value: "/app/migrations" } });

          const scope = mockUpdateSchema({
            payload: {
              appId: currentApp.id,
              envId: currentEnv.id!,
              migrationsEnabled,
              migrationsFolder: "/app/migrations",
              injectEnvVars: false,
            },
          });

          fireEvent.click(wrapper.getByText("Save"));

          await waitFor(() => {
            expect(scope.isDone()).toBe(true);
            expect(
              wrapper.getByText("Schema updated successfully"),
            ).toBeTruthy();
          });
        });

        it("should toggle migrations when switch is clicked", async () => {
          await waitFor(() => {
            expect(
              wrapper.getByRole("switch", {
                name: /enable schema migrations/i,
              }),
            ).toBeTruthy();
          });

          const switchInput = wrapper.getByRole("switch", {
            name: /enable schema migrations/i,
          }) as HTMLInputElement;

          fireEvent.click(switchInput);

          const scope = mockUpdateSchema({
            payload: {
              appId: currentApp.id,
              envId: currentEnv.id!,
              migrationsEnabled: !migrationsEnabled,
              migrationsFolder: "/db/migrations",
              injectEnvVars: false,
            },
          });

          fireEvent.click(wrapper.getByText("Save"));

          await waitFor(() => {
            expect(scope.isDone()).toBe(true);
            expect(
              wrapper.getByText("Schema updated successfully"),
            ).toBeTruthy();
          });
        });
      },
    );

    describe("error handling for migrations configuration", () => {
      beforeEach(async () => {
        await createWrapper({
          schema: {
            name: "a1e1",
            tables: [],
            migrationsEnabled: true,
            migrationsFolder: "/db/migrations",
          },
        });
      });

      it("should display error when update fails", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Save")).toBeTruthy();
        });

        const scope = mockUpdateSchema({
          payload: {
            appId: currentApp.id,
            envId: currentEnv.id!,
            migrationsEnabled: true,
            migrationsFolder: "/db/migrations",
            injectEnvVars: false,
          },
          status: 500,
        });

        const saveButton = wrapper.getByText("Save");
        fireEvent.click(saveButton);

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
          expect(
            wrapper.getByText(
              "Unknown error while updating schema. Please try again.",
            ),
          ).toBeTruthy();
        });
      });
    });
  });

  describe("inject environment variables", () => {
    describe.each`
      injectEnvVars
      ${true}
      ${false}
    `("when injectEnvVars is: $injectEnvVars", ({ injectEnvVars }) => {
      beforeEach(async () => {
        await createWrapper({
          schema: {
            name: "a1e1",
            tables: [],
            injectEnvVars,
            migrationsEnabled: false,
            migrationsFolder: "/migrations",
          },
        });
      });

      it("should display the switch with correct checked state", async () => {
        await waitFor(() => {
          const switchInput = wrapper.getByRole("switch", {
            name: /inject environment variables/i,
          }) as HTMLInputElement;

          expect(switchInput.checked).toBe(injectEnvVars);
        });
      });
    });

    describe("help content", () => {
      beforeEach(async () => {
        await createWrapper({
          schema: {
            name: "a1e1",
            tables: [],
            injectEnvVars: true,
            migrationsEnabled: false,
            migrationsFolder: "/migrations",
          },
        });
      });

      it("should display Learn more link in description", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Learn more.")).toBeTruthy();
        });
      });

      it("should display all environment variables in help popover", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Learn more.")).toBeTruthy();
        });

        fireEvent.click(wrapper.getByText("Learn more."));

        await waitFor(() => {
          const expectedVars = [
            "POSTGRES_HOST",
            "POSTGRES_PORT",
            "POSTGRES_DB",
            "POSTGRES_SCHEMA",
            "POSTGRES_USER",
            "POSTGRES_PASSWORD",
            "DATABASE_URL",
          ];

          expectedVars.forEach(varName => {
            expect(wrapper.getByText(`- ${varName}`)).toBeTruthy();
          });
        });
      });

      it("should display DATABASE_URL example in help popover", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("Learn more.")).toBeTruthy();
        });

        fireEvent.click(wrapper.getByText("Learn more."));

        await waitFor(() => {
          expect(
            wrapper.getByText(
              "postgresql://user:password@host:port/dbname?options=-csearch_path=schema_name",
            ),
          ).toBeTruthy();
        });
      });
    });
  });
});
