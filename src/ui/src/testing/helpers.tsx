import { createRoutesStub } from "react-router";
import { render } from "@testing-library/react";

interface RenderWithRouterProps {
  el?: any;
  initialEntries?: string[];
  path?: string;
}

export const renderWithRouter = ({
  el: Element,
  initialEntries = ["/"],
  path = "*",
}: RenderWithRouterProps = {}) => {
  const Stub = createRoutesStub([
    {
      path,
      Component: Element,
    },
  ]);

  return render(<Stub initialEntries={initialEntries} initialIndex={0} />);
};

export const waitForPromises = () => new Promise(setImmediate);
