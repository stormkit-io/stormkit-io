import React from "react";
import { render, fireEvent, waitFor, getByText } from "@testing-library/react";
import * as data from "~/testing/data";
import EnvironmentsSelector from "./index";

describe("EnvironmentsSelector component", () => {
  let wrapper;
  let environments;
  let onSelect;
  const placeholder = "Select environment";

  const findSelector = () => wrapper.getByLabelText(placeholder);
  const findOption = text => getByText(document.body, text);

  beforeEach(() => {
    onSelect = jest.fn();
    environments = data.mockEnvironmentsResponse().envs;

    wrapper = render(
      <form>
        <EnvironmentsSelector
          environments={environments}
          placeholder={placeholder}
          onSelect={onSelect}
        />
      </form>
    );
  });

  test.skip("Should select the environments and trigger an onSelect call", async () => {
    await waitFor(() => {
      const selector = findSelector();
      expect(selector).toBeTruthy();
      fireEvent.mouseDown(selector);
    });

    await waitFor(() => {
      const option = findOption("(app.stormkit.io)");
      fireEvent.click(option);
    });

    await waitFor(() => {
      expect(onSelect).toHaveBeenCalledWith(environments[0]);
    });
  });
});
