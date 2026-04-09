import type { Scope } from "nock";
import { RenderResult, waitFor } from "@testing-library/react";
import { fireEvent, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  mockFetchAPIKeys,
  mockGenerateAPIKey,
  mockDeleteAPIKey,
} from "~/testing/nocks/nock_api_keys";
import APIKeyList from "./APIKeyList";

const envId = "1429333243019";

const defaultProps = {
  subtitle: "Manage your API keys.",
  emptyMessage: "No API keys found.",
  apiKeyProps: { envId, scope: "env" as const },
};

type WrapperProps = Partial<typeof defaultProps> & { title?: string };

describe("~/shared/api-keys/APIKeyList.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;

  const createWrapper = (props: WrapperProps = {}) => {
    fetchScope = mockFetchAPIKeys({ envId });

    wrapper = render(<APIKeyList {...defaultProps} {...props} />);
  };

  describe("listing keys", () => {
    test("renders the subtitle and key names after fetching", async () => {
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByText(defaultProps.subtitle)).toBeTruthy();
        expect(wrapper.getByText("Default")).toBeTruthy();
        expect(wrapper.getByText("CI")).toBeTruthy();
      });
    });

    test("uses the default title 'API Keys'", async () => {
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByText("API Keys")).toBeTruthy();
      });
    });

    test("uses a custom title when provided", async () => {
      createWrapper({ title: "Environment Keys" });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByText("Environment Keys")).toBeTruthy();
      });
    });

    test("shows the empty message when there are no keys", async () => {
      fetchScope = mockFetchAPIKeys({ envId, response: { keys: [] } });
      wrapper = render(<APIKeyList {...defaultProps} />);

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByText(defaultProps.emptyMessage)).toBeTruthy();
      });
    });

    test("shows an error message when the fetch fails", async () => {
      fetchScope = mockFetchAPIKeys({ envId, status: 500 });
      wrapper = render(<APIKeyList {...defaultProps} />);

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "An error occurred while fetching your API key. Please try again later.",
          ),
        ).toBeTruthy();
      });
    });
  });

  describe("creating a new key", () => {
    test("opens the modal, submits, then shows a one-time token alert", async () => {
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      fireEvent.click(wrapper.getByText("New API Key"));

      await waitFor(() => {
        expect(wrapper.getByText("Generate New Key")).toBeTruthy();
      });

      const generateScope = mockGenerateAPIKey({
        name: "My Token",
        scope: "env",
        envId,
      });

      await userEvent.type(wrapper.getByLabelText("API Key name"), "My Token");
      fireEvent.click(wrapper.getByText("Create"));

      await waitFor(() => {
        expect(generateScope.isDone()).toBe(true);

        // Modal should be closed
        expect(() => wrapper.getByText("Generate New Key")).toThrow();

        // One-time token alert must be visible
        expect(
          wrapper.getByText(
            "Make sure to copy your new API key now. It won't be shown again.",
          ),
        ).toBeTruthy();

        expect(
          wrapper.getByDisplayValue(
            "SK_newtoken1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdefghij",
          ),
        ).toBeTruthy();
      });
    });

    test("dismissing the alert removes the token from the page", async () => {
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      fireEvent.click(wrapper.getByText("New API Key"));

      const generateScope = mockGenerateAPIKey({
        name: "Temp",
        scope: "env",
        envId,
      });

      await userEvent.type(wrapper.getByLabelText("API Key name"), "Temp");
      fireEvent.click(wrapper.getByText("Create"));

      await waitFor(() => {
        expect(generateScope.isDone()).toBe(true);
      });

      const token =
        "SK_newtoken1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdefghij";

      await waitFor(() => {
        expect(wrapper.getByDisplayValue(token)).toBeTruthy();
      });

      // Close the alert
      fireEvent.click(wrapper.getByLabelText("Close"));

      await waitFor(() => {
        expect(() => wrapper.getByDisplayValue(token)).toThrow();
      });
    });

    test("copies the token to the clipboard when the copy button is clicked", async () => {
      const execCommand = vi.fn().mockReturnValue(true);
      document.execCommand = execCommand;

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      fireEvent.click(wrapper.getByText("New API Key"));

      const generateScope = mockGenerateAPIKey({
        name: "Copy Test",
        scope: "env",
        envId,
      });

      await userEvent.type(wrapper.getByLabelText("API Key name"), "Copy Test");
      fireEvent.click(wrapper.getByText("Create"));

      await waitFor(() => {
        expect(generateScope.isDone()).toBe(true);
      });

      const token =
        "SK_newtoken1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdefghij";

      await waitFor(() => {
        expect(wrapper.getByDisplayValue(token)).toBeTruthy();
      });

      fireEvent.click(wrapper.getByLabelText("Copy to clipboard"));

      expect(execCommand).toHaveBeenCalledWith("copy");
    });

    test("shows an error in the modal when creation fails with a 400", async () => {
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      fireEvent.click(wrapper.getByText("New API Key"));

      const generateScope = mockGenerateAPIKey({
        name: "Bad",
        scope: "env",
        envId,
        status: 400,
        response: { error: "Key name is a required field." } as any,
      });

      await userEvent.type(wrapper.getByLabelText("API Key name"), "Bad");
      fireEvent.click(wrapper.getByText("Create"));

      await waitFor(() => {
        expect(generateScope.isDone()).toBe(true);
      });

      await waitFor(() => {
        expect(wrapper.getByText("Key name is a required field.")).toBeTruthy();
      });
    });
  });

  describe("deleting a key", () => {
    test("shows a confirm modal then removes the key and refetches", async () => {
      createWrapper();

      await waitFor(() => {
        expect(wrapper.getByText("CI")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByLabelText("expand-9868814106"));
      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Confirm action")).toBeTruthy();
      });

      const deleteScope = mockDeleteAPIKey({ keyId: "9868814106" });
      const fetchScope2 = mockFetchAPIKeys({ envId, response: { keys: [] } });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(deleteScope.isDone()).toBe(true);
        expect(fetchScope2.isDone()).toBe(true);
        expect(() => wrapper.getByText("Confirm action")).toThrow();
        expect(() => wrapper.getByText("CI")).toThrow();
      });
    });

    test("shows an error when deletion fails", async () => {
      createWrapper();

      await waitFor(() => {
        expect(wrapper.getByText("CI")).toBeTruthy();
      });

      fireEvent.click(wrapper.getByLabelText("expand-9868814106"));
      fireEvent.click(wrapper.getByText("Delete"));

      await waitFor(() => {
        expect(wrapper.getByText("Confirm action")).toBeTruthy();
      });

      mockDeleteAPIKey({ keyId: "9868814106", status: 500 });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(
          wrapper.getByText("Something went wrong while deleting the API key."),
        ).toBeTruthy();
      });
    });
  });
});
