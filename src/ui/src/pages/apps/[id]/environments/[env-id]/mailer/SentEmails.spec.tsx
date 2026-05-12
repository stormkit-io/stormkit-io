import type { Scope } from "nock";
import type { MockEmail } from "~/testing/nocks/nock_mailer";
import { describe, expect, it, vi } from "vitest";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import { mockFetchEmails } from "~/testing/nocks/nock_mailer";
import SentEmails from "./SentEmails";

interface WrapperProps {
  emails?: MockEmail[];
}

describe("~/pages/apps/[id]/environments/[env-id]/mailer/SentEmails.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;
  let currentApp: App;
  let currentEnv: Environment;

  const createWrapper = ({ emails = [] }: WrapperProps) => {
    currentApp = mockApp();
    currentApp.id = "1";
    currentEnv = mockEnvironments({ app: currentApp })[0];

    fetchScope = mockFetchEmails({
      appId: currentApp.id,
      envId: currentEnv.id!,
      response: { emails },
    });

    wrapper = render(
      <AppContext.Provider
        value={{
          app: currentApp,
          environments: [currentEnv],
          setRefreshToken: vi.fn(),
        }}
      >
        <EnvironmentContext.Provider value={{ environment: currentEnv }}>
          <SentEmails />
        </EnvironmentContext.Provider>
      </AppContext.Provider>,
    );
  };

  it("should display empty state when no emails", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    expect(wrapper.getByText("Sent Emails")).toBeTruthy();
    expect(wrapper.getByText("No emails sent yet.")).toBeTruthy();
  });

  it("should render list of emails", async () => {
    const emails: MockEmail[] = [
      {
        id: "1",
        envId: currentEnv?.id || "1",
        from: "sender@example.com",
        to: "recipient@example.com",
        subject: "Welcome aboard",
        body: "<p>Hello!</p>",
        sentAt: 1700000000,
      },
    ];

    createWrapper({ emails });

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    expect(wrapper.getByText("Welcome aboard")).toBeTruthy();
    expect(wrapper.getByText("To: recipient@example.com")).toBeTruthy();
  });

  it("should open email preview dialog on row click", async () => {
    const emails: MockEmail[] = [
      {
        id: "1",
        envId: currentEnv?.id || "1",
        from: "sender@example.com",
        to: "recipient@example.com",
        subject: "Welcome aboard",
        body: "<p>Hello!</p>",
        sentAt: 1700000000,
      },
    ];

    createWrapper({ emails });

    await waitFor(() => {
      expect(wrapper.getByText("Welcome aboard")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByText("Welcome aboard"));

    await waitFor(() => {
      expect(wrapper.getByText("From: sender@example.com")).toBeTruthy();
      expect(wrapper.getAllByText("To: recipient@example.com")).toHaveLength(2);
    });
  });
});
