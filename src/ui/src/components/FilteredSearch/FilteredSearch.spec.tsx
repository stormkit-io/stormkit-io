import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, waitFor, type RenderResult } from "@testing-library/react";
import FilteredSearch from "./FilteredSearch";
import type { FilterDef, FilterValues } from "./types";

describe("~/components/FilteredSearch/FilteredSearch.tsx", () => {
  let wrapper: RenderResult;
  let onChange: ReturnType<typeof vi.fn>;
  let search: ReturnType<typeof vi.fn>;

  const defs = (): FilterDef[] => [
    {
      key: "appId",
      label: "App ID",
      kind: "text",
      searchHint: "Apps in your team",
      search,
    },
    { key: "path", label: "Path", kind: "text" },
    {
      key: "method",
      label: "Method",
      kind: "enum",
      options: [
        { value: "GET", text: "GET" },
        { value: "POST", text: "POST" },
      ],
      normalize: v => v.toUpperCase(),
    },
    {
      key: "isBot",
      label: "Bots",
      kind: "enum",
      options: [
        { value: "false", text: "Exclude bots" },
        { value: "true", text: "Only bots" },
      ],
      format: v => (v === "true" ? "only" : "excluded"),
    },
    { key: "from", label: "From", kind: "datetime" },
  ];

  const createWrapper = (values: FilterValues = {}) => {
    wrapper = render(
      <FilteredSearch defs={defs()} values={values} onChange={onChange} />,
    );
  };

  const input = () => wrapper.getByLabelText("Filter or search");

  beforeEach(() => {
    onChange = vi.fn();
    search = vi.fn().mockResolvedValue([
      { value: "app-1", text: "My App (app-1)" },
    ]);
  });

  it("suggests filter keys on focus and narrows them as you type", () => {
    createWrapper();
    fireEvent.focus(input());

    expect(wrapper.getByText("App ID")).toBeTruthy();
    expect(wrapper.getByText("Method")).toBeTruthy();

    fireEvent.change(input(), { target: { value: "met" } });

    expect(wrapper.queryByText("App ID")).toBe(null);
    expect(wrapper.getByText("Method")).toBeTruthy();
  });

  it("commits a two-step key then value selection", () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("Method"));

    expect(wrapper.getByTestId("token-pending").textContent).toContain(
      "Method:",
    );

    fireEvent.click(wrapper.getByText("POST"));

    expect(onChange).toHaveBeenCalledWith({ method: "POST" });
  });

  it("normalizes and commits free text on Enter over the highlighted option", () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("Method"));
    fireEvent.change(input(), { target: { value: "put" } });
    fireEvent.keyDown(input(), { key: "Enter" });

    expect(onChange).toHaveBeenCalledWith({ method: "PUT" });
  });

  it("keeps existing filters when adding another one", () => {
    createWrapper({ path: "/api" });
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("Method"));
    fireEvent.click(wrapper.getByText("GET"));

    expect(onChange).toHaveBeenCalledWith({ path: "/api", method: "GET" });
  });

  it("hides filter keys that are already applied", () => {
    createWrapper({ path: "/api" });
    fireEvent.focus(input());

    expect(wrapper.queryByText("Path")).toBe(null);
  });

  it("renders committed filters as tokens using the formatter", () => {
    createWrapper({ isBot: "true" });

    const token = wrapper.getByTestId("token-isBot");

    expect(token.textContent).toContain("Bots:");
    expect(token.textContent).toContain("only");
  });

  it("removes a filter when its token is deleted", () => {
    createWrapper({ path: "/api", method: "GET" });
    fireEvent.click(wrapper.getByLabelText("Remove Path filter"));

    expect(onChange).toHaveBeenCalledWith({ method: "GET" });
  });

  it("removes the last token on backspace with an empty input", () => {
    createWrapper({ path: "/api", method: "GET" });
    fireEvent.keyDown(input(), { key: "Backspace" });

    expect(onChange).toHaveBeenCalledWith({ path: "/api" });
  });

  it("clears the pending key on backspace before removing tokens", () => {
    createWrapper({ path: "/api" });
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("Method"));
    fireEvent.keyDown(input(), { key: "Backspace" });

    expect(wrapper.queryByTestId("token-pending")).toBe(null);
    expect(onChange).not.toHaveBeenCalled();
  });

  it("clears everything through the clear all chip", () => {
    createWrapper({ path: "/api", method: "GET" });
    fireEvent.click(wrapper.getByTestId("clear-all"));

    expect(onChange).toHaveBeenCalledWith({});
  });

  it("fetches remote suggestions for async filters", async () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("App ID"));

    await waitFor(() => {
      expect(wrapper.getByText("My App (app-1)")).toBeTruthy();
    });

    expect(wrapper.getByText("Apps in your team")).toBeTruthy();

    fireEvent.click(wrapper.getByText("My App (app-1)"));

    expect(onChange).toHaveBeenCalledWith({ appId: "app-1" });
  });

  it("still accepts a typed value when suggestions do not include it", async () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("App ID"));
    fireEvent.change(input(), { target: { value: "unlisted-app" } });
    fireEvent.keyDown(input(), { key: "Enter" });

    expect(onChange).toHaveBeenCalledWith({ appId: "unlisted-app" });
  });

  it("offers relative presets and a custom picker for datetime filters", () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("From"));

    expect(wrapper.getByText("24 hours ago")).toBeTruthy();
    expect(wrapper.getByTestId("custom-datetime")).toBeTruthy();

    fireEvent.click(wrapper.getByText("24 hours ago"));

    const value = onChange.mock.calls[0][0].from;

    // Local `datetime-local` shape, roughly 24h in the past.
    expect(value).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    expect(Date.now() - new Date(value).getTime()).toBeGreaterThan(
      23 * 60 * 60 * 1000,
    );
  });

  it("commits a custom datetime from the picker", () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("From"));
    fireEvent.change(
      wrapper.getByTestId("custom-datetime").querySelector("input")!,
      { target: { value: "2026-07-01T10:30" } },
    );

    expect(onChange).toHaveBeenCalledWith({ from: "2026-07-01T10:30" });
  });

  it("navigates suggestions with the arrow keys", () => {
    createWrapper();
    fireEvent.focus(input());
    fireEvent.click(wrapper.getByText("Method"));
    fireEvent.keyDown(input(), { key: "ArrowDown" });
    fireEvent.keyDown(input(), { key: "Enter" });

    expect(onChange).toHaveBeenCalledWith({ method: "POST" });
  });
});
