import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockApp from "~/testing/data/mock_app";
import mockTeams from "~/testing/data/mock_teams";
import TabAuthWall from "./TabAuthWall";

function Location() {
  const { pathname, search } = useLocation();
  return <span>{pathname + search}</span>;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabAuthWall/TabAuthWall.tsx", () => {
  it("sends old Auth Wall tab links to the team page, filtered to the app", () => {
    const teams = mockTeams();
    const app = mockApp();
    app.teamId = teams[1].id;

    const wrapper = render(
      <AuthContext.Provider value={{ teams }}>
        <MemoryRouter initialEntries={["/apps/1/environments/2#authwall"]}>
          <Routes>
            <Route
              path="/apps/1/environments/2"
              element={<TabAuthWall app={app} />}
            />
            <Route path="*" element={<Location />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>,
    );

    expect(
      wrapper.getByText(`/${teams[1].slug}/auth-wall?appId=${app.id}`),
    ).toBeTruthy();
  });
});
