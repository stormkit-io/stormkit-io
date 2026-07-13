import type { RenderResult } from "@testing-library/react";
import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { LocalStorage } from "~/utils/storage";
import { RootContext } from "../Root.context";
import UpdateSnackbar from "./UpdateSnackbar";

interface WrapperProps {
  details?: InstanceDetails;
}

describe("pages/auth/UpdateSnackbar.tsx", () => {
  let wrapper: RenderResult;

  const apiVersion = "v1.7.30";
  const apiCommit = "a4ee052";

  const createWrapper = ({ details }: WrapperProps) => {
    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: () => {},
          details,
          loading: false,
        }}
      >
        <UpdateSnackbar />
      </RootContext.Provider>
    );

    return wrapper;
  };

  describe("when self-hosted", () => {
    afterEach(() => {
      LocalStorage.del("STORMKIT_UPDATE");
      delete process.env.GIT_HASH;
    });

    describe("when api needs an update", () => {
      beforeEach(() => {
        LocalStorage.set("STORMKIT_UPDATE", "my-version");

        createWrapper({
          details: {
            latest: { apiVersion: "v1.8.35" },
            stormkit: { edition: "self-hosted", apiCommit, apiVersion },
            update: { api: true },
          },
        });
      });

      it("displays the snackbar", async () => {
        await waitFor(() => {
          expect(wrapper.container.textContent).toBe(
            "Stormkit API has a newer version"
          );
        });
      });
    });

    describe("when previously dismissed", () => {
      beforeEach(() => {
        LocalStorage.set("STORMKIT_UPDATE", apiVersion);

        createWrapper({
          details: {
            latest: { apiVersion },
            stormkit: { edition: "self-hosted", apiCommit, apiVersion },
          },
        });
      });

      it("does not display the snackbar", () => {
        expect(wrapper.container.innerHTML).toBe("");
      });
    });
  });

  describe("when cloud", () => {
    afterEach(() => {
      LocalStorage.del("STORMKIT_UPDATE");
      delete process.env.GIT_HASH;
    });

    describe("when both api and ui needs an update", () => {
      beforeEach(() => {
        createWrapper({
          details: {
            stormkit: { edition: "self-hosted", apiCommit, apiVersion },
          },
        });
      });

      it("does not display the snackbar", () => {
        expect(wrapper.container.innerHTML).toBe("");
      });
    });
  });
});
