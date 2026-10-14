import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { type RenderResult, waitFor } from "@testing-library/react";
import nock from "nock";
import { renderWithRouter } from "~/testing/helpers";
import NewForgejoApp from "./NewForgejoApp";

const apiDomain = process.env.API_DOMAIN || "";

describe("~/pages/apps/new/forgejo/NewForgejoApp.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    nock(apiDomain)
      .get("/provider/forgejo/repos?search=&page=1")
      .reply(200, {
        hasNextPage: false,
        repos: [{ name: "app", fullName: "stormkit/app" }],
      });

    wrapper = renderWithRouter({ el: () => <NewForgejoApp /> });
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it("should have a proper title", () => {
    expect(wrapper.getByText("Import from Forgejo")).toBeTruthy();
  });

  it("should render repositories fetched through the API", async () => {
    await waitFor(() => {
      expect(wrapper.getByText("app")).toBeTruthy();
    });
  });
});
