import { buildStatusChecks } from "./helpers";

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabStatusChecks/helpers.ts", () => {
  const statusChecks = [
    {
      name: "Run e2e tests",
      cmd: "npm run e2e",
      description: "My description",
    },
    {
      name: "Run random tests",
      cmd: "npm run test:random",
      description: "My other description",
    },
  ];

  describe("buildStatusChecks", () => {
    test("should delete the given index", () => {
      expect(buildStatusChecks(statusChecks, undefined, 1)).toEqual([
        statusChecks[0],
      ]);

      expect(buildStatusChecks(statusChecks, undefined, 0)).toEqual([
        statusChecks[1],
      ]);
    });

    test("should update the given index", () => {
      const modified: StatusCheck = {
        name: "Modified name",
        cmd: "npm run modified:command",
      };

      expect(buildStatusChecks(statusChecks, modified, 0)).toEqual([
        modified,
        statusChecks[1],
      ]);

      expect(buildStatusChecks(statusChecks, modified, 1)).toEqual([
        statusChecks[0],
        modified,
      ]);
    });

    test("should append the new status check", () => {
      const created: StatusCheck = {
        name: "Created name",
        cmd: "npm run created:command",
      };

      expect(buildStatusChecks(statusChecks, created, -1)).toEqual([
        ...statusChecks,
        created,
      ]);
    });
  });
});
