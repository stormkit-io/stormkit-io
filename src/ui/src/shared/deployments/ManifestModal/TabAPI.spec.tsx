import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, waitFor, render } from "@testing-library/react";
import mockManifest from "~/testing/data/mock_deployment_manifest";
import TabAPI from "./TabAPI";

interface Props {
  manifest?: Manifest;
  previewEndpoint?: string;
}

describe("~/apps/[id]/environments/[env-id]/deployments/_components/ManifestModal/TabAPI.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({
    previewEndpoint = "",
    manifest = mockManifest(),
  }: Props | undefined = {}) => {
    wrapper = render(
      <TabAPI
        previewEndpoint={previewEndpoint}
        manifest={manifest}
        apiPathPrefix="/api"
      />
    );
  };

  it("should display an info message when api is not enabled", () => {
    createWrapper({ manifest: mockManifest({ apiFiles: [] }) });

    expect(wrapper.getByText(/REST API not detected\./)).toBeTruthy();
    expect(wrapper.getByText(/Create a top-level/)).toBeTruthy();
    expect(wrapper.getByText("/api")).toBeTruthy();
    expect(wrapper.getByText(/folder to get started/)).toBeTruthy();
    expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
      "https://www.stormkit.io/docs/features/writing-api"
    );
  });

  it("should list api endpoints", () => {
    createWrapper({
      manifest: mockManifest({
        apiFiles: [
          { fileName: "/index.js" },
          { fileName: "/user/[id]/list.post.js" },
        ],
      }),
    });

    expect(wrapper.getByText("/api")).toBeTruthy();
    expect(wrapper.getByText("ALL")).toBeTruthy();
    expect(wrapper.getByText("POST")).toBeTruthy();
    expect(wrapper.getByText("/api/user/:id/list")).toBeTruthy();
  });

  it("should copy curl commands", async () => {
    const writeText = vi.fn();

    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText,
      },
    });

    createWrapper({
      manifest: mockManifest({
        apiFiles: [
          { fileName: "/index.js" },
          { fileName: "/user/[id]/list.post.js" },
        ],
      }),
    });

    fireEvent.click(wrapper.getAllByLabelText("Copy CURL").at(1)!);

    await waitFor(() => {
      expect(wrapper.getByLabelText("Copied!")).toBeTruthy();
    });

    expect(writeText).toHaveBeenCalledWith(`curl -X POST "/api/user/:id/list"`);
  });
});
