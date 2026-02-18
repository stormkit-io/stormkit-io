import type { Scope } from "nock";
import type { MockMailerConfig } from "~/testing/nocks/nock_mailer";
import { describe, expect, it, vi } from "vitest";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AppContext } from "~/pages/apps/[id]/App.context";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockApp from "~/testing/data/mock_app";
import mockEnvironments from "~/testing/data/mock_environments";
import {
  mockFetchMailerConfig,
  mockSendTestEmail,
  mockSetMailerConfig,
} from "~/testing/nocks/nock_mailer";
import TabMailer from "./Mailer";

interface WrapperProps {
  config?: MockMailerConfig;
  app?: App;
  setRefreshToken?: () => void;
}

describe("~/pages/apps/[id]/environments/[env-id]/mailer/Mailer.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;
  let currentApp: App;
  let currentEnv: Environment;

  const setupFetchScope = (config?: MockMailerConfig) => {
    fetchScope = mockFetchMailerConfig({
      appId: currentApp.id,
      envId: currentEnv.id!,
      response: config ? { config } : undefined,
    });
  };

  const createWrapper = ({ app, config }: WrapperProps) => {
    currentApp = app || mockApp();
    currentApp.id = "1"; // Same as api key id
    currentEnv = mockEnvironments({ app: currentApp })[0];

    setupFetchScope(config);

    wrapper = render(
      <AppContext.Provider
        value={{
          app: currentApp,
          environments: [currentEnv],
          setRefreshToken: vi.fn(),
        }}
      >
        <EnvironmentContext.Provider value={{ environment: currentEnv }}>
          <TabMailer />
        </EnvironmentContext.Provider>
      </AppContext.Provider>,
    );
  };

  it("should fetch mailer config", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    const subheader = "Simple Email Service to send transactional emails.";

    // Header
    expect(wrapper.getByText("Mailer Configuration")).toBeTruthy();
    expect(wrapper.getByText(subheader)).toBeTruthy();

    // Not yet configured, so no test email
    expect(() => wrapper.getByText("Send test email")).toThrow();
  });

  it("should create a mailer configuration", async () => {
    createWrapper({});

    const username = "joe@example.org";
    const password = "my-app-token";
    const smtpHost = "smtp.example.org";
    const smtpPort = "587";

    const scope = mockSetMailerConfig({
      appId: currentApp.id,
      envId: currentEnv.id,
      username,
      password,
      smtpHost,
      smtpPort,
    });

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
      expect(wrapper.getByLabelText("SMTP Host")).toBeTruthy();
    });

    await userEvent.type(wrapper.getByLabelText("SMTP Host"), smtpHost);
    await userEvent.type(wrapper.getByLabelText("SMTP Port"), smtpPort);
    await userEvent.type(wrapper.getByLabelText("Username"), username);
    await userEvent.type(wrapper.getByLabelText("Password"), password);

    setupFetchScope();

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByText("Mailer configuration saved successfully."));
    });

    // Should re-fetch
    expect(fetchScope.isDone()).toBe(true);
  });

  it("should send a test email", async () => {
    createWrapper({
      config: {
        host: "smtp.example.org",
        port: "587",
        username: "joe@example.org",
        password: "123",
      },
    });

    await waitFor(() => {
      expect(wrapper.getByDisplayValue("smtp.example.org")).toBeTruthy();
    });

    const scope = mockSendTestEmail({
      appId: currentApp.id,
      envId: currentEnv.id!,
      from: "joe@example.org",
      to: "joe@example.org",
    });

    fireEvent.click(wrapper.getByText("Send test email"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  });

  it("should display help drawer with configuration guide", async () => {
    createWrapper({});

    await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
    });

    // Click the "Learn more" link
    fireEvent.click(await waitFor(() => wrapper.getByText("Learn more.")));

    // Help drawer should open with expected content
    await waitFor(() => {
      expect(wrapper.getByText("Mailer Configuration Help")).toBeTruthy();

      // Check for help content
      expect(
        wrapper.getByText(/Configure your SMTP settings to enable/),
      ).toBeTruthy();
      expect(wrapper.getByText("Environment Variables")).toBeTruthy();
      expect(wrapper.getAllByText("MAILER_URL")).toBeTruthy();
      expect(
        wrapper.getByText(
          "The SMTP connection string (e.g., smtp://user:pass@host:port)",
        ),
      ).toBeTruthy();

      // Gmail configuration example
      expect(wrapper.getByText("Example Configuration (Gmail)")).toBeTruthy();
      expect(wrapper.getByText("smtp.gmail.com")).toBeTruthy();
      expect(wrapper.getByText("587")).toBeTruthy();
      expect(wrapper.getByText("your-email@gmail.com")).toBeTruthy();

      // API documentation link
      const apiDocLink = wrapper.getByText("API documentation");
      expect(apiDocLink).toBeTruthy();
      expect(apiDocLink.closest("a")?.getAttribute("href")).toBe(
        "https://www.stormkit.io/docs/api/mailer",
      );
    });
  });
});
