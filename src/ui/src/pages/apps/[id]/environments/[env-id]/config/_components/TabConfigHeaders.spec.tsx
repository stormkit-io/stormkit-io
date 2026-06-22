import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockUpdateEnvironment } from "~/testing/nocks/nock_environment";
import TabConfigHeaders from "./TabConfigHeaders";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabConfigHeaders.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let setRefreshToken: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    setRefreshToken = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <TabConfigHeaders
        app={currentApp}
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
      />
    );
  };

  it("should have a form", () => {
    createWrapper({});

    expect(wrapper.getByText("Headers")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Configure response headers. Rules match against the request path and apply to static, SSR and proxied responses."
      )
    ).toBeTruthy();

    const headersFileInput = wrapper.getByLabelText(
      "Headers file location"
    ) as HTMLInputElement;

    expect(headersFileInput.value).toBe("");

    // Initially the docs should not be visible
    expect(() => wrapper.getByText("docs")).toThrow();

    expect(wrapper.getByText("Overwrite headers")).toBeTruthy();

    fireEvent.click(wrapper.getByText("Overwrite headers"));

    expect(wrapper.getByText("docs")).toBeTruthy();
  });

  it("should submit the form without custom headers", async () => {
    createWrapper({});

    await userEvent.type(
      wrapper.getByLabelText("Headers file location"),
      "/headers.json"
    );

    const scope = mockUpdateEnvironment({
      payload: { headers: "", headersFile: "/headers.json" },
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

  it("should submit the form with headers", async () => {
    createWrapper({});

    await userEvent.type(
      wrapper.getByLabelText("Headers file location"),
      "/headers.json"
    );

    const scope = mockUpdateEnvironment({
      payload: { headers: "", headersFile: "/headers.json" },
      status: 200,
      response: {
        ok: true,
      },
    });

    fireEvent.click(wrapper.getByLabelText("Overwrite headers"));
    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });
});
