import { render, RenderResult } from "@testing-library/react";
import { describe, expect, it, beforeEach } from "vitest";
import { MemoryRouter } from "react-router";
import mockApp from "~/testing/data/mock_app";
import AppName from "./AppName";

interface Props {
  app: App;
}

describe("~/components/AppName/AppName.tsx", () => {
  let wrapper: RenderResult;
  let app: App;

  const createWrapper = ({ app }: Props) => {
    wrapper = render(
      <MemoryRouter>
        <AppName app={app} />
      </MemoryRouter>
    );
  };

  beforeEach(() => {
    app = mockApp();
  });

  it("should not have stormkit logo when app is the stormkit ui", () => {
    createWrapper({ app });
    expect(() => wrapper.getByTestId("app-logo")).toThrow();
  });

  it("should display the app name with the display name and repo", () => {
    createWrapper({ app });

    expect(() => wrapper.getByTestId("BoltIcon")).toThrow();
    expect(wrapper.getByText("app")).toBeTruthy();
    expect(wrapper.getByText("stormkit-io/frontend")).toBeTruthy();
  });

  it("should display only the name if an app is bare", () => {
    createWrapper({ app: { ...app, isBare: true } });

    expect(wrapper.container.textContent).toBe("app");
    expect(wrapper.getByTestId("BoltIcon")).toBeTruthy();
    expect(() => wrapper.getByText("stormkit-io/frontend")).toThrow();
  });
});
