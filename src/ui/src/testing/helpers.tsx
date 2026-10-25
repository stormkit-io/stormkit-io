import { createRoutesStub } from "react-router";
import { act, render } from "@testing-library/react";
import { vi } from "vitest";
import api from "~/utils/api/Api";

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

// trackFetches records the api requests made from now on. settle resolves once
// every recorded request has answered and React has applied the result, so a
// test can assert after a delayed response without sleeping.
export const trackFetches = () => {
  const requests: Promise<unknown>[] = [];
  const original = api.fetch.bind(api);

  const spy = vi.spyOn(api, "fetch").mockImplementation((...args) => {
    const request = original(...args);
    requests.push(request);
    return request;
  });

  return {
    settle: async () => {
      let count;

      // A settled request can start another one, e.g. a breakdown after its
      // properties. React only renders, and so starts it, once act ends.
      do {
        count = requests.length;

        await act(async () => {
          await Promise.allSettled(requests);
          await waitForPromises();
        });
      } while (count !== requests.length);
    },
    restore: () => spy.mockRestore(),
  };
};
