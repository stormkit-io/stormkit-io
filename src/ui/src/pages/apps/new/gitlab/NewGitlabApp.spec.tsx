import { describe, expect, it, afterEach, beforeEach } from "vitest";
import { type RenderResult } from "@testing-library/react";
import { waitFor } from "@testing-library/react";
import gitlabApi from "~/utils/api/Gitlab";
import * as nocks from "~/testing/nocks/nock_gitlab";
import { renderWithRouter } from "~/testing/helpers";
import NewGitLabApp from "./NewGitlabApp";

const { mockFetchRepositories } = nocks;

describe("~/pages/apps/new/github/NewGitlabApp.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = () => {
    wrapper = renderWithRouter({ el: () => <NewGitLabApp /> });
  };

  describe("fetching data", () => {
    const originalBaseUrl = gitlabApi.baseurl;

    beforeEach(() => {
      gitlabApi.accessToken = "123456";
      gitlabApi.baseurl = process.env.API_DOMAIN || "";

      mockFetchRepositories({
        query: {
          page: 1,
          per_page: 20,
        },
        response: [
          {
            name: "simple-project",
            path_with_namespace: "jdoe/simple-project",
          },
        ],
      });

      createWrapper();
    });

    afterEach(() => {
      gitlabApi.accessToken = "";
      gitlabApi.baseurl = originalBaseUrl;
    });

    it("should have a proper title", () => {
      expect(wrapper.getByText("Import from GitLab")).toBeTruthy();
    });

    it("should render repositories", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("jdoe/simple-project")).toBeTruthy();
      });
    });
  });
});
