import { describe, it, expect, beforeEach } from "vitest";
import { render, type RenderResult } from "@testing-library/react";
import Admin from "./Admin";

describe("~/pages/admin/Admin.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    wrapper = render(<Admin />);
  });

  it("should render", () => {
    expect(wrapper.container.innerHTML).not.toBe("");
  });
});
