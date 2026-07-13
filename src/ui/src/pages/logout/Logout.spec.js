import { waitFor } from "@testing-library/react";
import { withMockContext } from "~/testing/helpers";

const fileName = "pages/logout";

describe(fileName, () => {
  const path = `~/${fileName}`;

  test.skip("should trigger the logout function on mount", async () => {
    const spy = jest.fn();

    withMockContext({
      path,
      props: {
        logout: spy,
      },
    });

    await waitFor(() => {
      expect(spy).toHaveBeenCalledWith();
    });
  });
});
