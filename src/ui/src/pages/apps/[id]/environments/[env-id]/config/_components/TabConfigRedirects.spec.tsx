import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigRedirects from "./TabConfigRedirects";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigRedirects.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabConfigRedirects
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  it("should have a form", () => {
    createWrapper({});

    expect(wrapper.getByText("Redirects")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Configure redirects and path rewrites for your application."
      )
    ).toBeTruthy();

    const redirectsFileInput = wrapper.getByLabelText(
      "Redirects file location"
    ) as HTMLInputElement;

    const errorFileInput = wrapper.getByLabelText(
      "Custom error file"
    ) as HTMLInputElement;

    expect(redirectsFileInput.value).toBe("");
    expect(errorFileInput.value).toBe("");

    // Initially the docs should not be visible
    expect(() => wrapper.getByText("docs")).toThrow();

    expect(wrapper.getByText("Overwrite redirects")).toBeTruthy();

    fireEvent.click(wrapper.getByText("Overwrite redirects"));

    expect(wrapper.getByText("docs")).toBeTruthy();
  });

  it("should submit the form without redirects", async () => {
    createWrapper({});

    await userEvent.type(
      wrapper.getByLabelText("Redirects file location"),
      "/redirects.json"
    );

    await userEvent.type(
      wrapper.getByLabelText("Custom error file"),
      "/index.html"
    );

    const scope = mockUpdateEnvironment({
      payload: { redirectsFile: "/redirects.json", errorFile: "/index.html" },
      status: 200,
      response: {
        ok: true,
      },
    });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("should submit the form with redirects", async () => {
    createWrapper({});

    await userEvent.type(
      wrapper.getByLabelText("Redirects file location"),
      "/redirects.json"
    );

    await userEvent.type(
      wrapper.getByLabelText("Custom error file"),
      "/index.html"
    );

    const scope = mockUpdateEnvironment({
      payload: {
        redirects: [],
        redirectsFile: "/redirects.json",
        errorFile: "/index.html",
      },
      status: 200,
      response: {
        ok: true,
      },
    });

    fireEvent.click(wrapper.getByLabelText("Overwrite redirects"));
    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });
});
