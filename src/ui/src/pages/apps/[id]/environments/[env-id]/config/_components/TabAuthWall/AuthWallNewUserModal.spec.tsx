import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach, Mock, vi } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import mockApp from "~/testing/data/mock_app";
import mockEnvironment from "~/testing/data/mock_environment";
import { mockCreateNewLogin } from "~/testing/nocks/nock_auth_wall";
import AuthWallNewUserModal from "./AuthWallNewUserModal";
import userEvent from "@testing-library/user-event";

describe("~/pages/apps/[id]/environments/[env-id]/config/_components/TabAuthWall/AuthWallNewUserModal.tsx", () => {
  let wrapper: RenderResult;
  let app: App;
  let env: Environment;
  let onClose: Mock;
  let onSuccess: Mock;

  beforeEach(() => {
    app = mockApp();
    env = mockEnvironment({ app });
    createWrapper();
  });

  const createWrapper = () => {
    onClose = vi.fn();
    onSuccess = vi.fn();
    wrapper = render(
      <AuthWallNewUserModal
        appId={app.id}
        envId={env.id!}
        onClose={onClose}
        onSuccess={onSuccess}
      />
    );
  };

  it("should submit the form", async () => {
    const email = wrapper.getByLabelText("Email address");
    const password = wrapper.getByLabelText("Password");

    expect(email).toBeTruthy();
    expect(password).toBeTruthy();

    const scope = mockCreateNewLogin({
      appId: app.id,
      envId: env.id!,
      email: "email@example.org",
      password: "12345678",
    });

    await userEvent.type(email, "email@example.org");
    await userEvent.type(password, "12345678");
    await fireEvent.click(wrapper.getByText("Create"));

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });
});
