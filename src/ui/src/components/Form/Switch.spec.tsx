import { render, RenderResult } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import SwitchForm from "./Switch";

interface Props {
  name: string;
  label: string;
  checked: boolean;
  setChecked: (val: boolean) => void;
  description: string;
}

describe("~/components/Form/Switch.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = (props: Props) => {
    wrapper = render(<SwitchForm {...props} />);
  };

  const findSwitchInput = () => wrapper.getByRole("switch") as HTMLInputElement;

  it("should render the label and description", () => {
    const setChecked = vi.fn();

    createWrapper({
      name: "test-switch",
      label: "Enable feature",
      checked: false,
      setChecked,
      description: "This enables the feature",
    });

    expect(wrapper.getByText("Enable feature")).toBeTruthy();
    expect(wrapper.getByText("This enables the feature")).toBeTruthy();
  });

  it("should render with checked state", () => {
    const setChecked = vi.fn();

    createWrapper({
      name: "test-switch",
      label: "Enable feature",
      checked: true,
      setChecked,
      description: "This enables the feature",
    });

    expect(findSwitchInput().checked).toBe(true);
  });

  it("should call setChecked when toggled", async () => {
    const setChecked = vi.fn();

    createWrapper({
      name: "test-switch",
      label: "Enable feature",
      checked: false,
      setChecked,
      description: "This enables the feature",
    });

    await userEvent.click(findSwitchInput());

    expect(setChecked).toHaveBeenCalledWith(true);
  });
});
