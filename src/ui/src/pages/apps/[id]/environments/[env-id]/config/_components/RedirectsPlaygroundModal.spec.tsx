import { RenderResult, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockPlayground } from "~/testing/nocks/nock_redirects_playground";
import RedirectsPlaygroundModal from "./RedirectsPlaygroundModal";

interface WrapperProps {
  app?: App;
  environment?: Environment;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/RedirectsPlaygroundModal.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let closeSpy: Mock;

  const createWrapper = ({ app, environment }: WrapperProps) => {
    closeSpy = vi.fn();
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironments({ app: currentApp })[0];

    wrapper = render(
      <RedirectsPlaygroundModal
        appId={currentApp.id}
        env={currentEnv}
        onClose={closeSpy}
      />
    );
  };

  it("should have a form", () => {
    createWrapper({});

    expect(wrapper.getByText("Redirects Playground")).toBeTruthy();

    const requestUrlInput = wrapper.getByLabelText(
      "Request URL"
    ) as HTMLInputElement;

    expect(requestUrlInput.value).toBe("https://app.stormkit.io");

    expect(
      wrapper.getByText("Provide a URL to test against the redirects.")
    ).toBeTruthy();

    expect(wrapper.getByText("docs").getAttribute("href")).toBe(
      "https://www.stormkit.io/docs/features/redirects-and-path-rewrites"
    );
  });

  it("should submit the form", async () => {
    createWrapper({});

    const scope = mockPlayground({
      appId: currentApp.id,
      envId: currentEnv.id!,
      address: "https://app.stormkit.io",
      redirects: [{ from: "/my-path", to: "/my-new-path", status: 302 }],
      response: {
        match: true,
        proxy: false,
        status: 302,
        redirect: "https://app.stormkit.io/new-url",
        rewrite: "",
      },
    });

    fireEvent.submit(wrapper.getByTestId("redirects-playground-form"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("It's a match!"));
      expect(wrapper.getByText("Redirect 302"));
      expect(wrapper.getByText("https://app.stormkit.io/new-url"));
      expect(wrapper.getByText("Should proxy?"));
      expect(wrapper.getByText("No"));
    });
  });
});
