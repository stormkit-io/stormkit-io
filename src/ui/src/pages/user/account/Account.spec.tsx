import type { RenderResult } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import mockTeams from "~/testing/data/mock_teams";
import mockUser from "~/testing/data/mock_user";
import { AuthContext } from "~/pages/auth/Auth.context";
import { renderWithRouter } from "~/testing/helpers";
import Account from "./Account";

interface Props {
  user?: User;
}

describe("~/pages/user/account/Account", () => {
  let wrapper: RenderResult;

  const teams = mockTeams();

  const createWrapper = ({ user = mockUser() }: Props) => {
    wrapper = renderWithRouter({
      el: () => (
        <AuthContext.Provider value={{ user, teams, accounts: [] }}>
          <Account />
        </AuthContext.Provider>
      ),
    });
  };

  it("should load connected accounts", () => {
    createWrapper({
      user: mockUser(),
    });

    expect(wrapper.getByText("Connected Accounts")).toBeTruthy();
    expect(
      wrapper.getByText(/List of connected emails and providers/)
    ).toBeTruthy();
  });
});
