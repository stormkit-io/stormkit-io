import type { RenderResult } from "@testing-library/react";
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { render, waitFor, fireEvent } from "@testing-library/react";
import nock from "nock";
import PendingUsers from "./PendingUsers";

interface User {
  id: string;
  email: string;
  displayName: string;
  memberSince: number;
}

describe("~/pages/admin/AuthConfig/PendingUsers.tsx", () => {
  let wrapper: RenderResult;

  const mockUsers: User[] = [
    {
      id: "user-1",
      email: "user1@example.com",
      displayName: "User One",
      memberSince: Date.now() - 86400000, // 1 day ago
    },
    {
      id: "user-2",
      email: "user2@example.com",
      displayName: "User Two",
      memberSince: Date.now() - 172800000, // 2 days ago
    },
    {
      id: "user-3",
      email: "user3@example.com",
      displayName: "User Three",
      memberSince: Date.now() - 259200000, // 3 days ago
    },
  ];

  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  const fetchPendingUsersScope = (users: User[] = mockUsers) => {
    return nock(process.env.API_DOMAIN || "")
      .get("/admin/users/pending")
      .reply(200, { users });
  };

  const createWrapper = () => {
    wrapper = render(<PendingUsers />);
  };

  describe("fetching pending users", () => {
    it("renders the component with loading state", async () => {
      const scope = fetchPendingUsersScope();
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("Pending Users")).toBeTruthy();
        expect(
          wrapper.getByText("List of users awaiting approval")
        ).toBeTruthy();
      });
    });

    it("displays error when API fails", async () => {
      nock(process.env.API_DOMAIN || "")
        .get("/admin/users/pending")
        .reply(500);

      createWrapper();

      await waitFor(() => {
        expect(
          wrapper.getByText(/Something went wrong while fetching pending users/)
        ).toBeTruthy();
      });
    });

    it("displays info message when no pending users exist", async () => {
      const scope = fetchPendingUsersScope([]);
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(
          wrapper.getByText("There are no pending users at the moment.")
        ).toBeTruthy();
      });
    });

    it("renders 3 pending users in table", async () => {
      const scope = fetchPendingUsersScope();
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
        expect(wrapper.getByText("user1@example.com")).toBeTruthy();
        expect(wrapper.getByText("user2@example.com")).toBeTruthy();
        expect(wrapper.getByText("user3@example.com")).toBeTruthy();
        expect(wrapper.getByText("User One")).toBeTruthy();
        expect(wrapper.getByText("User Two")).toBeTruthy();
        expect(wrapper.getByText("User Three")).toBeTruthy();
      });
    });
  });

  describe("selecting pending users", () => {
    it("allows selecting individual users", async () => {
      const scope = fetchPendingUsersScope();
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      // Get all checkboxes (including select all)
      const checkboxes = wrapper.getAllByRole("checkbox");

      // Initially buttons should be disabled
      const approveButton = wrapper.getByText("Approve selected");
      const rejectButton = wrapper.getByText("Reject selected");

      await waitFor(() => {
        expect(approveButton.getAttribute("disabled")).toBe("");
        expect(rejectButton.getAttribute("disabled")).toBe("");
      });

      // Click the first user checkbox (index 1, as 0 is select all)
      fireEvent.click(checkboxes[1]);

      await waitFor(() => {
        expect(approveButton.getAttribute("disabled")).toBe(null);
        expect(rejectButton.getAttribute("disabled")).toBe(null);
      });
    });

    it("deselecting all users disables buttons", async () => {
      const scope = fetchPendingUsersScope();
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");

      // Select a user
      fireEvent.click(checkboxes[1]);

      await waitFor(() => {
        expect(
          wrapper.getByText("Approve selected").getAttribute("disabled")
        ).toBe(null);
      });

      // Deselect the user
      fireEvent.click(checkboxes[1]);

      await waitFor(() => {
        expect(
          wrapper.getByText("Approve selected").getAttribute("disabled")
        ).toBe("");
        expect(
          wrapper.getByText("Reject selected").getAttribute("disabled")
        ).toBe("");
      });
    });
  });

  describe("selecting and deselecting all users", () => {
    it("select all checkbox selects all users", async () => {
      const scope = fetchPendingUsersScope();
      createWrapper();

      await waitFor(() => {
        expect(scope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");
      const selectAllCheckbox = checkboxes[0];

      // Click select all
      fireEvent.click(selectAllCheckbox);

      await waitFor(() => {
        // All checkboxes should be checked
        checkboxes.forEach(checkbox => {
          expect((checkbox as HTMLInputElement).checked).toBe(true);
        });

        // Buttons should be enabled
        expect(
          wrapper.getByText("Approve selected").getAttribute("disabled")
        ).toBe(null);
        expect(
          wrapper.getByText("Reject selected").getAttribute("disabled")
        ).toBe(null);
      });
    });
  });

  describe("rejecting selected users", () => {
    it("successfully rejects 2 selected users", async () => {
      const fetchScope = fetchPendingUsersScope();
      const manageScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/manage", {
          userIds: ["user-1", "user-2"],
          action: "reject",
        })
        .reply(200, { ok: true });

      const refreshScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/users/pending")
        .reply(200, {
          users: [mockUsers[2]], // Only user-3 remains
        });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");

      // Select first two users
      fireEvent.click(checkboxes[1]);
      fireEvent.click(checkboxes[2]);

      await waitFor(() => {
        expect(
          wrapper.getByText("Reject selected").getAttribute("disabled")
        ).toBe(null);
      });

      // Click reject button
      fireEvent.click(wrapper.getByText("Reject selected"));

      // Confirm modal should appear
      await waitFor(() => {
        expect(wrapper.getByText("Reject Users")).toBeTruthy();
        expect(
          wrapper.getByText(/You are about to reject the selected users/)
        ).toBeTruthy();
      });

      // Click confirm button
      const confirmButton = wrapper.getByRole("button", { name: /continue/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(manageScope.isDone()).toBe(true);
        expect(refreshScope.isDone()).toBe(true);

        // Only user-3 should remain
        expect(wrapper.getByText("user3@example.com")).toBeTruthy();
        expect(() => wrapper.getByText("user1@example.com")).toThrow();
        expect(() => wrapper.getByText("user2@example.com")).toThrow();
      });
    });

    it("displays error when rejection fails", async () => {
      const fetchScope = fetchPendingUsersScope();
      const manageScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/manage")
        .reply(500);

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");

      // Select a user
      fireEvent.click(checkboxes[1]);

      await waitFor(() => {
        expect(
          wrapper.getByText("Reject selected").getAttribute("disabled")
        ).toBe(null);
      });

      // Click reject button
      fireEvent.click(wrapper.getByText("Reject selected"));

      // Confirm modal should appear
      await waitFor(() => {
        expect(wrapper.getByText("Reject Users")).toBeTruthy();
      });

      // Click confirm button
      const confirmButton = wrapper.getByRole("button", { name: /continue/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(manageScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            /Something went wrong while managing the selected users/
          )
        ).toBeTruthy();
      });
    });
  });

  describe("approving selected users", () => {
    it("successfully approves 2 selected users", async () => {
      const fetchScope = fetchPendingUsersScope();
      const manageScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/manage", {
          userIds: ["user-1", "user-2"],
          action: "approve",
        })
        .reply(200, { ok: true });

      const refreshScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/users/pending")
        .reply(200, {
          users: [mockUsers[2]], // Only user-3 remains
        });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");

      // Select first two users
      fireEvent.click(checkboxes[1]);
      fireEvent.click(checkboxes[2]);

      await waitFor(() => {
        expect(
          wrapper.getByText("Approve selected").getAttribute("disabled")
        ).toBe(null);
      });

      // Click approve button
      fireEvent.click(wrapper.getByText("Approve selected"));

      // Confirm modal should appear
      await waitFor(() => {
        expect(wrapper.getByText("Approve Users")).toBeTruthy();
        expect(
          wrapper.getByText(/You are about to approve the selected users/)
        ).toBeTruthy();
      });

      // Click confirm button
      const confirmButton = wrapper.getByRole("button", { name: /continue/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(manageScope.isDone()).toBe(true);
        expect(refreshScope.isDone()).toBe(true);

        // Only user-3 should remain
        expect(wrapper.getByText("user3@example.com")).toBeTruthy();
        expect(() => wrapper.getByText("user1@example.com")).toThrow();
        expect(() => wrapper.getByText("user2@example.com")).toThrow();
      });
    });

    it("displays error when approval fails", async () => {
      const fetchScope = fetchPendingUsersScope();
      const manageScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/manage")
        .reply(500);

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");

      // Select a user
      fireEvent.click(checkboxes[1]);

      await waitFor(() => {
        expect(
          wrapper.getByText("Approve selected").getAttribute("disabled")
        ).toBe(null);
      });

      // Click approve button
      fireEvent.click(wrapper.getByText("Approve selected"));

      // Confirm modal should appear
      await waitFor(() => {
        expect(wrapper.getByText("Approve Users")).toBeTruthy();
      });

      // Click confirm button
      const confirmButton = wrapper.getByRole("button", { name: /continue/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(manageScope.isDone()).toBe(true);
        expect(
          wrapper.getByText(
            /Something went wrong while managing the selected users/
          )
        ).toBeTruthy();
      });
    });

    it("clears selection after successful approval", async () => {
      const fetchScope = fetchPendingUsersScope();
      const manageScope = nock(process.env.API_DOMAIN || "")
        .post("/admin/users/manage", {
          userIds: ["user-1"],
          action: "approve",
        })
        .reply(200, { ok: true });

      const refreshScope = nock(process.env.API_DOMAIN || "")
        .get("/admin/users/pending")
        .reply(200, {
          users: [mockUsers[1], mockUsers[2]], // user-2 and user-3 remain
        });

      createWrapper();

      await waitFor(() => {
        expect(fetchScope.isDone()).toBe(true);
      });

      const checkboxes = wrapper.getAllByRole("checkbox");

      // Select first user
      fireEvent.click(checkboxes[1]);

      // Click approve button
      fireEvent.click(wrapper.getByText("Approve selected"));

      // Confirm modal should appear and confirm
      await waitFor(() => {
        expect(wrapper.getByText("Approve Users")).toBeTruthy();
      });

      const confirmButton = wrapper.getByRole("button", { name: /continue/i });
      fireEvent.click(confirmButton);

      await waitFor(() => {
        expect(manageScope.isDone()).toBe(true);
        expect(refreshScope.isDone()).toBe(true);

        // Buttons should be disabled again (selection cleared)
        expect(
          wrapper.getByText("Approve selected").getAttribute("disabled")
        ).toBe("");
        expect(
          wrapper.getByText("Reject selected").getAttribute("disabled")
        ).toBe("");
      });
    });
  });
});
