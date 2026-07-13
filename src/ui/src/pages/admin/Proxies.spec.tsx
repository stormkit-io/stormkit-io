import { describe, it, expect } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import nock from "nock";
import Proxies from "./Proxies";

const apiDomain = process.env.API_DOMAIN || "";

const fetchProxiesScope = (
  rules: Record<
    string,
    { target: string; headers?: Record<string, string> }
  > = {}
) => {
  return nock(apiDomain)
    .get("/admin/system/proxies")
    .reply(200, { proxies: { rules } });
};

const putProxiesScope = (
  payload: Record<string, { target?: string; headers: Record<string, string> }>
) => {
  return nock(apiDomain)
    .put("/admin/system/proxies", { proxies: payload })
    .reply(200, { ok: true });
};

describe("~/pages/admin/Proxies.tsx", () => {
  it("renders with empty state", async () => {
    const scope = fetchProxiesScope({});
    const { getByText } = render(<Proxies />);

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(getByText("Reverse proxies")).toBeTruthy();
      expect(getByText("New proxy")).toBeTruthy();
      expect(getByText("No proxy configured yet.")).toBeTruthy();
    });
  });

  it("renders existing proxies", async () => {
    const scope = fetchProxiesScope({
      "example.com": { target: "http://localhost:3000" },
      "api.example.org": { target: "https://api.service.tld" },
    });

    const { getByText, queryByText } = render(<Proxies />);

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(queryByText("No proxy configured yet.")).toBeNull();
      expect(getByText(/example.com/)).toBeTruthy();
      expect(getByText(/http:\/\/localhost:3000/)).toBeTruthy();
      expect(getByText(/api.example.org/)).toBeTruthy();
      expect(getByText(/https:\/\/api.service.tld/)).toBeTruthy();
    });
  });

  it("opens modal, validates fields, and creates a new proxy (normalizes domain)", async () => {
    // Initial fetch (empty)
    const scopeInitial = fetchProxiesScope({});

    const { getByText, getByLabelText, queryByLabelText } = render(<Proxies />);

    await waitFor(() => {
      expect(scopeInitial.isDone()).toBe(true);
    });

    // Open modal
    fireEvent.click(getByText("New proxy"));

    // Try to save with empty fields
    fireEvent.click(getByText("Save"));

    await waitFor(() => {
      expect(getByText("From field is required")).toBeTruthy();
    });

    // Fill only From
    await userEvent.type(getByLabelText("From"), "my.example.org");

    fireEvent.click(getByText("Save"));

    await waitFor(() => {
      expect(getByText("Target field is required")).toBeTruthy();
    });

    // Invalid target
    await userEvent.type(getByLabelText("Target"), "localhost:3000");

    fireEvent.click(getByText("Save"));

    await waitFor(() => {
      expect(
        getByText(
          "Target must be a fully qualified URL (e.g. http://example.com)"
        )
      ).toBeTruthy();
    });

    await userEvent.clear(getByLabelText("From"));
    await userEvent.clear(getByLabelText("Target"));

    // Valid data with protocol in From (should normalize to domain only)
    await userEvent.type(getByLabelText("From"), "my.example.org");
    await userEvent.type(getByLabelText("Target"), "http://localhost:3000");

    const scopePut = putProxiesScope({
      "my.example.org": {
        target: "http://localhost:3000",
        headers: {},
      },
    });

    // After success, component refetches proxies
    const scopeRefetch = fetchProxiesScope({
      "my.example.org": { target: "http://localhost:3000", headers: {} },
    });

    fireEvent.click(getByText("Save"));

    await waitFor(() => {
      expect(scopePut.isDone()).toBe(true);
      expect(scopeRefetch.isDone()).toBe(true);
    });

    // Modal closed, list updated
    await waitFor(() => {
      expect(queryByLabelText("From")).toBeNull();
      expect(getByText(/my.example.org/)).toBeTruthy();
      expect(getByText(/http:\/\/localhost:3000/)).toBeTruthy();
    });
  });
});
