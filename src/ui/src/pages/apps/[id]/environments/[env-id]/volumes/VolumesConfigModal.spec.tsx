import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import {
  fireEvent,
  render,
  waitFor,
  getAllByText,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { mockSetVolumesConfig } from "~/testing/nocks/nock_volumes";
import VolumesConfigModal from "./VolumesConfigModal";

interface Props {
  config?: VolumeConfig;
}

describe("~/pages/apps/[id]/environments/[env-id]/volumes/VolumesConfigModal.tsx", () => {
  let wrapper: RenderResult;
  let onClose: Mock;
  let onSuccess: Mock;

  const createWrapper = ({ config }: Props) => {
    onClose = vi.fn();
    onSuccess = vi.fn();

    wrapper = render(
      <VolumesConfigModal
        onClose={onClose}
        onSuccess={onSuccess}
        config={config}
      />
    );
  };

  describe("when volumes is not yet configured", () => {
    beforeEach(() => {
      createWrapper({});
    });

    it("should display filesys as default selected value", () => {
      expect(wrapper.getByText("File System")).toBeTruthy();
      expect(wrapper.getByDisplayValue("/shared/volumes")).toBeTruthy();
    });

    it("should display a learn more button", () => {
      expect(wrapper.getByText("Learn more").getAttribute("href")).toBe(
        "https://www.stormkit.io/docs/features/volumes#filesys"
      );
    });

    it("should save save the config", async () => {
      const scope = mockSetVolumesConfig({
        config: { mountType: "filesys", rootPath: "/shared/volumes" },
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(onSuccess).toHaveBeenCalled();
      });
    });
  });

  describe("when volumes is already configured", () => {
    beforeEach(() => {
      createWrapper({
        config: { mountType: "filesys", rootPath: "/mnt/stormkit" },
      });
    });

    it("should display the pre-filled configuration", () => {
      expect(wrapper.getByText("File System")).toBeTruthy();
      expect(wrapper.getByDisplayValue("/mnt/stormkit")).toBeTruthy();
    });
  });

  describe("aws config", () => {
    const findDropdown = () => wrapper.getByRole("combobox");
    const findOption = (text: string) =>
      getAllByText(document.body, text).at(0);

    const openDropdown = async () => {
      let selector;

      await waitFor(() => {
        selector = findDropdown();
      });

      if (!selector) {
        throw new Error("Cannot find dropdown");
      }

      expect(selector).toBeTruthy();
      fireEvent.mouseDown(selector);
      return selector;
    };

    beforeEach(async () => {
      createWrapper({});
      await openDropdown();
      fireEvent.click(findOption("AWS S3")!);
    });

    it("should submit the form", async () => {
      expect(wrapper.getByLabelText("Region")).toBeTruthy();
      expect(wrapper.getByLabelText("Bucket name")).toBeTruthy();
      expect(wrapper.getByLabelText("Access key")).toBeTruthy();
      expect(wrapper.getByLabelText("Secret key")).toBeTruthy();
      expect(() => wrapper.getByLabelText("Root path")).toThrow();

      await userEvent.clear(wrapper.getByLabelText("Region"));
      await userEvent.type(wrapper.getByLabelText("Region"), "eu-central-1");
      await userEvent.type(wrapper.getByLabelText("Bucket name"), "my-bucket");
      await userEvent.type(wrapper.getByLabelText("Access key"), "hello-world");
      await userEvent.type(wrapper.getByLabelText("Secret key"), "hi-world");

      const scope = mockSetVolumesConfig({
        config: {
          mountType: "aws:s3",
          region: "eu-central-1",
          accessKey: "hello-world",
          secretKey: "hi-world",
          bucketName: "my-bucket",
        },
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(onSuccess).toHaveBeenCalled();
      });
    });
  });
});
