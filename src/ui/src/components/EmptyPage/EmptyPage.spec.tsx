import type { SxProps } from "@mui/material";
import { describe, expect, it } from "vitest";
import type { RenderResult } from "@testing-library/react";
import { render } from "@testing-library/react";
import EmptyPage from "./EmptyPage";

interface Props {
  children: React.ReactNode;
  sx: SxProps;
}

describe("~/components/EmptyPage.tsx", () => {
  let wrapper: RenderResult;

  const createWrapper = ({ children, sx }: Props) => {
    wrapper = render(<EmptyPage children={children} sx={sx} />);
  };

  it("should render the component as expected", () => {
    createWrapper({
      children: <span>Hello world</span>,
      sx: { color: "black" },
    });

    expect(wrapper).toMatchSnapshot();
  });
});
