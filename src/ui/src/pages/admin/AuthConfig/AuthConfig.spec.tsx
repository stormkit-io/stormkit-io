import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import { RootContext } from "~/pages/Root.context";
import nock, { Scope } from "nock";
import AuthConfig from "./AuthConfig";

interface Props {
  seats?: number;
  remaining?: number;
  edition?: "community" | "enterprise";
  signUpMode?: "on" | "off" | "waitlist";
  whitelist?: string[];
  status?: number;
}

describe("~/pages/admin/AuthConfig/AuthConfig.tsx", () => {
  let wrapper: RenderResult;
  let fetchScope: Scope;

  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  const createWrapper = async ({
    seats = 1,
    remaining = 0,
    edition = "community",
    signUpMode = "on",
    whitelist = [],
    status = 200,
  }: Props = {}) => {
    fetchScope = nock(process.env.API_DOMAIN || "")
      .get("/admin/users/sign-up-mode")
      .reply(status || 200, {
        signUpMode,
        whitelist,
      });

    wrapper = render(
      <RootContext.Provider
        value={{
          mode: "dark",
          setMode: vi.fn(),
          setRefreshToken: vi.fn(),
          details: { license: { seats, remaining, edition } },
        }}
      >
        <AuthConfig />
      </RootContext.Provider>
    );
  };

  const findSignUpModeSelect = () => {
    return wrapper.getByLabelText("Sign up mode");
  };

  const findWhitelistInput = () => {
    return wrapper.getByLabelText("Whitelist") as HTMLInputElement;
  };

  const openSignUpModeSelect = async () => {
    const select = await waitFor(() => {
      expect(fetchScope.isDone()).toBe(true);
      expect(findSignUpModeSelect()).toBeTruthy();
      return findSignUpModeSelect();
    });

    fireEvent.mouseDown(select);
  };

  const updateSignUpModeScope = ({
    signUpMode,
    whitelist,
    status = 200,
  }: Pick<Props, "signUpMode" | "whitelist" | "status">) => {
    return nock(process.env.API_DOMAIN || "")
      .post("/admin/users/sign-up-mode", {
        signUpMode,
        whitelist,
      })
      .reply(status, { ok: true });
  };

  describe("community edition", () => {
    it("renders the component with loading state", async () => {
      createWrapper({ signUpMode: "on", status: 200 });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByText("User management")).toBeTruthy();
        expect(
          wrapper.getByText("Configure how your users can access Stormkit")
        ).toBeTruthy();
      });
    });

    it("displays error when API fails", async () => {
      createWrapper({ status: 500 });

      await waitFor(() => {
        expect(
          wrapper.getByText(
            /Something went wrong while fetching user management configuration/
          )
        ).toBeTruthy();
      });
    });

    it("does not render pending users section for community edition", async () => {
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(() => wrapper.getByText("Pending Users")).toThrow();
      });
    });

    it("disables approval mode option for community edition", async () => {
      createWrapper();

      await openSignUpModeSelect();

      await waitFor(() => {
        const approvalOption = wrapper.getByText(
          /Approval mode \(Enterprise only\)/
        );
        expect(approvalOption).toBeTruthy();
        expect(
          approvalOption.closest("li")?.getAttribute("aria-disabled")
        ).toBe("true");
      });
    });
  });

  describe("form submission", () => {
    it("successfully submits form with signUpMode 'waitlist' and whitelist", async () => {
      const updateScope = updateSignUpModeScope({
        signUpMode: "waitlist",
        whitelist: ["example.org", "test.com"],
      });

      createWrapper({ signUpMode: "waitlist", status: 200 });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

        fireEvent.change(findWhitelistInput(), {
          target: { value: "example.org, test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            /The user management configuration is updated successfully/
          )
        ).toBeTruthy();
      });
    });

    it("displays error message when submission fails", async () => {
      createWrapper({ signUpMode: "on", status: 200 });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      const updateScope = updateSignUpModeScope({
        status: 500,
        signUpMode: "on",
        whitelist: [],
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            /Something went wrong while updating user management config/
          )
        ).toBeTruthy();
      });
    });

    it("trims whitespace from whitelist entries", async () => {
      const updateScope = updateSignUpModeScope({
        signUpMode: "waitlist",
        whitelist: ["example.org", "test.com", "another.com"],
      });

      createWrapper({ signUpMode: "waitlist", status: 200 });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

        fireEvent.change(findWhitelistInput(), {
          target: { value: "  example.org  ,  test.com  , another.com  " },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
      });
    });
  });

  describe("whitelist validation", () => {
    beforeEach(() => {
      createWrapper({ signUpMode: "waitlist", status: 200 });
    });

    it("allows all positive whitelist entries", async () => {
      const updateScope = updateSignUpModeScope({
        signUpMode: "waitlist",
        whitelist: ["example.org", "test.com"],
      });

      await waitFor(() => {
        fireEvent.change(findWhitelistInput(), {
          target: { value: "example.org, test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("false");
      });
    });

    it("allows all negated whitelist entries", async () => {
      const updateScope = updateSignUpModeScope({
        signUpMode: "waitlist",
        whitelist: ["!example.org", "!test.com"],
      });

      await waitFor(() => {
        fireEvent.change(findWhitelistInput(), {
          target: { value: "!example.org, !test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("false");
      });
    });

    it("shows error when mixing negated and positive entries", async () => {
      await waitFor(() => {
        fireEvent.change(findWhitelistInput(), {
          target: { value: "!example.org, test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("true");
        expect(
          wrapper.getByText(/All domains must either be allowed or denied/)
        ).toBeTruthy();
      });
    });

    it("shows error when mixing positive and negated entries", async () => {
      await waitFor(() => {
        fireEvent.change(findWhitelistInput(), {
          target: { value: "example.org, !test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("true");
        expect(
          wrapper.getByText(/All domains must either be allowed or denied/)
        ).toBeTruthy();
      });
    });

    it("prevents form submission when whitelist has mixed negations", async () => {
      await waitFor(() => {
        fireEvent.change(findWhitelistInput(), {
          target: { value: "example.org, !test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("true");
      });
    });

    it("clears validation error on re-submission with valid data", async () => {
      const updateScope = updateSignUpModeScope({
        signUpMode: "waitlist",
        whitelist: ["example.org", "test.com"],
      });

      await waitFor(() => {
        // First submission with invalid data
        fireEvent.change(findWhitelistInput(), {
          target: { value: "example.org, !test.com" },
        });
      });

      const saveButton = wrapper.getByText("Save");

      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("true");
      });

      // Second submission with valid data
      fireEvent.change(findWhitelistInput(), {
        target: { value: "example.org, test.com" },
      });

      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("false");
      });
    });
  });

  describe("enterprise edition features", () => {
    it("loads config with signUpMode 'waitlist' and whitelist", async () => {
      createWrapper({
        edition: "enterprise",
        signUpMode: "waitlist",
        whitelist: ["example.org", "test.com"],
      });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(findSignUpModeSelect().textContent).toBe("Approval mode");
        expect(findWhitelistInput()?.value).toBe("example.org, test.com");
      });
    });

    it("renders pending users section for enterprise edition", async () => {
      createWrapper({ edition: "enterprise", signUpMode: "on" });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByText("Pending Users")).toBeTruthy();
      });
    });

    it("enables approval mode option for enterprise edition", async () => {
      createWrapper({ edition: "enterprise", signUpMode: "on" });

      await openSignUpModeSelect();

      const opt = await waitFor(() => {
        const opt = wrapper.getByText("Approval mode");
        expect(opt).toBeTruthy();
        expect(opt.closest("li")?.getAttribute("aria-disabled")).toBe(null);
        return opt;
      });

      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode", {
          signUpMode: "waitlist",
          whitelist: [],
        })
        .reply(200, { ok: true });

      fireEvent.click(opt);
      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(updateScope.isDone()).toBe(true);
      });
    });

    it("hides whitelist field when signUpMode is not 'waitlist'", async () => {
      createWrapper({ edition: "enterprise", signUpMode: "waitlist" });

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
        expect(wrapper.getByLabelText("Whitelist")).toBeTruthy();
      });

      fireEvent.mouseDown(findSignUpModeSelect());

      const onOption = await waitFor(() =>
        wrapper.getByText("On (all users are allowed)")
      );

      fireEvent.click(onOption);

      await waitFor(() => {
        expect(() => findWhitelistInput()).toThrow();
        expect(findSignUpModeSelect().textContent).toBe(
          "On (all users are allowed)"
        );
      });
    });
  });
});
