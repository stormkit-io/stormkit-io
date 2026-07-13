import type { RenderResult } from "@testing-library/react";
import type { Mock } from "vitest";
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import DiffModal from "./DiffModal";

interface Props {
  oldDiff?: DiffFields;
  newDiff?: DiffFields;
}

vi.mock("@codemirror/lang-json", () => ({ json: vi.fn() }));

describe("~/shared/feed/AuditMessage.tsx", () => {
  let wrapper: RenderResult;
  let onClose: Mock;

  const createWrapper = ({ oldDiff = {}, newDiff = {} }: Props) => {
    onClose = vi.fn();

    const audit: Audit = {
      id: "4",
      action: "UPDATE:SNIPPET",
      diff: {
        new: newDiff,
        old: oldDiff,
      },
    };

    wrapper = render(<DiffModal onClose={onClose} audit={audit} />);
  };

  it("should display both new and old diffs", () => {
    createWrapper({
      newDiff: { snippetTitle: "new-field" },
      oldDiff: { snippetTitle: "old-field" },
    });

    expect(wrapper.getByText("Old version")).toBeTruthy();
    expect(wrapper.getByText("New version")).toBeTruthy();
  });

  it("should display only new diff", () => {
    createWrapper({
      newDiff: { snippetTitle: "new-field" },
    });

    expect(() => wrapper.getByText("Old version")).toThrow();
    expect(wrapper.getByText("New version")).toBeTruthy();
  });

  it("should display only old diff", () => {
    createWrapper({
      oldDiff: { snippetTitle: "old-field" },
    });

    expect(wrapper.getByText("Old version")).toBeTruthy();
    expect(() => wrapper.getByText("New version")).toThrow();
  });
});
