import type { RenderResult } from "@testing-library/react";
import type { InstanceMailer } from "./actions";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import nock from "nock";
import { AuthContext } from "~/pages/auth/Auth.context";
import mockUser from "~/testing/data/mock_user";
import Mailer from "./Mailer";

const endpoint = process.env.API_DOMAIN || "";

const storedMailer: InstanceMailer = {
  host: "smtp.instance.test",
  port: "587",
  username: "instance-user",
  password: "****-****-****-****",
  from: "Stormkit <noreply@instance.test>",
};

describe("~/pages/admin/Mailer/Mailer.tsx", () => {
  let wrapper: RenderResult;
  let user: User;

  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  const createWrapper = async (mailer: InstanceMailer | null) => {
    const scope = nock(endpoint).get("/admin/mailer").reply(200, { mailer });

    user = mockUser();

    wrapper = render(
      <AuthContext.Provider value={{ user }}>
        <Mailer />
      </AuthContext.Provider>,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(wrapper.getByLabelText("SMTP Host")).toBeTruthy();
    });
  };

  it("should save a new mailer", async () => {
    await createWrapper(null);

    expect(() => wrapper.getByText("Send test email")).toThrow();
    expect(() => wrapper.getByText("Remove")).toThrow();

    await userEvent.type(
      wrapper.getByLabelText("SMTP Host"),
      "smtp.instance.test",
    );
    await userEvent.type(wrapper.getByLabelText("Username"), "instance-user");
    await userEvent.type(wrapper.getByLabelText("Password"), "instance-secret");
    await userEvent.type(
      wrapper.getByLabelText("From"),
      "noreply@instance.test",
    );

    const saveScope = nock(endpoint)
      .post("/admin/mailer", {
        host: "smtp.instance.test",
        port: "",
        username: "instance-user",
        password: "instance-secret",
        from: "noreply@instance.test",
      })
      .reply(200, { mailer: storedMailer });

    const refetchScope = nock(endpoint)
      .get("/admin/mailer")
      .reply(200, { mailer: storedMailer });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(saveScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
      expect(wrapper.getByText("The default mailer is saved.")).toBeTruthy();
    });
  });

  it("should show the validation errors", async () => {
    await createWrapper(null);

    nock(endpoint)
      .post("/admin/mailer")
      .reply(400, { errors: { host: "SMTP Host is a required field." } });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(wrapper.getByText("SMTP Host is a required field.")).toBeTruthy();
    });
  });

  it("should send a test email to the signed-in admin", async () => {
    await createWrapper(storedMailer);

    expect(wrapper.getByText(/A password is stored/)).toBeTruthy();

    const scope = nock(endpoint)
      .post("/admin/mailer/test", { to: user.email })
      .reply(200, { ok: true });

    fireEvent.click(wrapper.getByText("Send test email"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(
        wrapper.getByText(`A test email is sent to ${user.email}.`),
      ).toBeTruthy();
    });
  });

  it("should show why a test email failed", async () => {
    await createWrapper(storedMailer);

    nock(endpoint).post("/admin/mailer/test").reply(502, {
      error: "The SMTP server refused the email: 535 Authentication failed",
    });

    fireEvent.click(wrapper.getByText("Send test email"));

    await waitFor(() => {
      expect(wrapper.getByText(/535 Authentication failed/)).toBeTruthy();
    });
  });

  it("should remove the mailer", async () => {
    await createWrapper(storedMailer);

    fireEvent.click(wrapper.getByText("Remove"));

    const deleteScope = nock(endpoint)
      .delete("/admin/mailer")
      .reply(200, { ok: true });

    const refetchScope = nock(endpoint)
      .get("/admin/mailer")
      .reply(200, { mailer: null });

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(deleteScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
      expect(wrapper.getByText("The default mailer is removed.")).toBeTruthy();
    });
  });
});
