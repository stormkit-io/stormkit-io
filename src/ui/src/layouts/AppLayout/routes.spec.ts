import { describe, test, expect } from "vitest";
import { routes } from "./routes";

describe("~/layouts/AppLayout/routes.ts", () => {
  test("should match routes", () => {
    // Only the paths are snapshotted: elements carry React internals that
    // change between React versions without any route change.
    expect(routes.map(route => route.path)).toMatchSnapshot();
  });
});
