import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import CodeBlock from "./CodeBlock";

describe("~/components/CodeBlock/CodeBlock.tsx", () => {
  it("highlights json content", () => {
    const wrapper = render(<CodeBlock>{'{"a": 1}'}</CodeBlock>);

    expect(wrapper.getByTestId("code-block").innerHTML).toContain(
      "tok-propertyName"
    );
  });

  it("highlights html content", () => {
    const wrapper = render(
      <CodeBlock>{"<!DOCTYPE html>\n<html lang=\"en\"></html>"}</CodeBlock>
    );

    const block = wrapper.getByTestId("code-block");

    expect(block.innerHTML).toContain("tok-");
    // The markup is shown as text, not parsed into the page.
    expect(block.querySelector("html")).toBe(null);
    expect(block.textContent).toContain("<!DOCTYPE html>");
  });

  it("renders plain text as is", () => {
    const wrapper = render(<CodeBlock>No payload</CodeBlock>);

    const block = wrapper.getByTestId("code-block");

    expect(block.textContent).toBe("No payload");
    expect(block.innerHTML).not.toContain("tok-");
  });

  it("renders oversized content as plain text", () => {
    const big = `{"a": "${"x".repeat(100_000)}"}`;
    const wrapper = render(<CodeBlock>{big}</CodeBlock>);

    const block = wrapper.getByTestId("code-block");

    expect(block.textContent).toBe(big);
    expect(block.innerHTML).not.toContain("tok-");
  });

  it("honours an explicit language over detection", () => {
    const wrapper = render(<CodeBlock language="json">{"[1]"}</CodeBlock>);

    expect(wrapper.getByTestId("code-block").innerHTML).toContain("tok-number");
  });
});
