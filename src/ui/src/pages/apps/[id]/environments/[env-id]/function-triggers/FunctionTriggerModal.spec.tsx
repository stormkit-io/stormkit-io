import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, vi, type Mock } from "vitest";
import { fireEvent, waitFor, getAllByText } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import mockApp from "~/testing/data/mock_app";
import mockFunctionTriggers from "~/testing/data/mock_function_triggers";
import mockEnvironment from "~/testing//data/mock_environment";
import * as mockActions from "~/testing/nocks/nock_function_triggers";
import { mockFetchDomains } from "~/testing/nocks/nock_domains";
import { renderWithRouter } from "~/testing/helpers";
import FunctionTriggerModal from "./FunctionTriggerModal";

interface Props {
  app?: App;
  environment?: Environment;
  trigger?: FunctionTrigger;
}

describe("~/apps/[id]/environments/[env-id]/function-triggers/_components/FunctionTriggerModal.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let closeModal: Mock;
  let successHandler: Mock;

  const findDropdown = () => wrapper.getByRole("combobox");
  const findOption = (text: string) => getAllByText(document.body, text).at(0);
  const openDropdown = async () => {
    let selector;

    await waitFor(() => {
      selector = findDropdown();
    });

    if (!selector) {
      throw new Error("Cannot find dropdown");
    }

    expect(selector).toBeTruthy();
    fireEvent.mouseDown(selector);
    return selector;
  };

  const createWrapper = async ({ app, environment, trigger }: Props = {}) => {
    currentApp = app || mockApp();
    currentEnv = environment || mockEnvironment({ app: currentApp });
    closeModal = vi.fn();
    successHandler = vi.fn();

    const domains = [
      { domainName: "www.e.org", verified: true, id: "2" },
      { domainName: "e.org", verified: true, id: "3" },
    ];

    const fetchDomainsScope = mockFetchDomains({
      appId: currentApp.id,
      envId: currentEnv.id!,
      verified: true,
      response: {
        domains,
      },
    });

    wrapper = renderWithRouter({
      el: () => (
        <FunctionTriggerModal
          triggerFunction={trigger}
          onSuccess={successHandler}
          closeModal={closeModal}
          app={currentApp}
          environment={currentEnv}
        />
      ),
    });

    await waitFor(() => {
      expect(fetchDomainsScope.isDone()).toBe(true);
    });
  };

  it("should update a given trigger", async () => {
    const trigger = mockFunctionTriggers()[0];
    trigger.options.url = "https://www.e.org/previous-test";

    await createWrapper({ trigger });

    const scope = mockActions.mockUpdateFunctionTrigger({
      tfid: trigger.id!,
      appId: currentApp.id,
      envId: currentEnv.id!,
      status: trigger.status,
      options: trigger.options,
      cron: "2 * * * *",
    });

    const cronInput = wrapper.getByLabelText("Cron");

    await userEvent.clear(cronInput);
    await userEvent.type(cronInput, "2 * * * *");
    await fireEvent.click(wrapper.getByText("Update"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(successHandler).toHaveBeenCalled();
    });
  });

  it("should create a new trigger", async () => {
    await createWrapper();
    await openDropdown();
    await fireEvent.click(findOption("www.e.org")!);

    const scope = mockActions.mockCreateFunctionTrigger({
      appId: currentApp.id,
      envId: currentEnv.id!,
      status: false,
      options: {
        url: `https://www.e.org/test`,
        method: "GET",
        headers: {},
        payload: "",
      },
      cron: "1 * * * *",
    });

    await userEvent.type(wrapper.getByLabelText("Cron"), "1 * * * *");
    await userEvent.type(wrapper.getByLabelText(/trigger periodi/), "test");
    await fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(successHandler).toHaveBeenCalled();
    });
  });
});
