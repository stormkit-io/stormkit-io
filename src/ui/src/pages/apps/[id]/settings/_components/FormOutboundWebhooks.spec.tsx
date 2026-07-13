import { describe, beforeEach, it, expect } from "vitest";
import type { OutboundWebhook } from "../types";
import { fireEvent, RenderResult } from "@testing-library/react";
import { waitFor, render } from "@testing-library/react";
import mockOutboundWebhooks from "~/testing/data/mock_outbound_webhooks";
import mockApp from "~/testing/data/mock_app";
import {
  mockFetchOutboundWebhooks,
  mockDeleteOutboundWebhook,
} from "~/testing/nocks/nock_outbound_webhooks";
import FormOutboundWebhooks from "./FormOutboundWebhooks";

interface Props {
  app: App;
}

describe("~/pages/apps/[id]/settings/_components/FormOutboundWebhooks", () => {
  let currentApp: App;
  let currentOWHs: OutboundWebhook[];
  let wrapper: RenderResult;

  const createWrapper = ({ app }: Props) => {
    wrapper = render(<FormOutboundWebhooks app={app} />);
  };

  beforeEach(() => {
    currentApp = mockApp();
    currentOWHs = mockOutboundWebhooks();

    mockFetchOutboundWebhooks({
      appId: currentApp.id,
      status: 200,
      response: { webhooks: currentOWHs },
    });

    createWrapper({ app: currentApp });
  });

  it("the button is at loading state initially", () => {
    expect(wrapper.getByLabelText("Add new webhook").innerHTML).toContain(
      "<svg"
    );
  });

  it("the button is not at loading state when the query has loaded", async () => {
    await waitFor(() => {
      expect(wrapper.getByLabelText("Add new webhook").innerHTML).not.toContain(
        "spinner"
      );
    });
  });

  it("should handle webhook deletion", async () => {
    await waitFor(() => {
      expect(wrapper.getAllByLabelText("expand")).toHaveLength(2);
    });

    await fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);

    await waitFor(() => {
      expect(wrapper.getByText("Delete")).toBeTruthy();
    });

    const deleteScope = mockDeleteOutboundWebhook({
      appId: currentApp.id,
      whId: currentOWHs[0].id!,
    });

    // Should re-fetch the webhooks after deletion
    const fetchScope = mockFetchOutboundWebhooks({
      appId: currentApp.id,
      status: 200,
      response: { webhooks: currentOWHs },
    });

    await fireEvent.click(wrapper.getByText("Delete"));

    await waitFor(() => {
      expect(deleteScope.isDone()).toBe(true);
      expect(fetchScope.isDone()).toBe(true);
    });
  });

  it("clicking add new webhook opens a modal", async () => {
    await waitFor(() => {
      expect(
        wrapper.getAllByText("Triggered after a deployment is published")
      ).toBeTruthy();
    });

    fireEvent.click(wrapper.getByText("Add new webhook"));

    await waitFor(() => {
      expect(wrapper.getAllByText("Create an outbound webhook")).toHaveLength(
        1
      );
    });
  });

  it.each`
    description    | endpoint
    ${"short url"} | ${"https://discord.com/example/endpoint"}
    ${"long url"}  | ${"https://discord.com/api/webhooks/example/endpoint..."}
  `("lists the hooks: $description", async ({ endpoint }) => {
    await waitFor(() => {
      expect(wrapper.getByText(endpoint)).toBeTruthy();
    });
  });
});
