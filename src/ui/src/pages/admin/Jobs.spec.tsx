import { describe, it, expect, beforeEach } from "vitest";
import {
  fireEvent,
  render,
  waitFor,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import AdminJobs from "./Jobs";

describe("~/pages/admin/Jobs.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    wrapper = render(<AdminJobs />);
  });

  describe.each`
    job                               | index | endpoint
    ${"Sync analytics last 30 days"}  | ${0}  | ${"/admin/jobs/sync-analytics?ts=30d"}
    ${"Sync analytics last 7 days"}   | ${1}  | ${"/admin/jobs/sync-analytics?ts=7d"}
    ${"Sync analytics last 24 hours"} | ${2}  | ${"/admin/jobs/sync-analytics?ts=24h"}
    ${"Remove old artifacts"}         | ${3}  | ${"/admin/jobs/remove-old-artifacts"}
  `("Job: $job", ({ job, index, endpoint }) => {
    it("should render", () => {
      expect(wrapper.getByText(job)).toBeTruthy();
    });

    it("should fetch the correct endpoint", async () => {
      const scope = nock(process.env.API_DOMAIN || "")
        .post(endpoint)
        .reply(200);

      fireEvent.click(wrapper.getAllByText("Sync").at(index)!);

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });

    it("shows a generic success message when no deleted IDs are returned", async () => {
      nock(process.env.API_DOMAIN || "")
        .post(endpoint)
        .reply(200, {});

      fireEvent.click(wrapper.getAllByText("Sync").at(index)!);

      await waitFor(() => {
        expect(
          wrapper.getByText("Job has been successfully run.")
        ).toBeTruthy();
      });
    });
  });

  it("shows deleted count in success message when Remove old artifacts returns deleted IDs", async () => {
    nock(process.env.API_DOMAIN || "")
      .post("/admin/jobs/remove-old-artifacts")
      .reply(200, { deleted: ["1", "2", "3"] });

    fireEvent.click(wrapper.getAllByText("Sync").at(3)!);

    await waitFor(() => {
      expect(
        wrapper.getByText("Deleted 3 records successfully.")
      ).toBeTruthy();
    });
  });

  it("shows singular form when only 1 record is deleted", async () => {
    nock(process.env.API_DOMAIN || "")
      .post("/admin/jobs/remove-old-artifacts")
      .reply(200, { deleted: ["42"] });

    fireEvent.click(wrapper.getAllByText("Sync").at(3)!);

    await waitFor(() => {
      expect(
        wrapper.getByText("Deleted 1 record successfully.")
      ).toBeTruthy();
    });
  });
});
