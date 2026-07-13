import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, waitFor, type RenderResult } from "@testing-library/react";
import nock from "nock";
import Git from "./Git";
import { AuthContext } from "~/pages/auth/Auth.context";
import { GitDetails } from "./types.d";

// Mock react-router-dom
const mockUseSearchParams = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");

  return {
    ...actual,
    useSearchParams: () => mockUseSearchParams(),
  };
});

const apiDomain = process.env.API_DOMAIN || "";

// Mock components that we don't need to test in detail
vi.mock("./GitHubModal", () => ({
  default: () => <div data-testid="github-modal">GitHub Modal</div>,
}));

vi.mock("./GitLabModal", () => ({
  default: () => <div data-testid="gitlab-modal">GitLab Modal</div>,
}));

vi.mock("./BitbucketModal", () => ({
  default: () => <div data-testid="bitbucket-modal">Bitbucket Modal</div>,
}));

describe("~/pages/admin/Git/Git.tsx", () => {
  let wrapper: RenderResult;
  const mockLogout = vi.fn();

  const defaultAuthContext = {
    logout: mockLogout,
  };

  const defaultGitDetails: GitDetails = {
    github: {
      appId: "test-app-id",
      account: "test-account",
      clientId: "test-client-id",
      hasClientSecret: true,
      hasPrivateKey: true,
    },
    gitlab: {
      clientId: "gitlab-client-id",
      hasClientSecret: true,
      redirectUrl: "https://example.com/oauth/callback",
    },
    bitbucket: {
      clientId: "bitbucket-client-id",
      hasClientSecret: true,
      hasDeployKey: true,
    },
  };

  beforeEach(() => {
    nock.cleanAll();
    vi.clearAllMocks();

    // Default mock for useSearchParams returns no query parameters
    mockUseSearchParams.mockReturnValue([
      {
        get: vi.fn().mockReturnValue(null),
      },
      vi.fn(),
    ]);
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it("should mount and render the component", async () => {
    const scope = nock(apiDomain)
      .get("/admin/git/details")
      .reply(200, defaultGitDetails);

    // Use a simpler approach without nested BrowserRouter
    wrapper = render(
      <AuthContext.Provider value={defaultAuthContext}>
        <Git />
      </AuthContext.Provider>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("Git Provider Configuration")).toBeTruthy();
      expect(
        wrapper.getByText("Configure authentication for your Git providers")
      ).toBeTruthy();
      expect(wrapper.getByText("GitHub")).toBeTruthy();
      expect(wrapper.getByText("GitLab")).toBeTruthy();
      expect(wrapper.getByText("Bitbucket")).toBeTruthy();
    });
  });

  it("should fetch git details on mount", async () => {
    const scope = nock(apiDomain)
      .get("/admin/git/details")
      .reply(200, defaultGitDetails);

    wrapper = render(
      <AuthContext.Provider value={defaultAuthContext}>
        <Git />
      </AuthContext.Provider>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });

    // Check that check icons are displayed for configured providers
    const checkIcons = wrapper.container.querySelectorAll(
      '[data-testid="CheckIcon"]'
    );
    expect(checkIcons.length).toBe(3); // One for each provider
  });

  it("should handle git details fetch error", async () => {
    const scope = nock(apiDomain)
      .get("/admin/git/details")
      .reply(500, { error: "Internal server error" });

    wrapper = render(
      <AuthContext.Provider value={defaultAuthContext}>
        <Git />
      </AuthContext.Provider>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText("Something went wrong while fetching git details.")
      ).toBeTruthy();
    });
  });

  it("should call logout when success=github_app_created query parameter is provided", async () => {
    const scope = nock(apiDomain)
      .get("/admin/git/details")
      .reply(200, defaultGitDetails);

    // Mock useSearchParams to return success=github_app_created
    mockUseSearchParams.mockReturnValue([
      {
        get: vi.fn().mockImplementation((key: string) => {
          if (key === "success") return "github_app_created";
          return null;
        }),
      },
      vi.fn(),
    ]);

    wrapper = render(
      <AuthContext.Provider value={defaultAuthContext}>
        <Git />
      </AuthContext.Provider>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(mockLogout).toHaveBeenCalledTimes(1);
    });
  });

  it("should not call logout when success query parameter has different value", async () => {
    const scope = nock(apiDomain)
      .get("/admin/git/details")
      .reply(200, defaultGitDetails);

    // Mock useSearchParams to return success=other_value
    mockUseSearchParams.mockReturnValue([
      {
        get: vi.fn().mockImplementation((key: string) => {
          if (key === "success") return "other_value";
          return null;
        }),
      },
      vi.fn(),
    ]);

    wrapper = render(
      <AuthContext.Provider value={defaultAuthContext}>
        <Git />
      </AuthContext.Provider>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(mockLogout).not.toHaveBeenCalled();
    });
  });
});
