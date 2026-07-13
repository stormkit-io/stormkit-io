import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { waitFor } from "@testing-library/react";
import bitbucketApi from "~/utils/api/Bitbucket";
import * as nocks from "~/testing/nocks/nock_bitbucket";
import NewBitbucketApp from "./NewBitbucketApp";
import { renderWithRouter } from "~/testing/helpers";

const { mockFetchRepositories } = nocks;

describe("~/pages/apps/new/github/NewBitbucketApp.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = () => {
    wrapper = renderWithRouter({ el: () => <NewBitbucketApp /> });
  };

  describe("fetching data", () => {
    const originalBaseUrl = bitbucketApi.baseurl;

    beforeEach(() => {
      bitbucketApi.accessToken = "123456";
      bitbucketApi.baseurl = process.env.API_DOMAIN || "";

      mockFetchRepositories({
        query: {
          role: "admin",
          pagelen: 100,
        },
        response: {
          next: "",
          values: [
            {
              name: "simple-project",
              full_name: "jdoe/simple-project",
              type: "repository",
            },
          ],
        },
      });

      createWrapper();
    });

    afterEach(() => {
      bitbucketApi.accessToken = "";
      bitbucketApi.baseurl = originalBaseUrl;
    });

    it("should have a proper title", () => {
      expect(wrapper.getByText("Import from Bitbucket")).toBeTruthy();
    });

    it("should render repositories", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("jdoe/simple-project")).toBeTruthy();
      });
    });
  });
});
