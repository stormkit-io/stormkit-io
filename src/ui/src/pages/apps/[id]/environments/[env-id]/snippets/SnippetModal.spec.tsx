import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { MemoryRouter } from "react-router";
import { waitFor, fireEvent, render, getByText } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import {
  mockUpdateSnippet,
  mockInsertSnippet,
} from "~/testing/nocks/nock_snippets";
import { mockFetchDomains } from "~/testing/nocks/nock_domains";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import SnippetModal from "./SnippetModal";

interface Props {
  app: App;
  env: Environment;
  snippet?: Snippet;
  closeModal: () => void;
  setRefreshToken: (s: number) => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/snippets/SnippetModal.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let snippets: Snippet[];
  let closeModal: Mock;
  let setRefreshToken: Mock;
  let fetchDomainsScope: Scope;

  const snippet: Snippet = {
    title: "Google Analytics",
    prepend: false,
    enabled: false,
    content: "<script>\n    console.log('Hello world');\n</script>",
    location: "head",
  };

  const findDropdown = () =>
    wrapper.getByLabelText("Inject only for specified domains");
  const findOption = (text: string) => getByText(document.body, text);

  const createWrapper = ({ app, env, closeModal, snippet }: Props) => {
    fetchDomainsScope = mockFetchDomains({
      appId: app.id,
      envId: env.id!,
      verified: true,
      response: {
        domains: [
          { domainName: "www.e.org", verified: true, id: "2" },
          { domainName: "e.org", verified: true, id: "2" },
        ],
      },
    });

    wrapper = render(
      <MemoryRouter>
        <AppContext.Provider
          value={{
            app,
            environments: [env],
            setRefreshToken: vi.fn(),
          }}
        >
          <EnvironmentContext.Provider value={{ environment: env }}>
            <SnippetModal
              setRefreshToken={setRefreshToken}
              closeModal={closeModal}
              snippet={snippet}
            />
          </EnvironmentContext.Provider>
        </AppContext.Provider>
      </MemoryRouter>
    );
  };

  describe("insert mode", () => {
    beforeEach(() => {
      currentApp = mockApp();
      currentEnv = mockEnvironment({ app: currentApp });
      snippets = [];
      closeModal = vi.fn();
      setRefreshToken = vi.fn();

      createWrapper({
        app: currentApp,
        env: currentEnv,
        closeModal,
        setRefreshToken,
      });
    });

    it("should fetch domains", async () => {
      await waitFor(() => {
        expect(fetchDomainsScope.isDone()).toBe(true);
      });
    });

    it("should handle form submission", async () => {
      const scope = mockInsertSnippet({
        appId: currentApp.id,
        envId: currentEnv.id!,
        snippets: [
          { ...snippet, rules: { hosts: ["*.dev", "www.e.org", "e.org"] } },
        ],
      });

      await userEvent.type(wrapper.getByLabelText("Title"), "Google Analytics");

      const selector = findDropdown();
      expect(selector).toBeTruthy();
      fireEvent.mouseDown(selector);

      fireEvent.click(findOption("All development endpoints (*.dev)"));
      fireEvent.click(findOption("www.e.org"));
      fireEvent.click(findOption("e.org"));

      // Closes the dropdown
      await userEvent.keyboard("{Escape}");

      await fireEvent.click(wrapper.getByText("Create"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(closeModal).toHaveBeenCalled();
        expect(setRefreshToken).toHaveBeenCalled();
      });
    });
  });

  describe("edit mode", () => {
    beforeEach(() => {
      currentApp = mockApp();
      currentEnv = mockEnvironment({ app: currentApp });
      snippets = [{ ...snippet, location: "body", id: "1" }];

      closeModal = vi.fn();
      setRefreshToken = vi.fn();

      createWrapper({
        app: currentApp,
        env: currentEnv,
        snippet: snippets[0],
        closeModal,
        setRefreshToken,
      });
    });

    it("should handle form submission", async () => {
      const scope = mockUpdateSnippet({
        appId: currentApp.id,
        envId: currentEnv.id!,
        snippet: {
          ...snippet,
          location: "body",
          title: "Hotjar",
          id: "1",
          rules: {},
        },
      });

      await userEvent.clear(wrapper.getByLabelText("Title"));
      await userEvent.type(wrapper.getByLabelText("Title"), "Hotjar");
      await fireEvent.click(wrapper.getByText("Update"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      expect(closeModal).toHaveBeenCalled();
      expect(setRefreshToken).toHaveBeenCalled();
    });
  });
});
