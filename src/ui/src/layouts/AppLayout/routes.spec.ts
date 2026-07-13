import { describe, test, expect } from "vitest";
import { routes } from "./routes";

describe("~/layouts/AppLayout/routes.ts", () => {
  test("should match routes", () => {
    expect(routes).toMatchSnapshot();
  });
});
