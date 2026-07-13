import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import { type RenderResult, render } from "@testing-library/react";
import MyDropzone from "./Dropzone";

interface Props {
  openDialog?: { mode: string };
  loading?: boolean;
  files?: File[];
  clickToOpen?: boolean;
  showDropZone?: boolean;
}

describe("~/components/Dropzone.tsx", () => {
  let wrapper: RenderResult;
  let onDrop: Mock;

  const createWrapper = ({
    showDropZone,
    clickToOpen,
    openDialog,
    files = [],
  }: Props) => {
    onDrop = vi.fn();
    wrapper = render(
      <MyDropzone
        openDialog={openDialog}
        onDrop={onDrop}
        files={files}
        showDropZone={showDropZone}
        clickToOpen={clickToOpen}
      />
    );
  };

  describe("with no files and showDropZone = true ", () => {
    beforeEach(() => {
      createWrapper({ showDropZone: true, clickToOpen: true });
    });

    it("should mount dropzone", () => {
      expect(wrapper.getByTestId("my-dropzone")).toBeTruthy();
    });

    it("should display an empty message", () => {
      expect(wrapper.getByText("No files uploaded yet")).toBeTruthy();
    });

    it("should display a click here message", () => {
      expect(wrapper.getByText("Click or drop here")).toBeTruthy();
    });
  });

  describe("with no files and showDropZone = false ", () => {
    beforeEach(() => {
      createWrapper({ showDropZone: false });
    });

    it("should mount dropzone", () => {
      expect(wrapper.getByTestId("my-dropzone")).toBeTruthy();
    });

    it("should not display an empty message", () => {
      expect(() => wrapper.getByText("No files uploaded yet")).toThrow();
    });

    it("should not display a drop here message", () => {
      expect(() => wrapper.getByText("Drop here")).toThrow();
    });
  });
});
