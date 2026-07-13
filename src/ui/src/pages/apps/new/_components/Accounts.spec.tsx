import type { RenderResult } from "@testing-library/react";
import type { Account } from "../types.d";
import { render, fireEvent, waitFor } from "@testing-library/react";
import Accounts from "./Accounts";

interface Props {
  loading?: boolean;
  accounts?: Account[];
  selected?: string;
  onAccountChange?: (val: string) => void;
}

describe("~/pages/apps/new/_components/Accounts.tsx", () => {
  let wrapper: RenderResult;

  const accounts: Account[] = [
    {
      id: "johndoe",
      avatar: "https://example.com/avatar/johndoe",
      login: "jdoe",
    },
    {
      id: "janedoe",
      avatar: "https://example.com/avatar/janedoe",
      login: "jane",
    },
  ];

  const createWrapper = ({
    loading = false,
    accounts = [],
    selected = "",
    onAccountChange = () => {},
  }: Props) => {
    wrapper = render(
      <Accounts
        loading={loading}
        accounts={accounts}
        onAccountChange={onAccountChange}
        selected={selected}
      />
    );
  };

  const findSelector = () => wrapper.getByRole("combobox");

  describe("loading", () => {
    test("does not display a spinner when loading is false", () => {
      createWrapper({ accounts, loading: false });
      expect(wrapper.container.innerHTML).not.toContain("spinner-bounce");
    });
  });

  describe("accounts", () => {
    test("displays a list of accounts", async () => {
      createWrapper({ accounts });
      const input = findSelector();
      expect(input).toBeTruthy();
      fireEvent.mouseDown(input);

      await waitFor(() => {
        expect(wrapper.getByText("jdoe")).toBeTruthy();
        expect(wrapper.getByText("jane")).toBeTruthy();
      });
    });
  });
});
