import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import mockDomain from "~/testing/data/mock_domain";
import {
  mockDeleteDomain,
  mockFetchDomains,
} from "~/testing/nocks/nock_domains";
import TabDomainConfig from "./TabDomainConfig";

interface Props {
  app: App;
  environment: Environment;
}

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabDomainConfig/TabDomainConfig.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let domain: Domain;

  const createWrapper = ({ app, environment }: Props) => {
    wrapper = render(<TabDomainConfig app={app} environment={environment} />);
  };

  describe("with no domain", () => {
    beforeEach(async () => {
      currentApp = mockApp();
      currentEnv = mockEnvironment({ app: currentApp });

      const scope = mockFetchDomains({
        envId: currentEnv.id!,
        appId: currentApp.id,
        status: 200,
        response: { domains: [] },
      });

      createWrapper({ app: currentApp, environment: currentEnv });

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });

    it("should display a configure domain button", async () => {
      await fireEvent.click(wrapper.getByText("Add domain"));
      await waitFor(() => {
        expect(wrapper.getByText("Setup a new domain")).toBeTruthy();
      });
    });

    it("should display an informative text", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("No custom domain configuration found.")
        ).toBeTruthy();
        expect(
          wrapper.getByText("Add a domain to serve your app directly from it.")
        ).toBeTruthy();
      });
    });
  });

  describe("with domain", () => {
    beforeEach(async () => {
      currentApp = mockApp();
      currentEnv = mockEnvironment({ app: currentApp });
      domain = mockDomain();
      domain.customCert = {
        key: "my-cert-key",
        value: "my-cert-value",
      };

      const scope = mockFetchDomains({
        envId: currentEnv.id!,
        appId: currentApp.id,
        status: 200,
        response: {
          domains: [
            domain,
            { id: "2", domainName: "www.stormkit.io", verified: false },
            {
              id: "3",
              domainName: "api.stormkit.io",
              verified: true,
              lastPing: {
                status: 200,
                lastPingAt: new Date(2024, 6, 5).getTime() / 1000,
              },
            },
          ],
        },
      });

      createWrapper({ app: currentApp, environment: currentEnv });

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });

    it("should not display the configure domain button anymore", async () => {
      await waitFor(() => {
        expect(() => wrapper.getByText("Configure domain")).toThrow();
      });
    });

    it("should open the custom certificate modal", async () => {
      await waitFor(() => {
        const expandButton = wrapper.getAllByLabelText("expand").at(0);
        expect(expandButton).toBeTruthy();
        fireEvent.click(expandButton!);
      });

      await waitFor(() => {
        const customCertButton = wrapper
          .getAllByText("Custom certificate")
          .at(1);

        expect(customCertButton).toBeTruthy();
        fireEvent.click(customCertButton!);
      });

      await waitFor(() => {
        expect(wrapper.getByText("Configure custom certificate")).toBeTruthy();
      });
    });

    it("should list domains", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("app.stormkit.io")).toBeTruthy();
        expect(wrapper.getByText("www.stormkit.io")).toBeTruthy();
        expect(wrapper.getByText("api.stormkit.io")).toBeTruthy();
        expect(wrapper.getByTestId("app.stormkit.io-status").textContent).toBe(
          "Status: not yet pinged"
        );
        expect(wrapper.getByTestId("www.stormkit.io-status").textContent).toBe(
          "Status: not yet verified"
        );
        expect(wrapper.getByTestId("api.stormkit.io-status").textContent).toBe(
          "Status: 200"
        );
      });
    });

    it("should allow removing a domain", async () => {
      const scope = mockDeleteDomain({
        appId: currentApp.id,
        envId: currentEnv.id!,
        domainId: domain.id,
        response: { ok: true },
      });

      await waitFor(() => {
        expect(wrapper.getAllByLabelText("expand").at(0)).toBeTruthy();
      });

      fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);
      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Yes, continue")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });
  });
});
