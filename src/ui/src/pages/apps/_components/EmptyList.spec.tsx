import type { RenderResult } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import EmptyList from "./EmptyList";
import { renderWithRouter } from "~/testing/helpers";

describe("~/pages/apps/_components/EmptyList", () => {
  let wrapper: RenderResult;

  const createWrapper = () => {
    wrapper = renderWithRouter({
      el: () => (
        <EmptyList
          primaryDesc="Lorem ipsum dolor sit amet"
          primaryActionText="Import from GitHub"
          primaryLink="/apps/new/github"
          secondaryDesc="Consectetur adipiscing elit"
          secondaryLink="/apps/new/url"
          secondaryActionText="Import from URL"
        />
      ),
    });
  };

  it("renders correctly", () => {
    createWrapper();

    expect(
      wrapper.getByText("Choose one of these options to get started")
    ).toBeTruthy();

    expect(wrapper.getByText("Lorem ipsum dolor sit amet")).toBeTruthy();
    expect(wrapper.getByText("Consectetur adipiscing elit")).toBeTruthy();

    expect(
      wrapper
        .getByRole("link", { name: "Import from URL" })
        .getAttribute("href")
    ).toBe("/apps/new/url");

    expect(
      wrapper
        .getByRole("link", { name: "Import from GitHub" })
        .getAttribute("href")
    ).toBe("/apps/new/github");

    expect(
      wrapper.getByRole("heading", { name: "Public repositories" })
    ).toBeTruthy();

    expect(
      wrapper.getByRole("heading", { name: "Public repositories" })
    ).toBeTruthy();
  });
});
