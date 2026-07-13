import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, beforeEach } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import ConnectedAccounts from "./ConnectedAccounts";
import { mockUseFetchEmails } from "~/testing/nocks/nock_user";

interface Props {
  accounts?: Array<ConnectedAccount>;
}

describe("~/pages/user/account/_components/ConnectedAccounts", () => {
  let wrapper: RenderResult;

  const createWrapper = async ({ accounts = [] }: Props) => {
    const scope = mockUseFetchEmails({
      response: {
        emails: [
          { address: "hello@example.org", verified: true, primary: true },
          { address: "hi@example.org", verified: true, primary: false },
        ],
      },
    });

    wrapper = render(
      <MemoryRouter>
        <ConnectedAccounts accounts={accounts} />
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  };

  describe("emails", () => {
    beforeEach(async () => {
      await createWrapper({
        accounts: [
          { provider: "gitlab", hasPersonalAccessToken: true, displayName: "" },
        ],
      });
    });

    it("should list the email addresses", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("hello@example.org")).toBeTruthy();
        expect(wrapper.getByText("hi@example.org")).toBeTruthy();
      });
    });

    it("should mark the Primary email", async () => {
      await waitFor(() => {
        expect(wrapper.getByTestId("hello@example.org").innerHTML).toContain(
          "Primary"
        );

        expect(wrapper.getByTestId("hi@example.org").innerHTML).not.toContain(
          "Primary"
        );
      });
    });
  });

  describe.each`
    provider       | human          | hasPersonalAccessTokenSupport
    ${"github"}    | ${"GitHub"}    | ${false}
    ${"gitlab"}    | ${"GitLab"}    | ${true}
    ${"bitbucket"} | ${"Bitbucket"} | ${false}
  `("For provider", ({ provider, human, hasPersonalAccessTokenSupport }) => {
    beforeEach(async () => {
      await createWrapper({
        accounts: [
          {
            provider: "github",
            hasPersonalAccessToken: false,
            displayName: "",
          },
          {
            provider: "gitlab",
            hasPersonalAccessToken: false,
            displayName: "",
          },
          {
            provider: "bitbucket",
            hasPersonalAccessToken: false,
            displayName: "",
          },
        ],
      });
    });

    it("should display the provider as a connected account", async () => {
      await waitFor(() => {
        expect(wrapper.getByText(human)).toBeTruthy();
      });
    });

    it("should display a set personal access token link", async () => {
      await waitFor(() => {
        const parent = wrapper
          .getByText(human)
          .closest(`[data-testid=${provider}]`);

        expect(/Set personal access token/.test(parent?.innerHTML || "")).toBe(
          hasPersonalAccessTokenSupport
        );
      });
    });
  });

  describe("when has personal access token", () => {
    beforeEach(async () => {
      await createWrapper({
        accounts: [
          { provider: "gitlab", hasPersonalAccessToken: true, displayName: "" },
        ],
      });
    });

    it("displays reset instead of set when hasPersonalAccessToken is true", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Reset personal access token")).toBeTruthy();
      });
    });

    it("the link should toggle the modal", async () => {
      await waitFor(() => {
        wrapper.getByText("Reset personal access token");
      });

      fireEvent.click(wrapper.getByText("Reset personal access token"));

      await waitFor(() => {
        expect(
          wrapper.getAllByText("Reset personal access token")
        ).toHaveLength(2);
      });
    });
  });
});
