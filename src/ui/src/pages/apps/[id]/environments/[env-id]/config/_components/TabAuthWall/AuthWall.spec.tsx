import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import AuthWall from "./AuthWall";

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabAuthWall/AuthWall.tsx", () => {
  it("should load two sections: auth config and auth users", () => {
    const app = mockApp();
    const env = mockEnvironment({ app });
    const wrapper = render(<AuthWall app={app} environment={env} />);

    expect(
      wrapper.getByText(
        "Limit access to your deployments with an authentication wall."
      )
    );

    expect(
      wrapper.getByText(
        "Manage users who have access to deployments in this environment."
      )
    );
  });
});
