import type { RenderResult } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { mockTriggerLog } from "~/testing/data/mock_function_triggers";
import TriggerLogDetails from "./TriggerLogDetails";

describe("~/apps/[id]/environments/[env-id]/function-triggers/_components/TriggerLogDetails.tsx", () => {
  let wrapper: RenderResult;

  it("renders documentation as markdown below the response body", () => {
    wrapper = render(
      <TriggerLogDetails
        log={mockTriggerLog()}
        documentation={"Runs the **rollup**.\n\nPing #data if it fails."}
      />
    );

    expect(wrapper.getByText("Documentation")).toBeTruthy();
    expect(wrapper.getByTestId("markdown").innerHTML).toContain(
      "<strong>rollup</strong>"
    );

    // The section is appended after the response body, not before it.
    const html = wrapper.container.innerHTML;
    expect(html.indexOf("Response body")).toBeLessThan(
      html.indexOf("Documentation")
    );
  });

  it("omits the section when the trigger has no documentation", () => {
    wrapper = render(<TriggerLogDetails log={mockTriggerLog()} />);

    expect(wrapper.queryByText("Documentation")).toBe(null);
    expect(wrapper.queryByTestId("markdown")).toBe(null);
  });
});
