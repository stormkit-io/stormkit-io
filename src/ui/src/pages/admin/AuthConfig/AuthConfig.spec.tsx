import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import nock from "nock";
import AuthConfig from "./AuthConfig";

describe("~/pages/admin/AuthConfig/AuthConfig.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  const fetchAuthConfigScope = (config = {}) => {
    return nock(process.env.API_DOMAIN || "")
      .get("/admin/users/sign-up-mode")
      .reply(200, {
        signUpMode: "on",
        whitelist: [],
        ...config,
      });
  };

  const createWrapper = async () => {
    wrapper = render(<AuthConfig />);
  };

  const findSignUpModeSelect = () => {
    return wrapper.getByLabelText("Sign up mode");
  };

  const findWhitelistInput = () => {
    return wrapper.getByLabelText("Whitelist") as HTMLInputElement;
  };

  describe("initial render", () => {
    it("renders the component with loading state", async () => {
      const scope = fetchAuthConfigScope();
      createWrapper();

      expect(wrapper.getByText("User management")).toBeTruthy();
      expect(
        wrapper.getByText("Configure how your users can access Stormkit")
      ).toBeTruthy();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });
    });

    it("displays error when API fails", async () => {
      nock(process.env.API_DOMAIN || "")
        .get("/admin/users/sign-up-mode")
        .reply(500);

      createWrapper();

      await waitFor(() => {
        expect(
          wrapper.getByText(
            /Something went wrong while fetching user management configuration/
          )
        ).toBeTruthy();
      });
    });

    it("loads config with signUpMode 'waitlist' and whitelist", async () => {
      const scope = fetchAuthConfigScope({
        signUpMode: "waitlist",
        whitelist: ["example.org", "test.com"],
      });

      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(findSignUpModeSelect().textContent).toBe("Approval mode");
        expect(findWhitelistInput()?.value).toBe("example.org, test.com");
      });
    });

    it("hides whitelist field when signUpMode is not 'waitlist'", async () => {
      const scope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByLabelText("Whitelist")).toBeTruthy();
      });

      const select = findSignUpModeSelect();

      fireEvent.mouseDown(select);

      const onOption = await waitFor(() =>
        wrapper.getByText("On (all users are allowed)")
      );

      fireEvent.click(onOption);

      await waitFor(() => {
        expect(select.textContent).toBe("On (all users are allowed)");
        expect(() => findWhitelistInput()).toThrow();
      });
    });
  });

  describe("form submission", () => {
    it("successfully submits form with signUpMode 'waitlist' and whitelist", async () => {
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode", {
          signUpMode: "waitlist",
          whitelist: ["example.org", "test.com"],
        })
        .reply(200, { ok: true });

      createWrapper();

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
      const fetchScope = fetchAuthConfigScope({ signUpMode: "on" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode")
        .reply(500);

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
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
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode", {
          signUpMode: "waitlist",
          whitelist: ["example.org", "test.com", "another.com"],
        })
        .reply(200, { ok: true });

      createWrapper();

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
    it("allows all positive whitelist entries", async () => {
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode", {
          signUpMode: "waitlist",
          whitelist: ["example.org", "test.com"],
        })
        .reply(200, { ok: true });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

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
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode", {
          signUpMode: "waitlist",
          whitelist: ["!example.org", "!test.com"],
        })
        .reply(200, { ok: true });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

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
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

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
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

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
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode")
        .reply(200, { ok: true });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

        fireEvent.change(findWhitelistInput(), {
          target: { value: "example.org, !test.com" },
        });
      });

      fireEvent.click(wrapper.getByText("Save"));

      await waitFor(() => {
        expect(findWhitelistInput().getAttribute("aria-invalid")).toBe("true");
      });

      // Should not make the API call
      expect(updateScope.isDone()).toBe(false);
    });

    it("clears validation error on re-submission with valid data", async () => {
      const fetchScope = fetchAuthConfigScope({ signUpMode: "waitlist" });
      const updateScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/sign-up-mode", {
          signUpMode: "waitlist",
          whitelist: ["example.org", "test.com"],
        })
        .reply(200, { ok: true });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);

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
});
