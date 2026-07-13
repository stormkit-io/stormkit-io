import type { Scope } from "nock";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import { fireEvent, type RenderResult, waitFor } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import {
  mockFetchFiles,
  mockRemoveFiles,
  mockToggleVisibility,
} from "~/testing/nocks/nock_volumes";
import VolumesDropzone from "./VolumesDropzone";
import { renderWithRouter } from "~/testing/helpers";

interface Props {
  openDialog?: { mode: string };
  loading?: boolean;
  files?: VolumeFile[];
}

describe("~/pages/apps/[id]/environments/[env-id]/volumes/VolumesDropzone.tsx", () => {
  let wrapper: RenderResult;
  let currentApp: App;
  let currentEnv: Environment;
  let fetchFilesScope: Scope;
  let setError: Mock;
  let setLoading: Mock;

  const createWrapper = ({ openDialog, loading, files = [] }: Props) => {
    setError = vi.fn();
    setLoading = vi.fn();
    currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    fetchFilesScope = mockFetchFiles({
      appId: currentApp.id,
      envId: currentEnv.id!,
      beforeId: "",
      response: { files },
    });

    wrapper = renderWithRouter({
      el: () => (
        <VolumesDropzone
          loading={loading}
          appId={currentApp.id}
          envId={currentEnv.id!}
          openDialog={openDialog}
          setError={setError}
          setLoading={setLoading}
        />
      ),
    });
  };

  describe("with no files", () => {
    beforeEach(() => {
      createWrapper({});
    });

    it("should mount dropzone", () => {
      expect(wrapper.getByTestId("volumes-dropzone")).toBeTruthy();
    });

    it("should display an empty message", () => {
      expect(wrapper.getByText("No files uploaded yet")).toBeTruthy();
    });
  });

  describe("while fetching files", () => {
    beforeEach(() => {
      createWrapper({
        loading: true,
      });
    });

    it("should mount dropzone", () => {
      expect(wrapper.getByTestId("volumes-dropzone")).toBeTruthy();
    });

    it("should not display an empty message", () => {
      expect(() => wrapper.getByText("No files uploaded yet")).toThrow();
    });
  });

  describe("with files", () => {
    const files: VolumeFile[] = [
      {
        name: "text.txt",
        size: 45010,
        id: "141",
        isPublic: false,
        createdAt: 1712418330,
      },
      {
        name: "index.js",
        size: 45010,
        id: "142",
        isPublic: true,
        createdAt: 1712418330,
        publicLink: "http://localhost/volumes/142",
      },
    ];

    beforeEach(() => {
      createWrapper({
        loading: false,
        files,
      });
    });

    it("should mount dropzone", () => {
      expect(wrapper.getByTestId("volumes-dropzone")).toBeTruthy();
    });

    it("should display an empty message first and then the files", async () => {
      expect(wrapper.getByText("No files uploaded yet")).toBeTruthy();
      expect(() => wrapper.getByText("test.txt")).toThrow();

      await waitFor(() => {
        expect(fetchFilesScope.isDone()).toBe(true);
        expect(setLoading).toHaveBeenCalledWith(false);
      });

      expect(wrapper.getByText("text.txt")).toBeTruthy();
      expect(wrapper.getByText("index.js")).toBeTruthy();
      expect(wrapper.getByText("http://localhost/volumes/142")).toBeTruthy();
    });

    it("should delete a file", async () => {
      await waitFor(() => {
        expect(wrapper.getByText("text.txt")).toBeTruthy();
      });

      fireEvent.click(wrapper.getAllByTestId("MoreHorizIcon").at(0)!);
      fireEvent.click(wrapper.getByText("Delete"));

      expect(wrapper.getByText(/This will remove/)).toBeTruthy();

      const scope = mockRemoveFiles({
        appId: currentApp.id,
        envId: currentEnv.id!,
        fileId: "141",
      });

      fetchFilesScope = mockFetchFiles({
        appId: currentApp.id,
        envId: currentEnv.id!,
        beforeId: "",
        response: { files: [] },
      });

      fireEvent.click(wrapper.getByText("Yes, continue"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);

        // Should trigger a new fetch
        expect(fetchFilesScope.isDone()).toBe(true);

        // Should close the modal
        expect(() => wrapper.getByText("Yes, continue")).toThrow();
      });
    });

    it.each`
      status       | fileIndex | buttonText        | expectedText
      ${"private"} | ${0}      | ${"Make public"}  | ${/a public file. Anyone with the link will be able to see the file/}
      ${"public"}  | ${1}      | ${"Make private"} | ${/ Only authenticated users will be able to access the file./}
    `(
      "should make a file $status",
      async ({ status, fileIndex, buttonText, expectedText }) => {
        await waitFor(() => {
          expect(wrapper.getByText("text.txt")).toBeTruthy();
        });

        fireEvent.click(wrapper.getAllByTestId("MoreHorizIcon").at(fileIndex)!);
        fireEvent.click(wrapper.getByText(buttonText));

        expect(wrapper.getByText(expectedText)).toBeTruthy();

        const scope = mockToggleVisibility({
          appId: currentApp.id,
          envId: currentEnv.id!,
          fileId: files[fileIndex].id,
          visibility: status === "private" ? "public" : "private",
        });

        fetchFilesScope = mockFetchFiles({
          appId: currentApp.id,
          envId: currentEnv.id!,
          beforeId: "",
          response: { files: [] },
        });

        fireEvent.click(wrapper.getByText("Yes, continue"));

        await waitFor(() => {
          expect(scope.isDone()).toBe(true);

          // Should trigger a new fetch
          expect(fetchFilesScope.isDone()).toBe(true);

          // Should close the modal
          expect(() => wrapper.getByText("Yes, continue")).toThrow();
        });
      }
    );
  });
});
