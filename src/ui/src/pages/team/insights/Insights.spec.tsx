import type { Scope } from "nock";
import type { RenderResult } from "@testing-library/react";
import { waitFor } from "@testing-library/react";
import { expect, describe, it, beforeEach, vi } from "vitest";
import { render } from "@testing-library/react";
import { AuthContext } from "~/pages/auth/Auth.context";
import Insights from "./Insights";
import { mockFetchTopDomains } from "~/testing/nocks/nock_team_stats";

const mockTeam: Team = {
  id: "41",
  name: "Test Team",
  slug: "test-team",
  isDefault: true,
  currentUserRole: "owner",
};

describe("~/pages/team/insights/Insights.tsx", () => {
  let wrapper: RenderResult;
  let mockUseFetchTopDomains: Scope;
  const teamId = "41";

  const createWrapper = () => {
    wrapper = render(
      <AuthContext.Provider value={{ teams: [mockTeam] }}>
        <Insights />
      </AuthContext.Provider>
    );
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("loading state", () => {
    beforeEach(() => {
      mockFetchTopDomains({ teamId });
      createWrapper();
    });

    it("should render loading state", () => {
      expect(wrapper.getByTestId("card-loading")).toBeTruthy();
    });
  });

  describe("error state", () => {
    beforeEach(() => {
      mockFetchTopDomains({ teamId, status: 500 });
      createWrapper();
    });

    it("should render error alert when there's an error", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText(
            "Something went wrong while fetching domains. Please try again later."
          )
        ).toBeTruthy();
      });
    });
  });

  describe("empty state", () => {
    beforeEach(() => {
      mockFetchTopDomains({ teamId, response: { domains: [] } });
      createWrapper();
    });

    it("should render empty state message", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("No domains founds. Data is fetched daily.")
        ).toBeTruthy();
      });
    });
  });

  describe("success state", () => {
    beforeEach(() => {
      mockUseFetchTopDomains = mockFetchTopDomains({
        teamId,
      });

      createWrapper();
    });

    it("should render card header with correct title and subtitle", () => {
      expect(wrapper.getByText("Requests")).toBeTruthy();
      expect(
        wrapper.getByText(
          /Top domains by requests over the last 30 days period/
        )
      ).toBeTruthy();
      expect(
        wrapper.getByText(/Comparisons are made against previous 30 days/)
      ).toBeTruthy();
    });

    it("should render date range in header", async () => {
      const thirtyDaysAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000);
      const today = new Date();

      const expectedStart = thirtyDaysAgo.toLocaleDateString("en", {
        year: "numeric",
        month: "long",
        day: "2-digit",
      });
      const expectedEnd = today.toLocaleDateString("en", {
        year: "numeric",
        month: "long",
        day: "2-digit",
      });

      await waitFor(() => {
        expect(wrapper.getByText(new RegExp(expectedStart))).toBeTruthy();
        expect(wrapper.getByText(new RegExp(expectedEnd))).toBeTruthy();
      });
    });

    it("should render all domains with correct data", async () => {
      await waitFor(() => {
        // Check domain names with numbering
        expect(wrapper.getByText("1.")).toBeTruthy();
        expect(wrapper.getByText("example.com")).toBeTruthy();
        expect(wrapper.getByText("2.")).toBeTruthy();
        expect(wrapper.getByText("test.com")).toBeTruthy();
        expect(wrapper.getByText("3.")).toBeTruthy();
        expect(wrapper.getByText("demo.org")).toBeTruthy();
      });
    });

    it("should display correct request numbers", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("15k")).toBeTruthy(); // example.com current
        expect(wrapper.getByText("8.5k")).toBeTruthy(); // test.com current
        expect(wrapper.getByText("5.2k")).toBeTruthy(); // demo.org current
      });
    });

    it("should make API call with correct teamId", async () => {
      await waitFor(() => {
        expect(mockUseFetchTopDomains.isDone()).toBe(true);
      });
    });
  });

  describe("trend icons", () => {
    it("should show trending up icon for positive changes", async () => {
      mockFetchTopDomains({
        teamId,
        response: {
          domains: [
            {
              id: "1",
              domainName: "example.com",
              current: 15000,
              previous: 12000,
            },
          ],
        },
      });
      createWrapper();

      await waitFor(() => {
        expect(
          wrapper.container.querySelector('[data-testid="TrendingUpIcon"]')
        ).toBeTruthy();
      });
    });

    it("should show trending down icon for negative changes", async () => {
      mockFetchTopDomains({
        teamId,
        response: {
          domains: [
            {
              id: "1",
              domainName: "test.com",
              current: 8500,
              previous: 9000,
            },
          ],
        },
      });
      createWrapper();

      await waitFor(() => {
        expect(
          wrapper.container.querySelector('[data-testid="TrendingDownIcon"]')
        ).toBeTruthy();
      });
    });

    it("should show trending flat icon for no changes", async () => {
      mockFetchTopDomains({
        teamId,
        response: {
          domains: [
            {
              id: "1",
              domainName: "demo.org",
              current: 5200,
              previous: 5200,
            },
          ],
        },
      });
      createWrapper();

      await waitFor(() => {
        expect(
          wrapper.container.querySelector('[data-testid="TrendingFlatIcon"]')
        ).toBeTruthy();
      });
    });
  });

  describe("formatting", () => {
    it.each`
      current          | previous         | formatted
      ${999}           | ${800}           | ${"999"}
      ${1_000}         | ${900}           | ${"1k"}
      ${1_000_000}     | ${800_000}       | ${"1m"}
      ${2_500_000_000} | ${2_000_000_000} | ${"2.5b"}
      ${0}             | ${5}             | ${"0"}
      ${999_999}       | ${1_000_000}     | ${"999.9k"}
      ${123456}        | ${100000}        | ${"123.4k"}
    `(
      "should format numbers correctly for current: $current",
      async ({ current, previous, formatted }) => {
        mockFetchTopDomains({
          teamId,
          response: {
            domains: [
              {
                id: "1",
                domainName: "test.com",
                current,
                previous,
              },
            ],
          },
        });

        createWrapper();

        await waitFor(() => {
          expect(wrapper.getByText(formatted)).toBeTruthy();
        });
      }
    );
  });

  describe("edge cases", () => {
    it("should handle zero values gracefully", async () => {
      mockFetchTopDomains({
        teamId,
        response: {
          domains: [
            {
              id: "1",
              domainName: "zero.com",
              current: 0,
              previous: 0,
            },
          ],
        },
      });
      createWrapper();

      await waitFor(() => {
        expect(wrapper.getByText("0")).toBeTruthy();
        expect(wrapper.getByText("zero.com")).toBeTruthy();
      });
    });

    it("should handle large numbers correctly", async () => {
      mockFetchTopDomains({
        teamId,
        response: {
          domains: [
            {
              id: "1",
              domainName: "popular.com",
              current: 2500000,
              previous: 2000000,
            },
          ],
        },
      });
      createWrapper();

      await waitFor(() => {
        expect(wrapper.getByText("2.5m")).toBeTruthy();
        expect(wrapper.getByText("+500k")).toBeTruthy();
      });
    });

    it("should handle missing previous data", async () => {
      mockFetchTopDomains({
        teamId,
        response: {
          domains: [
            {
              id: "1",
              domainName: "new.com",
              current: 1000,
              previous: 0,
            },
          ],
        },
      });
      createWrapper();

      await waitFor(() => {
        expect(wrapper.getByText("1k")).toBeTruthy();
        expect(() => wrapper.getByText("+1k")).toThrow();
        expect(() => wrapper.getByText("-1k")).toThrow();
      });
    });
  });
});
