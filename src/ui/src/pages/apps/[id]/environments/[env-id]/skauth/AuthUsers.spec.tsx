import type { RenderResult } from "@testing-library/react";
import { describe, expect, beforeEach, afterEach, it, vi } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import nock from "nock";
import { EnvironmentContext } from "~/pages/apps/[id]/environments/Environment.context";
import mockApp from "~/testing/data/mock_app";
import mockEnv from "~/testing/data/mock_environment";
import AuthUsers from "./AuthUsers";

const apiDomain = process.env.API_DOMAIN || "";

interface MockUser {
  id: string;
  firstName: string;
  lastName: string;
  email: string;
  avatar: string;
  createdAt: number;
  lastLoginAt: number;
}

const defaultUser: MockUser = {
  id: "uuid-1",
  firstName: "Jane",
  lastName: "Doe",
  email: "jane@doe.org",
  avatar: "",
  createdAt: 1700000000,
  lastLoginAt: 0,
};

const mockFetchUsers = (envId: string, results: MockUser[] = [defaultUser]) =>
  nock(apiDomain)
    .get(`/skauth/users?envId=${envId}&from=0`)
    .reply(200, { results, hasNextPage: false });

describe("~/pages/apps/[id]/environments/[env-id]/skauth/AuthUsers.tsx", () => {
  let wrapper: RenderResult;
  let currentEnv: Environment;

  const createWrapper = async () => {
    const currentApp = mockApp();
    currentEnv = mockEnv({ app: currentApp });

    const scope = mockFetchUsers(currentEnv.id!);

    wrapper = render(
      <EnvironmentContext.Provider value={{ environment: currentEnv }}>
        <AuthUsers />
      </EnvironmentContext.Provider>,
    );

    await waitFor(() => {
      expect(scope.isDone()).toBe(true);
    });
  };

  beforeEach(async () => {
    await createWrapper();
  });

  afterEach(() => {
    nock.cleanAll();
    vi.clearAllMocks();
  });

  it("renders the registered users", async () => {
    await waitFor(() => {
      expect(wrapper.getByText("jane@doe.org")).toBeTruthy();
      expect(wrapper.getByText("Jane Doe")).toBeTruthy();
    });
  });

  it("updates a user through the edit modal", async () => {
    fireEvent.click(wrapper.getByLabelText("User actions"));

    await waitFor(() => {
      expect(wrapper.getByText("Edit")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByText("Edit"));

    await waitFor(() => {
      expect(wrapper.getByText("Edit user")).toBeTruthy();
    });

    const updateScope = nock(apiDomain)
      .put(`/skauth/users/uuid-1`, {
        envId: currentEnv.id,
        email: "new@doe.org",
        firstName: "Jane",
        lastName: "Doe",
      })
      .reply(200, { ...defaultUser, email: "new@doe.org" });

    // Refetch after a successful update.
    const refetchScope = mockFetchUsers(currentEnv.id!, [
      { ...defaultUser, email: "new@doe.org" },
    ]);

    const emailInput = wrapper.getByDisplayValue("jane@doe.org");
    fireEvent.change(emailInput, { target: { value: "new@doe.org" } });

    fireEvent.click(wrapper.getByText("Save"));

    await waitFor(() => {
      expect(updateScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
    });
  });

  it("deletes a user after confirmation", async () => {
    fireEvent.click(wrapper.getByLabelText("User actions"));

    await waitFor(() => {
      expect(wrapper.getByText("Delete")).toBeTruthy();
    });

    fireEvent.click(wrapper.getByText("Delete"));

    await waitFor(() => {
      expect(
        wrapper.getByText("Are you sure you want to continue?"),
      ).toBeTruthy();
    });

    const deleteScope = nock(apiDomain)
      .delete(`/skauth/users/uuid-1?envId=${currentEnv.id}`)
      .reply(200, { ok: true });

    // Refetch after a successful delete.
    const refetchScope = mockFetchUsers(currentEnv.id!, []);

    fireEvent.click(wrapper.getByText("Yes, continue"));

    await waitFor(() => {
      expect(deleteScope.isDone()).toBe(true);
      expect(refetchScope.isDone()).toBe(true);
    });
  });
});
