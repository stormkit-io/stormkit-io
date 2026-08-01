import { describe, expect, it, beforeEach } from "vitest";
import { fireEvent, waitFor, type RenderResult } from "@testing-library/react";
import nock from "nock";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockUser from "~/testing/data/mock_user";
import mockTeams from "~/testing/data/mock_teams";
import { renderWithRouter } from "~/testing/helpers";
import AccessLogs from "./AccessLogs";

describe("~/pages/admin/AccessLogs.tsx", () => {
  let wrapper: RenderResult;

  const teams = mockTeams();

  const logs = [
    {
      id: "1",
      appId: "app-1",
      domainId: "dom-1",
      hostName: "example.org",
      requestTimestamp: "1750000000",
      method: "GET",
      path: "/api/users",
      statusCode: 500,
      clientIp: "10.0.0.1",
      userAgent: "curl",
      referrer: "",
      isBot: true,
      bytesSent: 12,
      durationMs: 340,
    },
  ];

  const mockLogs = (query: string, hasNextPage = false, cursor?: string) =>
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs" + query)
      .reply(200, {
        accessLogs: logs,
        pagination: { hasNextPage, cursor },
      });

  const createWrapper = (initialPath = "/admin/access-logs") => {
    wrapper = renderWithRouter({
      path: "/admin/access-logs",
      initialEntries: [initialPath],
      el: () => (
        <AuthContext.Provider value={{ user: mockUser(), teams }}>
          <AccessLogs />
        </AuthContext.Provider>
      ),
    });
  };

  beforeEach(() => {
    nock.cleanAll();
    localStorage.clear();
  });

  it("renders access logs returned by the API", async () => {
    const scope = mockLogs("?");
    createWrapper();

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/api/users")).toBeTruthy();
      expect(wrapper.getByText("340 ms")).toBeTruthy();
    });
  });

  it("sends filters read from the URL as query params", async () => {
    const scope = mockLogs("?appId=app-1&method=GET&status=500");
    createWrapper("/admin/access-logs?appId=app-1&method=GET&status=500");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("renders URL filters as tokens", async () => {
    mockLogs("?path=%2Fapi&isBot=true");
    createWrapper("/admin/access-logs?path=/api&isBot=true");

    await waitFor(() => {
      expect(wrapper.getByTestId("token-path").textContent).toContain("/api");
      expect(wrapper.getByTestId("token-isBot").textContent).toContain("only");
    });
  });

  it("converts datetime bounds to unix seconds for the API", async () => {
    const from = "2026-07-01T10:30";
    const expected = Math.floor(new Date(from).getTime() / 1000);
    const scope = mockLogs(`?from=${expected}`);

    createWrapper(`/admin/access-logs?from=${encodeURIComponent(from)}`);

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("ignores query params that are not known filters", async () => {
    const scope = mockLogs("?path=%2Fapi");
    createWrapper("/admin/access-logs?path=/api&bogus=1");

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("persists a newly added filter to the URL and refetches", async () => {
    mockLogs("?");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("/api/users")).toBeTruthy();
    });

    const scope = mockLogs("?method=GET");

    fireEvent.focus(wrapper.getByLabelText("Filter by app, host, path, status…"));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "Method" }));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "GET" }));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByTestId("token-method")).toBeTruthy();
    });
  });

  it("appends the next page when loading more", async () => {
    mockLogs("?", true, "cursor-1");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Load more")).toBeTruthy();
    });

    const scope = nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?cursor=cursor-1")
      .reply(200, {
        accessLogs: [{ ...logs[0], id: "2", path: "/api/teams" }],
        pagination: { hasNextPage: false },
      });

    fireEvent.click(wrapper.getByText("Load more"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("/api/users")).toBeTruthy();
      expect(wrapper.getByText("/api/teams")).toBeTruthy();
    });
  });

  it("drops the cursor when filters change", async () => {
    mockLogs("?", true, "cursor-1");
    createWrapper();

    await waitFor(() => {
      expect(wrapper.getByText("Load more")).toBeTruthy();
    });

    // Adding a filter must restart from the first page, not reuse the cursor.
    const scope = mockLogs("?method=GET");

    fireEvent.focus(wrapper.getByLabelText("Filter by app, host, path, status…"));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "Method" }));
    fireEvent.click(wrapper.getByRole("menuitem", { name: "GET" }));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("shows an error when the request fails", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/access-logs?")
      .reply(500);

    createWrapper();

    await waitFor(() => {
      expect(
        wrapper.getByText("Something went wrong while fetching access logs."),
      ).toBeTruthy();
    });
  });
});
