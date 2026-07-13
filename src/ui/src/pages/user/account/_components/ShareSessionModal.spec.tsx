import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, it, vi, type Mock } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { RootContext } from "~/pages/Root.context";
import { mockCreateSharedSession } from "~/testing/nocks/nock_user";
import ShareSessionModal from "./ShareSessionModal";

interface Props {
  isEnterprise?: boolean;
}

describe("~/pages/user/account/_components/ShareSessionModal", () => {
  let onClose: Mock;
  let wrapper: RenderResult;

  const createWrapper = ({ isEnterprise = false }: Props = {}) => {
    onClose = vi.fn();

    const details: InstanceDetails | undefined = isEnterprise
      ? { license: { seats: 10, remaining: 10, edition: "enterprise" } }
      : undefined;

    wrapper = render(
      <RootContext.Provider value={{ mode: "dark", setMode: () => {}, details }}>
        <ShareSessionModal onClose={onClose} />
      </RootContext.Provider>
    );
  };

  describe("non-enterprise user", () => {
    beforeEach(() => {
      createWrapper({ isEnterprise: false });
    });

    it("should not show the duration selector", () => {
      expect(wrapper.queryByLabelText("Session duration")).toBeNull();
    });

    it("should create a session with 1h and display the CopyBox snippet", async () => {
      const scope = mockCreateSharedSession({ payload: { hours: 1 } });

      fireEvent.click(wrapper.getByText("Create session"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "Copy this snippet and run it in the browser console of the target instance. This token expires in 1h:"
          )
        ).toBeTruthy();
        expect(
          wrapper.getByDisplayValue(
            "localStorage.setItem('skit_token', JSON.stringify('mock-session-token'))"
          )
        ).toBeTruthy();
      });
    });
  });

  describe("enterprise user", () => {
    beforeEach(() => {
      createWrapper({ isEnterprise: true });
    });

    it("should show the duration selector", () => {
      expect(wrapper.getByLabelText("Session duration")).toBeTruthy();
    });

    it("should create a session with the selected duration", async () => {
      const scope = mockCreateSharedSession({ payload: { hours: 6 } });

      fireEvent.mouseDown(wrapper.getByLabelText("Session duration"));
      const option = await waitFor(() => wrapper.getByText("6h"));
      fireEvent.click(option);

      fireEvent.click(wrapper.getByText("Create session"));

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            "Copy this snippet and run it in the browser console of the target instance. This token expires in 6h:"
          )
        ).toBeTruthy();
      });
    });

    it("should use the skit_token localStorage key in the snippet", async () => {
      mockCreateSharedSession({ payload: { hours: 1 } });

      fireEvent.click(wrapper.getByText("Create session"));

      await waitFor(() => {
        expect(
          wrapper.getByDisplayValue(
            "localStorage.setItem('skit_token', JSON.stringify('mock-session-token'))"
          )
        ).toBeTruthy();
      });
    });
  });

  describe("cancel button", () => {
    beforeEach(() => {
      createWrapper();
    });

    it("should call onClose when cancel is clicked", () => {
      fireEvent.click(wrapper.getByText("Cancel"));
      expect(onClose).toHaveBeenCalledOnce();
    });
  });
});
