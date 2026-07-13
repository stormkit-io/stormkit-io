import type { RenderResult } from "@testing-library/react";
import type { Repo } from "../types.d";
import { describe, expect, it, vi } from "vitest";
import * as router from "react-router";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import * as nocks from "~/testing/nocks";
import RepoList from "./RepoList";

declare const global: {
  NavigateMock: any;
};

interface Props {
  provider?: Provider;
  loading?: boolean;
  error?: string;
  repositories?: Repo[];
  isLoadingMore?: boolean;
  hasNextPage?: boolean;
  onNextPage?: () => void;
}

describe("~/pages/apps/new/_components/RepoList.tsx", () => {
  let wrapper: RenderResult;

  const repositories: Repo[] = [
    { name: "test-repo", fullName: "namespace/test-repo" },
    { name: "test-repo-2", fullName: "namespace/test-repo-2" },
  ];

  const teams = mockTeams();

  const createWrapper = ({
    provider = "github",
    loading = false,
    error,
    repositories = [],
    isLoadingMore = false,
    hasNextPage = false,
    onNextPage = () => {},
  }: Props) => {
    const { RouterProvider, createMemoryRouter } = router;

    const memoryRouter = createMemoryRouter([
      {
        path: "*",
        element: (
          <AuthContext.Provider value={{ teams, user: mockUser() }}>
            <RepoList
              loading={loading}
              error={error}
              provider={provider}
              repositories={repositories}
              isLoadingMore={isLoadingMore}
              hasNextPage={hasNextPage}
              onNextPage={onNextPage}
            />
          </AuthContext.Provider>
        ),
      },
    ]);

    wrapper = render(<RouterProvider router={memoryRouter} />);
  };

  describe("repositories", () => {
    it("displays a list of repositories", () => {
      createWrapper({ repositories });
      expect(wrapper.getByText("test-repo")).toBeTruthy();
      expect(wrapper.getByText("test-repo-2")).toBeTruthy();
    });

    it("inserts a repository when import is clicked", async () => {
      const id = "152591";
      const scope = nocks.mockInsertApp({
        provider: "github",
        repo: "namespace/test-repo",
        id,
        teamId: teams[0].id,
      });

      createWrapper({ repositories });
      fireEvent.click(wrapper.getByLabelText("test-repo"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(global.NavigateMock).toHaveBeenCalledWith(`/apps/${id}`);
      });
    });

    it("load more", () => {
      const onNextPage = vi.fn();
      createWrapper({ repositories, hasNextPage: true, onNextPage });
      const button = wrapper.getByText("Load more");
      expect(button).toBeTruthy();
      fireEvent.click(button);
      expect(onNextPage).toHaveBeenCalled();
    });
  });
});
