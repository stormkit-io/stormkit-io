import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockTeams from "~/testing/data/mock_teams";
import TeamNav from "./TeamNav";

describe("~/layouts/TeamLayout/TeamNav.tsx", () => {
  const createWrapper = (team: Team) =>
    render(
      <AuthContext.Provider value={{ teams: mockTeams() }}>
        <MemoryRouter initialEntries={[`/${team.slug}`]}>
          <Routes>
            <Route path="/:team" element={<TeamNav />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    );

  it.each(mockTeams())("links to the Auth Wall page of $name", team => {
    const wrapper = createWrapper(team);
    const link = wrapper.getByText("Auth Wall").closest("a");

    expect(link?.getAttribute("href")).toBe(`/${team.slug}/auth-wall`);
  });
});
