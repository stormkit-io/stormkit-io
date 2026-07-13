import { describe, expect, it, beforeEach, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  RenderResult,
  waitFor,
} from "@testing-library/react";
import nock from "nock";
import { AuthContext } from "~/pages/auth/Auth.context";
import { mockUser } from "~/testing/data";
import UserButtons from "./UserButtons";

vi.mock("~/utils/storage", () => ({
  LocalStorage: { get: () => "github", set: () => "" },
}));

const endpoint = process.env.API_DOMAIN || "";

interface MockFetchChangelogProps {
  status?: number;
  response?: { markdown: string };
}

const mockFetchChangelog = ({
  status = 200,
  response = { markdown: "## March 1st, 2026\n\nSome update." },
}: MockFetchChangelogProps = {}) =>
  nock(endpoint).get("/changelog").reply(status, response);

describe("~/layouts/TopMenu/UserButtons.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ user }: { user?: User } = {}) => {
    wrapper = render(
      <AuthContext.Provider value={{ user: user || mockUser() }}>
        <UserButtons />
      </AuthContext.Provider>,
    );
  };

  describe("when user is not logged in", () => {
    it("should render nothing", () => {
      wrapper = render(
        <AuthContext.Provider value={{}}>
          <UserButtons />
        </AuthContext.Provider>,
      );

      expect(wrapper.container.firstChild).toBeNull();
    });
  });

  describe("What's New sidebar", () => {
    beforeEach(() => {
      createWrapper();
    });

    it("should render the notifications button", () => {
      expect(wrapper.getByLabelText("What's new?")).toBeTruthy();
    });

    describe("when the notifications button is clicked", () => {
      let scope: ReturnType<typeof mockFetchChangelog>;

      beforeEach(() => {
        scope = mockFetchChangelog();
        fireEvent.click(wrapper.getByLabelText("What's new?"));
      });

      it("should fetch the changelog", async () => {
        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });
      });

      it("should display the changelog content", async () => {
        await waitFor(() => {
          expect(wrapper.getByText("March 1st, 2026")).toBeTruthy();
        });
      });

      it("should rewrite relative image src to absolute URLs", async () => {
        cleanup(); // unmount the beforeEach wrapper so we can render fresh
        mockFetchChangelog({
          response: { markdown: "![alt](/assets/img.png)" },
        });
        const { getByLabelText } = render(
          <AuthContext.Provider value={{ user: mockUser() }}>
            <UserButtons />
          </AuthContext.Provider>,
        );
        fireEvent.click(getByLabelText("What's new?"));

        await waitFor(() => {
          const img = document.body.querySelector('img[src*="stormkit"]');
          expect(img?.getAttribute("src")).toBe(
            "https://www.stormkit.io/assets/img.png",
          );
        });
      });

      it("should rewrite relative href links to absolute URLs", async () => {
        cleanup(); // unmount the beforeEach wrapper so we can render fresh
        mockFetchChangelog({
          response: { markdown: "[Read docs](/docs/getting-started)" },
        });
        const { getByLabelText } = render(
          <AuthContext.Provider value={{ user: mockUser() }}>
            <UserButtons />
          </AuthContext.Provider>,
        );
        fireEvent.click(getByLabelText("What's new?"));

        await waitFor(() => {
          const link = document.body.querySelector('a[href*="stormkit"]');
          expect(link?.getAttribute("href")).toBe(
            "https://www.stormkit.io/docs/getting-started",
          );
        });
      });

      it("should not fetch the changelog again when opened a second time", async () => {
        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });

        // Close then re-open
        fireEvent.click(wrapper.getByLabelText("What's new?"));
        fireEvent.click(wrapper.getByLabelText("What's new?"));

        // Content is still present without a new network call
        expect(wrapper.getByText("March 1st, 2026")).toBeTruthy();
      });
    });

    describe("when the request fails", () => {
      it("should display a fallback error message", async () => {
        mockFetchChangelog({ status: 500, response: { markdown: "" } });
        fireEvent.click(wrapper.getByLabelText("What's new?"));

        await waitFor(() => {
          expect(wrapper.getByText("Failed to load changelog.")).toBeTruthy();
        });
      });
    });
  });
});
