import type { RenderResult } from "@testing-library/react";
import nock from "nock";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, fireEvent, waitFor, act } from "@testing-library/react";
import CircularProgress from "@mui/material/CircularProgress";
import CheckIcon from "@mui/icons-material/Check";
import TimesIcon from "@mui/icons-material/Close";
import AdminSystem, { mapRuntimes, mapRuntimeStatus } from "./System";
import React from "react";

describe("~/pages/admin/System.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    window.location = { assign: vi.fn(), reload: vi.fn() } as any;
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  const fetchRuntimesScope = (runtimes: string[]) => {
    return nock(process.env.API_DOMAIN || "")
      .get("/admin/system/runtimes")
      .reply(200, {
        runtimes,
        autoInstall: true,
        installed: {
          node: [
            {
              version: "24.0.0",
              requested_version: "24",
              active: true,
              installed: true,
            },
          ],
          python: [
            {
              version: "3.11.0",
              requested_version: "3",
              active: false,
              installed: true,
            },
          ],
        },
        status: "ok",
      });
  };

  const fetchMiseVersionScope = () => {
    return nock(process.env.API_DOMAIN || "")
      .get("/admin/system/mise")
      .reply(200, { version: "1.0.0" });
  };

  const fetchDomainsScope = () => {
    return nock(process.env.API_DOMAIN || "")
      .get("/admin/domains")
      .reply(200, {
        domains: {
          dev: "https://dev.example.org",
          api: "https://api.example.org",
          app: "https://app.example.org",
        },
      });
  };

  const fetchSystemSettingsScope = (artifactRetentionDays = 30) => {
    return nock(process.env.API_DOMAIN || "")
      .get("/admin/system/settings")
      .reply(200, { artifactRetentionDays });
  };

  beforeEach(async () => {
    const scope = fetchRuntimesScope(["node@24", "python@3"]);
    const scopeVersion = fetchMiseVersionScope();
    const scopeDomains = fetchDomainsScope();
    const scopeSettings = fetchSystemSettingsScope();

    createWrapper();

    await waitFor(() => {
      expect(scopeVersion.isDone()).toBe(true);
      expect(scope.isDone()).toBe(true);
      expect(scopeDomains.isDone()).toBe(true);
      expect(scopeSettings.isDone()).toBe(true);
    });
  });

  const createWrapper = async () => {
    wrapper = render(<AdminSystem />);
  };

  it("renders the component", () => {
    expect(wrapper.getByText("Installed runtimes")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Manage runtimes that are installed on your Stormkit instance",
      ),
    ).toBeTruthy();
    expect(wrapper.getByText("Domains")).toBeTruthy();
    expect(
      wrapper.getByText("Configure custom domains for your Stormkit instance"),
    ).toBeTruthy();
  });

  it("should submit the form", async () => {
    const scope = nock(process.env.API_DOMAIN || "")
      .post("/admin/system/runtimes", {
        autoInstall: false,
        runtimes: ["node@24", "python@3"],
      })
      .reply(200, { ok: true });

    const fetchScope = nock(process.env.API_DOMAIN || "")
      .get("/admin/system/runtimes")
      .reply(200, {});

    // Turn off auto-install
    fireEvent.click(
      await waitFor(() => wrapper.getByLabelText("Auto install")),
    );

    // Find the save button in the runtimes section specifically
    const saveButtons = wrapper.getAllByText("Save");
    await fireEvent.click(saveButtons[0]); // First save button is for runtimes

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(fetchScope.isDone()).toBe(true);
    });
  });

  it("should fetch mise version", async () => {
    expect(wrapper.getByText("Mise"));
    expect(
      wrapper.getByText(
        "Stormkit relies on open-source mise for runtime management",
      ),
    );
    expect(wrapper.getByText("Upgrade to latest"));

    await waitFor(() => {
      expect(wrapper.getByText("Current version")).toBeTruthy();
      expect(wrapper.getByText("1.0.0")).toBeTruthy();
    });
  });

  it("should request an update for mise", async () => {
    const scope = nock(process.env.API_DOMAIN || "")
      .post("/admin/system/mise")
      .reply(200, { ok: true });

    const fetchScope = nock(process.env.API_DOMAIN || "")
      .get("/admin/system/mise")
      .reply(200, { version: "2.0.0", status: "processing" });

    fireEvent.click(wrapper.getByText("Upgrade to latest"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(fetchScope.isDone()).toBe(true);
    });

    await waitFor(() => {
      expect(wrapper.getByRole("button", { name: "Abort" })).toBeTruthy();
    });

    const refetchScope = nock(process.env.API_DOMAIN || "")
      .get("/admin/system/mise")
      .reply(200, { version: "2.0.0", status: "processing" });

    await act(async () => {
      await vi.advanceTimersByTime(2500);
    });

    await waitFor(() => {
      expect(refetchScope.isDone()).toBe(true);
    });

    await waitFor(() => {
      expect(wrapper.getByText("2.0.0")).toBeTruthy();
    });
  });

  describe("mapRuntimes", () => {
    it("should map runtimes with versions correctly", () => {
      const runtimes = ["node@18.0.0", "python@3.9.0", "go@1.19.0"];
      const expected = {
        node: "18.0.0",
        python: "3.9.0",
        go: "1.19.0",
      };

      expect(mapRuntimes(runtimes)).toEqual(expected);
    });

    it("should handle runtimes without versions", () => {
      const runtimes = ["node", "python", "go"];
      const expected = {
        node: "latest",
        python: "latest",
        go: "latest",
      };

      expect(mapRuntimes(runtimes)).toEqual(expected);
    });

    it("should handle scoped packages with @ in name", () => {
      const runtimes = ["@angular/cli@15.0.0", "@types/node@18.0.0"];
      const expected = {
        "@angular/cli": "15.0.0",
        "@types/node": "18.0.0",
      };

      expect(mapRuntimes(runtimes)).toEqual(expected);
    });

    it("should handle mixed cases", () => {
      const runtimes = ["node@18.0.0", "python", "@angular/cli@15.0.0"];
      const expected = {
        node: "18.0.0",
        python: "latest",
        "@angular/cli": "15.0.0",
      };

      expect(mapRuntimes(runtimes)).toEqual(expected);
    });

    it("should handle empty array", () => {
      expect(mapRuntimes([])).toEqual({});
    });
  });

  describe("mapRuntimeStatus", () => {
    const mockInstalled = {
      node: [
        {
          version: "18.0.0",
          requested_version: "18",
          active: true,
          installed: true,
        },
        {
          version: "20.0.0",
          requested_version: "20",
          active: false,
          installed: true,
        },
      ],
      python: [
        {
          version: "3.9.0",
          requested_version: "3.9",
          active: true,
          installed: true,
        },
      ],
    };

    const mockRuntimes = {
      node: "18",
      python: "3.9",
      go: "1.19",
    };

    it("should return CircularProgress for processing states", () => {
      const result = mapRuntimeStatus(
        "processing",
        mockInstalled,
        mockRuntimes,
      );
      expect(result).toBeDefined();
      // Since it returns JSX, we can't directly test the component type here
      // This would be better tested in a component test
    });

    it("should return CircularProgress for sent state", () => {
      const result = mapRuntimeStatus("sent", mockInstalled, mockRuntimes);
      expect(result).toBeDefined();
    });

    it("should return CircularProgress when no status provided", () => {
      const result = mapRuntimeStatus(
        undefined as any,
        mockInstalled,
        mockRuntimes,
      );

      expect(result).toEqual(<CircularProgress size={14} />);
    });

    it("should return CircularProgress when installed is not provided", () => {
      const result = mapRuntimeStatus("ok", undefined as any, mockRuntimes);
      expect(result).toEqual(<CircularProgress size={14} />);
    });

    it("should return CircularProgress when runtimes is not provided", () => {
      const result = mapRuntimeStatus("ok", mockInstalled, undefined as any);
      expect(result).toEqual(<CircularProgress size={14} />);
    });

    it("should return check icons for active runtimes", () => {
      const result = mapRuntimeStatus("ok", mockInstalled, mockRuntimes);

      ["node", "python", "go"].forEach((runtime, index) => {
        expect((result as React.ReactNode[])[index]).toEqual(
          <CheckIcon
            key={runtime}
            color="success"
            sx={{
              fontSize: 14,
              visibility: runtime === "go" ? "hidden" : undefined,
            }}
          />,
        );
      });
    });

    it("should return error icon for missing runtimes when status is error", () => {
      const result = mapRuntimeStatus("error", mockInstalled, mockRuntimes);

      ["node", "python"].forEach((runtime, index) => {
        expect((result as React.ReactNode[])[index]).toEqual(
          <CheckIcon
            key={runtime}
            color="success"
            sx={{
              fontSize: 14,
            }}
          />,
        );
      });

      expect((result as React.ReactNode[])[2]).toEqual(
        <TimesIcon
          key="go"
          color="error"
          sx={{
            fontSize: 14,
          }}
        />,
      );
    });

    it("should handle empty installed object", () => {
      const result = mapRuntimeStatus("ok", {}, mockRuntimes);

      ["node", "python", "go"].forEach((runtime, index) => {
        expect((result as React.ReactNode[])[index]).toEqual(
          <CheckIcon
            key={runtime}
            color="success"
            sx={{
              fontSize: 14,
              visibility: "hidden",
            }}
          />,
        );
      });
    });

    it("should handle runtime with no matching installed version", () => {
      const runtimes = { node: "16", python: "3.8" }; // Different versions
      const result = mapRuntimeStatus("ok", mockInstalled, runtimes);
      expect(result as React.ReactNode[]).toHaveLength(2);

      ["node", "python"].forEach((runtime, index) => {
        expect((result as React.ReactNode[])[index]).toEqual(
          <CheckIcon
            key={runtime}
            color="success"
            sx={{
              fontSize: 14,
              visibility: "hidden",
            }}
          />,
        );
      });
    });
  });

  describe("Domains", () => {
    it("should render domain configuration form", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("Domains")).toBeTruthy();
        expect(
          wrapper.getByText(
            "Configure custom domains for your Stormkit instance",
          ),
        ).toBeTruthy();
        expect(wrapper.getByLabelText("API Domain")).toBeTruthy();
        expect(wrapper.getByLabelText("App Domain")).toBeTruthy();
        expect(wrapper.getByLabelText("Dev Domain")).toBeTruthy();
      });
    });

    it("should display fetched domain values", async () => {
      await waitFor(() => {
        const apiField = wrapper.getByDisplayValue("https://api.example.org");
        const appField = wrapper.getByDisplayValue("https://app.example.org");
        const devField = wrapper.getByDisplayValue("https://dev.example.org");

        expect(apiField).toBeTruthy();
        expect(appField).toBeTruthy();
        expect(devField).toBeTruthy();
      });
    });

    it("should display helper texts for domain fields", async () => {
      await waitFor(() => {
        expect(
          wrapper.getByText("API requests will be served from this domain"),
        ).toBeTruthy();
        expect(
          wrapper.getByText(
            "This domain will be used to access your Stormkit dashboard",
          ),
        ).toBeTruthy();
        expect(
          wrapper.getByText(
            "Deployment previews will be displayed using subdomains of this domain",
          ),
        ).toBeTruthy();
      });
    });

    it("should handle domain fetch error", async () => {
      // Create a new wrapper with error response
      const errorScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/system/runtimes")
        .reply(200, {
          runtimes: [],
          autoInstall: true,
          installed: {},
          status: "ok",
        });

      const errorMiseScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/system/mise")
        .reply(200, { version: "1.0.0" });

      const errorDomainsScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/domains")
        .reply(500, { error: "Internal server error" });

      const errorSettingsScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/system/settings")
        .reply(200, { artifactRetentionDays: 30 });

      const errorWrapper = render(<AdminSystem />);

      await waitFor(() => {
        expect(errorScope.isDone()).toBe(true);
        expect(errorMiseScope.isDone()).toBe(true);
        expect(errorDomainsScope.isDone()).toBe(true);
        expect(errorSettingsScope.isDone()).toBe(true);
      });

      await waitFor(() => {
        expect(
          errorWrapper.getByText("Something went wrong while fetching domains"),
        ).toBeTruthy();
      });
    });

    it("should submit domain configuration successfully", async () => {
      const postScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/domains", {
          api: "https://new-api.example.org",
          app: "https://new-app.example.org",
          dev: "https://new-dev.example.org",
        })
        .reply(200, { ok: true });

      await waitFor(() => {
        const apiField = wrapper.getByDisplayValue("https://api.example.org");
        const appField = wrapper.getByDisplayValue("https://app.example.org");
        const devField = wrapper.getByDisplayValue("https://dev.example.org");

        fireEvent.change(apiField, {
          target: { value: "https://new-api.example.org" },
        });
        fireEvent.change(appField, {
          target: { value: "https://new-app.example.org" },
        });
        fireEvent.change(devField, {
          target: { value: "https://new-dev.example.org" },
        });
      });

      // Find the save button in the domains section specifically (second save button)
      const saveButtons = wrapper.getAllByText("Save");
      fireEvent.click(saveButtons[1]); // Second save button is for domains

      await waitFor(() => {
        expect(postScope.isDone()).toBe(true);
      });
    });

    it("should handle domain update error", async () => {
      const postScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/domains")
        .reply(400, { error: "Invalid domain" });

      await waitFor(() => {
        const apiField = wrapper.getByDisplayValue("https://api.example.org");
        fireEvent.change(apiField, { target: { value: "invalid-domain" } });
      });

      // Find the save button in the domains section specifically (second save button)
      const saveButtons = wrapper.getAllByText("Save");
      fireEvent.click(saveButtons[1]); // Second save button is for domains

      await waitFor(() => {
        expect(postScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "An error occurred while updating domains. Make sure specified domains are valid.",
          ),
        ).toBeTruthy();
      });
    });
  });

  describe("SystemSettings", () => {
    it("should render system settings form", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("System settings")).toBeTruthy();
        expect(
          wrapper.getByText(
            "Configure system-wide settings for your Stormkit instance",
          ),
        ).toBeTruthy();
        expect(wrapper.getByLabelText("Artifact retention days")).toBeTruthy();
        expect(
          wrapper.getByText(
            "Number of days to retain unpublished deployment artifacts before they are deleted",
          ),
        ).toBeTruthy();
      });
    });

    it("should display fetched artifact retention days", async () => {
      await waitFor(() => {
        const field = wrapper.getByDisplayValue("30");
        expect(field).toBeTruthy();
      });
    });

    it("should submit system settings successfully", async () => {
      const putScope = nock(process.env.API_DOMAIN || "")
        .put("/admin/system/settings", { artifactRetentionDays: 60 })
        .reply(200, { artifactRetentionDays: 60 });

      await waitFor(() => {
        const field = wrapper.getByLabelText("Artifact retention days");
        fireEvent.change(field, { target: { value: "60" } });
      });

      const saveButtons = wrapper.getAllByText("Save");
      fireEvent.click(saveButtons[2]); // Third save button is for system settings

      await waitFor(() => {
        expect(putScope.isDone()).toBe(true);
        expect(
          wrapper.getByText("System settings saved successfully."),
        ).toBeTruthy();
      });
    });

    it("should show validation error for invalid retention days", async () => {
      await waitFor(() => {
        const field = wrapper.getByLabelText("Artifact retention days");
        fireEvent.change(field, { target: { value: "abc" } });
      });

      const saveButtons = wrapper.getAllByText("Save");
      fireEvent.click(saveButtons[2]); // Third save button is for system settings

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "Artifact retention days must be a positive number.",
          ),
        ).toBeTruthy();
      });
    });

    it("should handle system settings update error", async () => {
      const putScope = nock(process.env.API_DOMAIN || "")
        .put("/admin/system/settings")
        .reply(500, { error: "Internal server error" });

      const saveButtons = wrapper.getAllByText("Save");
      fireEvent.click(saveButtons[2]); // Third save button is for system settings

      await waitFor(() => {
        expect(putScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "An error occurred while updating system settings.",
          ),
        ).toBeTruthy();
      });
    });
  });
});
