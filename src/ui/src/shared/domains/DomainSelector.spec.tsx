import type { RenderResult } from "@testing-library/react";
import type { Scope } from "nock";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";
import {
  waitFor,
  fireEvent,
  render,
  getAllByText,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { mockFetchDomains } from "~/testing/nocks/nock_domains";
import DomainSelector from "./DomainSelector";

interface Props {
  multiple?: boolean;
  withDevDomains?: boolean;
}

describe("~/shared/domains/DomainSelector.tsx", () => {
  let wrapper: RenderResult;
  let onDomainSelect: Mock;
  let fetchDomainsScope: Scope;

  const appId = "2541";
  const envId = "58181";

  const findDropdown = () => wrapper.getByRole("combobox");
  const findOption = (text: string) => getAllByText(document.body, text).at(0);

  const domains = [
    { domainName: "www.e.org", verified: true, id: "2" },
    { domainName: "e.org", verified: true, id: "2" },
  ];

  const createWrapper = async ({ withDevDomains, multiple }: Props) => {
    fetchDomainsScope = mockFetchDomains({
      appId,
      envId,
      verified: true,
      response: {
        domains,
      },
    });

    onDomainSelect = vi.fn();

    wrapper = render(
      <DomainSelector
        label="Domains"
        appId={appId}
        envId={envId}
        multiple={multiple}
        withDevDomains={withDevDomains}
        onDomainSelect={onDomainSelect}
      />
    );

    await waitFor(() => {
      expect(fetchDomainsScope.isDone()).toBe(true);
    });
  };

  const openDropdown = async () => {
    let selector;

    await waitFor(() => {
      selector = findDropdown();
    });

    if (!selector) {
      throw new Error("Cannot find dropdown");
    }

    expect(selector).toBeTruthy();
    fireEvent.mouseDown(selector);
    return selector;
  };

  describe("multi select", () => {
    beforeEach(async () => {
      await createWrapper({ withDevDomains: true, multiple: true });
      await openDropdown();
    });

    it("should display fetched domains", async () => {
      expect(findOption("All domains")).toBeTruthy();
      expect(findOption("All development endpoints (*.dev)")).toBeTruthy();

      fireEvent.click(findOption("www.e.org")!);
      fireEvent.click(findOption("e.org")!);

      // Closes the dropdown
      await userEvent.keyboard("{Escape}");

      expect(onDomainSelect).toHaveBeenCalledWith(["www.e.org", "e.org"]);
    });
  });

  describe("single select", () => {
    beforeEach(async () => {
      await createWrapper({ withDevDomains: false, multiple: false });
      await openDropdown();
    });

    it("should display fetched domains", async () => {
      expect(() => findOption("All development endpoints (*.dev)")).toThrow();

      fireEvent.click(findOption("www.e.org")!);

      // When single select, we return the domain object instead.
      expect(onDomainSelect).toHaveBeenCalledWith([domains[0]]);
    });
  });

  describe("searching", () => {
    beforeEach(async () => {
      await createWrapper({ withDevDomains: true, multiple: true });
      await openDropdown();
    });

    it("should search a domain", async () => {
      const searchInput = wrapper
        .getByTestId("multiselect-search")
        .querySelector("input");

      expect(searchInput).toBeTruthy();

      fetchDomainsScope = mockFetchDomains({
        appId,
        envId,
        verified: true,
        domainName: "my-domain",
        response: {
          domains: [],
        },
      });

      await userEvent.type(searchInput!, "my-domain");

      await waitFor(() => {
        expect(fetchDomainsScope.isDone()).toBe(true);
        expect(wrapper.getByText("No domain found")).toBeTruthy();
      });
    });
  });
});
