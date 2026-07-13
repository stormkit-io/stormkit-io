import type { RenderResult } from "@testing-library/react";
import type { Mock } from "vitest";
import { describe, expect, beforeEach, it, vi } from "vitest";
import { fireEvent, waitFor, render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { WelcomeModal } from "./";

interface Props {
  isOpen: boolean;
  toggleModal: Mock;
}

describe("~/pages/apps/_components/WelcomeModal", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ isOpen, toggleModal }: Props) => {
    wrapper = render(
      <MemoryRouter>
        <WelcomeModal
          welcomeModalId="welcome-modal-id"
          isOpen={isOpen}
          toggleModal={toggleModal}
        />
      </MemoryRouter>
    );
  };

  describe("when is open", () => {
    let toggleModal: Mock;

    beforeEach(() => {
      toggleModal = vi.fn();
      createWrapper({ isOpen: true, toggleModal });
    });

    it("closes the modal when close button is clicked", async () => {
      fireEvent.click(wrapper.getByText("Close"));

      await waitFor(() => {
        expect(toggleModal).toHaveBeenCalledWith(false);
      });
    });
  });
});
