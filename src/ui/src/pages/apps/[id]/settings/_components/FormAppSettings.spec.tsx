import type { RenderResult } from "@testing-library/react";
import type { AppSettings } from "../types";
import { describe, expect, it, beforeEach } from "vitest";
import { waitFor, fireEvent, render } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import { mockUpdateSettings } from "~/testing/nocks/nock_update_settings";
import FormAppSettings from "./FormAppSettings";

interface Props {
  app: App;
  additionalSettings: AppSettings;
}

describe("~/pages/apps/[id]/settings/_components/FormAppSettings", () => {
  let currentApp: App;
  let wrapper: RenderResult;

  const createWrapper = ({ app, additionalSettings }: Props) => {
    wrapper = render(
      <FormAppSettings app={app} additionalSettings={additionalSettings} />
    );
  };

  describe("when it is a normal app", () => {
    beforeEach(() => {
      currentApp = mockApp();
      createWrapper({
        app: currentApp,
        additionalSettings: { envs: [] },
      });
    });

    it("have the form prefilled", async () => {
      await waitFor(() => {
        expect(wrapper.getByDisplayValue(currentApp.displayName)).toBeTruthy();
        expect(
          wrapper.getByDisplayValue(
            "https://gitlab.com/stormkit-io/frontend.git"
          )
        ).toBeTruthy();
      });
    });

    it("updates the settings", async () => {
      const scope = mockUpdateSettings({
        payload: {
          appId: currentApp.id,
          displayName: currentApp.displayName,
          repo: currentApp.repo,
        },
        response: { ok: true },
      });

      fireEvent.click(wrapper.getByText("Update"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });
  });

  describe("when it is a bare app", () => {
    beforeEach(() => {
      currentApp = mockApp();
      currentApp.isBare = true;

      createWrapper({
        app: currentApp,
        additionalSettings: { envs: [] },
      });
    });

    it("should not display the repository field", () => {
      expect(() =>
        wrapper.getByDisplayValue("https://gitlab.com/stormkit-io/frontend.git")
      ).toThrow();
    });
  });
});
