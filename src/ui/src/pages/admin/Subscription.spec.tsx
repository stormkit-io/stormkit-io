import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import AdminSubscription from "./Subscription";
import { RootContext } from "../Root.context";

interface Props {
  details: InstanceDetails;
}

describe("~/pages/admin/Subscription.tsx", () => {
  let wrapper: RenderResult;
  let mockSetRefreshToken = vi.fn();

  const createWrapper = ({ details }: Props) => {
    mockSetRefreshToken = vi.fn();
    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details,
          setRefreshToken: mockSetRefreshToken,
        }}
      >
        <AdminSubscription />
      </RootContext.Provider>
    );
  };

  it("should display correct settings for premium plans", async () => {
    createWrapper({
      details: {
        license: {
          seats: 15,
          remaining: 12,
          edition: "enterprise",
        },
      },
    });

    await waitFor(() => {
      expect(wrapper.getByText("Enterprise")).toBeTruthy();
      expect(wrapper.getByText("15")).toBeTruthy();
      expect(wrapper.getByText("12")).toBeTruthy();
    });
  });

  it("should display correct settings for free plans", async () => {
    createWrapper({
      details: {
        license: {
          seats: -1,
          remaining: -1,
          edition: "community",
        },
      },
    });

    await waitFor(() => {
      expect(wrapper.getByText("Community")).toBeTruthy();
      expect(wrapper.getAllByText("Unlimited")).toHaveLength(2);
    });
  });

  describe("License submission", () => {
    beforeEach(() => {
      nock.cleanAll();
    });

    afterEach(() => {
      nock.cleanAll();
    });

    it("should successfully activate a new license", async () => {
      nock(process.env.API_DOMAIN || "")
        .post("/admin/license", { key: "valid-license-key" })
        .reply(200, { seats: 50, edition: "enterprise" });

      createWrapper({
        details: {
          license: {
            seats: -1,
            remaining: -1,
            edition: "community",
          },
        },
      });

      const licenseInput = wrapper.getByLabelText("License key");
      const submitButton = wrapper.getByRole("button", { name: "Activate" });

      // Fill in the license key
      fireEvent.change(licenseInput, {
        target: { value: "valid-license-key" },
      });

      // Submit the form
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(
          wrapper.getByText("License activated successfully.")
        ).toBeTruthy();
        expect(mockSetRefreshToken).toHaveBeenCalledWith(expect.any(Number));
      });
    });

    it("should show error for invalid license key", async () => {
      nock(process.env.API_DOMAIN || "")
        .post("/admin/license", { key: "invalid-license-key" })
        .reply(400, { error: "Invalid license key" });

      createWrapper({
        details: {
          license: {
            seats: -1,
            remaining: -1,
            edition: "community",
          },
        },
      });

      const licenseInput = wrapper.getByLabelText("License key");
      const submitButton = wrapper.getByRole("button", { name: "Activate" });

      // Fill in an invalid license key
      fireEvent.change(licenseInput, {
        target: { value: "invalid-license-key" },
      });

      // Submit the form
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(
          wrapper.getByText("The license is either invalid or expired.")
        ).toBeTruthy();

        expect(mockSetRefreshToken).not.toHaveBeenCalled();
      });
    });

    it("should successfully remove an enterprise license", async () => {
      nock(process.env.API_DOMAIN || "")
        .post("/admin/license", {})
        .reply(200, { seats: 20, edition: "community" });

      createWrapper({
        details: {
          license: {
            seats: 50,
            remaining: 30,
            edition: "enterprise",
          },
        },
      });

      const removeButton = wrapper.getByRole("button", {
        name: "Remove license",
      });

      // Click remove license
      fireEvent.click(removeButton);

      await waitFor(() => {
        expect(wrapper.getByText("License removed successfully.")).toBeTruthy();
        expect(mockSetRefreshToken).toHaveBeenCalledWith(expect.any(Number));
      });
    });

    it("should disable remove button for community edition", async () => {
      createWrapper({
        details: {
          license: {
            seats: -1,
            remaining: -1,
            edition: "community",
          },
        },
      });

      const removeButton = wrapper.getByRole("button", {
        name: "Remove license",
      }) as HTMLButtonElement;

      expect(removeButton.disabled).toBe(true);
    });
  });
});
