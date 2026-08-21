import type { Mock } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import nock from "nock";
import EmailIcon from "@mui/icons-material/Email";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import { mockFetchDomains } from "~/testing/nocks/nock_domains";
import ProviderSettings from "./ProviderSettings";
import type { AuthProvider } from "./actions";

const apiDomain = process.env.API_DOMAIN || "";

interface Props {
  environment: Environment;
  isDrawerOpen: boolean;
  provider: AuthProvider;
  onClose: () => void;
}

const mockProvider: AuthProvider = {
  id: "google",
  icon: EmailIcon,
  name: "Google",
  drawerTitle: "Google OAuth Settings",
  drawerDesc: "Sign in with Google OAuth 2.0.",
  hasRedirectUrl: true,
  redirectUrl: "/_stormkit/auth/callback",
  hasAuthUrl: true,
  authUrl: "https://api.example.com/auth",
  enabled: false,
  fields: [
    { name: "clientId", label: "Client ID", value: "test-client-id" },
    { name: "clientSecret", label: "Client Secret", value: "test-secret" },
  ],
  steps: ["Step 1: Do something", "Step 2: Do something else"],
};

describe("~/pages/apps/[id]/environments/[env-id]/skauth/ProviderSettings.tsx", () => {
  let wrapper: RenderResult;
  let onClose: Mock;
  let setRefreshToken: Mock;
  let currentEnv: Environment;

  const createWrapper = (props: Partial<Props> = {}) => {
    onClose = vi.fn();
    setRefreshToken = vi.fn();

    const app = mockApp();
    currentEnv = mockEnv({ app });

    mockFetchDomains({
      appId: app.id!,
      envId: currentEnv.id!,
      verified: true,
      status: 200,
      response: { domains: [] },
    });

    wrapper = render(
      <ProviderSettings
        environment={currentEnv}
        setRefreshToken={setRefreshToken}
        isDrawerOpen={true}
        provider={mockProvider}
        onClose={onClose}
        {...props}
      />,
    );
  };

  describe("when drawer is open", () => {
    beforeEach(() => {
      createWrapper();
    });

    it("should render the drawer content", () => {
      // Title and description
      expect(wrapper.getByText("Google OAuth Settings")).toBeTruthy();
      expect(wrapper.getByText("Sign in with Google OAuth 2.0.")).toBeTruthy();

      // Form fields
      expect(wrapper.getByLabelText("Client ID")).toBeTruthy();
      expect(wrapper.getByLabelText("Client Secret")).toBeTruthy();

      // Enable switch
      expect(wrapper.getByText("Enable provider")).toBeTruthy();
      expect(
        wrapper.getByText(
          "Allow or disallow sign-in with this provider. Disabling will not delete existing users.",
        ),
      ).toBeTruthy();

      // Callback URL — built from the environment's preview domain.
      expect(wrapper.getByText("Callback URL")).toBeTruthy();
      expect(
        wrapper.getByDisplayValue(
          "https://app.stormkit.io/_stormkit/auth/callback",
        ),
      ).toBeTruthy();

      // Authorization URL
      expect(wrapper.getByText("Authorization URL")).toBeTruthy();
      expect(
        wrapper.getByDisplayValue("https://api.example.com/auth/google"),
      ).toBeTruthy();
      expect(
        wrapper.getByText(/redirect users here to start sign-in/),
      ).toBeTruthy();

      // Setup steps
      expect(wrapper.getByText("1. Step 1: Do something")).toBeTruthy();
      expect(wrapper.getByText("2. Step 2: Do something else")).toBeTruthy();

      // Buttons
      expect(wrapper.getByText("Cancel")).toBeTruthy();
      expect(wrapper.getByText("Save")).toBeTruthy();
    });

    it("should call onClose when cancel is clicked", async () => {
      fireEvent.click(wrapper.getByText("Cancel"));
      expect(onClose).toHaveBeenCalled();
    });
  });

  describe("when provider is enabled", () => {
    beforeEach(() => {
      createWrapper({
        provider: { ...mockProvider, enabled: true },
      });
    });

    it("should render the switch as checked", () => {
      const switchInput = wrapper.getByRole("switch") as HTMLInputElement;
      expect(switchInput.checked).toBe(true);
    });
  });

  describe("when provider does not have auth URL", () => {
    beforeEach(() => {
      createWrapper({
        provider: {
          ...mockProvider,
          hasAuthUrl: false,
          authUrl: undefined,
        },
      });
    });

    it("should not render the Authorization URL section", () => {
      expect(() => wrapper.getByText("Authorization URL")).toThrow();
      expect(() =>
        wrapper.getByText(/redirect users here to start sign-in/),
      ).toThrow();
    });

    it("should still render the Callback URL", () => {
      expect(wrapper.getByText("Callback URL")).toBeTruthy();
    });
  });

  describe("when the env has multiple domains", () => {
    beforeEach(() => {
      nock.cleanAll();
      onClose = vi.fn();
      setRefreshToken = vi.fn();

      const app = mockApp();
      currentEnv = mockEnv({ app });

      mockFetchDomains({
        appId: app.id!,
        envId: currentEnv.id!,
        verified: true,
        status: 200,
        response: {
          domains: [
            {
              id: "1",
              domainName: "api.triplan.to",
              verified: true,
            } as Domain,
          ],
        },
      });

      wrapper = render(
        <ProviderSettings
          environment={currentEnv}
          setRefreshToken={setRefreshToken}
          isDrawerOpen={true}
          provider={mockProvider}
          onClose={onClose}
        />,
      );
    });

    it("should offer a domain dropdown and default to the preview domain", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Domain")).toBeTruthy();
      });

      // The callback URL defaults to the first (preview) domain.
      expect(
        wrapper.getByDisplayValue(
          "https://app.stormkit.io/_stormkit/auth/callback",
        ),
      ).toBeTruthy();
    });
  });

  describe("when drawer is closed", () => {
    beforeEach(() => {
      createWrapper({ isDrawerOpen: false });
    });

    it("should not render the drawer content", () => {
      expect(() => wrapper.getByText("Google OAuth Settings")).toThrow();
    });
  });

  describe("form submission", () => {
    beforeEach(() => {
      nock.cleanAll();
      createWrapper();
    });

    afterEach(() => {
      nock.cleanAll();
    });

    it.each([
      { toggleSwitch: false, expectedStatus: false, desc: "disabled" },
      { toggleSwitch: true, expectedStatus: true, desc: "enabled" },
    ])(
      "should send status as $expectedStatus when provider is $desc",
      async ({ toggleSwitch, expectedStatus }) => {
        if (toggleSwitch) {
          fireEvent.click(wrapper.getByRole("switch"));
        }

        const scope = nock(apiDomain)
          .post("/skauth", {
            envId: currentEnv.id,
            providerName: "google",
            clientId: "test-client-id",
            clientSecret: "test-secret",
            status: expectedStatus,
          })
          .reply(200, { success: true });

        fireEvent.click(wrapper.getByText("Save"));

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);
        });
      },
    );

    it("should call onClose on successful submission", async () => {
      nock(apiDomain).post("/skauth").reply(200, { success: true });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(onClose).toHaveBeenCalled();
        expect(setRefreshToken).toHaveBeenCalled();
      });
    });

    it("should display error message when API fails", async () => {
      nock(apiDomain).post("/skauth").reply(500);

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "Something went wrong while saving provider settings.",
          ),
        ).toBeTruthy();
      });
    });

    it("should display field validation errors on 400", async () => {
      nock(apiDomain)
        .post("/skauth")
        .reply(400, { errors: { clientId: "Client ID is required" } });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(wrapper.getByText("Client ID is required")).toBeTruthy();
      });
    });
  });
});
