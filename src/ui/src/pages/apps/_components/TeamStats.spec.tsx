import type { Scope } from "nock";
import type { RenderResult } from "@testing-library/react";
import { waitFor } from "@testing-library/react";
import { expect, describe, it, beforeEach, vi } from "vitest";
import { render } from "@testing-library/react";
import TeamStats from "./TeamStats";
import {
  mockFetchTeamStats,
  mockStatsData,
} from "~/testing/nocks/nock_team_stats";

describe("~/pages/apps/_components/TeamStats.tsx", () => {
  let wrapper: RenderResult;
  let mockUseFetchTeamStats: Scope;
  const teamId = "41";

  const createWrapper = () => {
    wrapper = render(<TeamStats teamId={teamId} />);
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("loading state", () => {
    beforeEach(() => {
      mockFetchTeamStats({ teamId, response: mockStatsData });
      createWrapper();
    });

    it("should render skeleton loaders when loading", () => {
      // Should render 4 skeleton loaders (one for each stat box)
      expect(wrapper.getAllByTestId("skeleton")).toHaveLength(4);
    });
  });

  describe("error state", () => {
    beforeEach(() => {
      mockFetchTeamStats({ teamId, status: 500 });
      createWrapper();
    });

    it("should render error alert when there's an error", async () => {
      const errorMessage =
        "Something went wrong while fetching stats. Please try again later.";

      await waitFor(() => {
        expect(wrapper.getByText(errorMessage)).toBeTruthy();
      });
    });
  });

  describe("success state", () => {
    beforeEach(() => {
      mockUseFetchTeamStats = mockFetchTeamStats({
        teamId,
        response: mockStatsData,
      });
      createWrapper();
    });

    it("should render all stat boxes with correct data", async () => {
      await waitFor(() => {
        // Check total apps
        expect(wrapper.getByText("25")).toBeTruthy();
        expect(wrapper.getByText("Apps")).toBeTruthy();

        // Check total deployments
        expect(wrapper.getByText("30")).toBeTruthy();
        expect(wrapper.getByText("Deployments")).toBeTruthy();

        // Check average deployment duration
        expect(wrapper.getByText("2m 5s")).toBeTruthy();
        expect(wrapper.getByText("Avg. deployment duration")).toBeTruthy();

        // Check total requests
        expect(wrapper.getByText("10k")).toBeTruthy();
        expect(wrapper.getByText("Requests")).toBeTruthy();
      });
    });

    it("should calculate and display correct trends", async () => {
      await waitFor(() => {
        // Total apps: 5 new - 2 deleted = +3
        expect(wrapper.getByText("+3 this month")).toBeTruthy();

        // Deployments: (30-25)/25 * 100 = 20% increase
        expect(wrapper.getByText("%20 this month")).toBeTruthy();

        // Duration: (150.2-125.5)/125.5 * 100 ≈ 20% improvement (inverted calculation)
        expect(wrapper.getByText("%20 this month")).toBeTruthy();

        // Requests: (10000-8500)/8500 * 100 ≈ 18% increase
        expect(wrapper.getByText("%18 this month")).toBeTruthy();
      });
    });

    it("should make API call with correct teamId", async () => {
      await waitFor(() => {
        expect(mockUseFetchTeamStats.isDone()).toBe(true);
      });
    });
  });

  describe("formatting", () => {
    it.each`
      current          | previous         | formatted
      ${999}           | ${3}             | ${"999"}
      ${172}           | ${120}           | ${"172"}
      ${1_000_000}     | ${800_000}       | ${"1m"}
      ${2_500_000_000} | ${2_000_000_000} | ${"2.5b"}
      ${0}             | ${5}             | ${"0"}
      ${999_999_999}   | ${1_000_000_000} | ${"999.9m"}
      ${123456}        | ${654321}        | ${"123.4k"}
    `(
      "should format numbers correctly for current: $current and previous: $previous",
      async ({ current, previous, formatted }) => {
        mockFetchTeamStats({
          teamId,
          response: {
            ...mockStatsData,
            totalRequests: {
              current,
              previous,
            },
          },
        });

        createWrapper();

        await waitFor(() => {
          expect(wrapper.getByText(formatted)).toBeTruthy();
        });
      }
    );

    it.each`
      current | previous | formatted
      ${45.5} | ${50.2}  | ${"45.50s"}
      ${3665} | ${3600}  | ${"61m 5s"}
      ${180}  | ${200}   | ${"3m"}
      ${60}   | ${60}    | ${"1m"}
    `(
      "should format avgDeploymentDuration correctly for current: $current and previous: $previous",
      async ({ current, previous, formatted }) => {
        mockFetchTeamStats({
          teamId,
          response: {
            ...mockStatsData,
            avgDeploymentDuration: {
              current,
              previous,
            },
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
      const statsWithZeros = {
        totalApps: { total: 0, new: 0, deleted: 0 },
        totalDeployments: { total: 0, current: 0, previous: 0 },
        avgDeploymentDuration: { current: 0, previous: 0 },
        totalRequests: { current: 0, previous: 0 },
      };

      mockFetchTeamStats({
        teamId,
        response: statsWithZeros,
      });

      createWrapper();

      await waitFor(() => {
        expect(wrapper.getByText("No change")).toBeTruthy();
        expect(wrapper.getByText("0.00s")).toBeTruthy();
      });
    });

    it("should handle missing previous data", async () => {
      const statsWithoutPrevious = {
        totalApps: { total: 10, new: 3, deleted: 1 },
        totalDeployments: { total: 50, current: 20, previous: 0 },
        avgDeploymentDuration: { current: 120, previous: 0 },
        totalRequests: { current: 5000, previous: 0 },
      };

      mockFetchTeamStats({
        teamId,
        response: statsWithoutPrevious,
      });

      createWrapper();

      await waitFor(() => {
        expect(wrapper.getAllByText("No data to compare")).toHaveLength(3);
      });
    });
  });
});
