import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  render,
  waitFor,
  fireEvent,
  type RenderResult,
} from "@testing-library/react";
import nock from "nock";
import CloudApps from "./CloudApps";

describe("~/pages/admin/CloudApps.tsx", () => {
  let wrapper: RenderResult;

  beforeEach(() => {
    nock.cleanAll();
    wrapper = render(<CloudApps />);
  });

  afterEach(() => {
    nock.cleanAll();
  });

  const openMenuAndClick = (label: string) => {
    fireEvent.click(wrapper.getAllByLabelText("expand").at(0)!);
    fireEvent.click(wrapper.getByText(label));
  };

  it("should render the component with search input", () => {
    expect(wrapper.getByLabelText("Search")).toBeTruthy();
    expect(
      wrapper.getByPlaceholderText(
        "Search an app by it's display or domain name and press Enter"
      )
    ).toBeTruthy();
    expect(wrapper.getByText("Apps")).toBeTruthy();
    expect(
      wrapper.getByText(
        "Use the input below to search for apps and manage them."
      )
    ).toBeTruthy();
  });

  it("should show info message when no app is found", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=nonexistent")
      .reply(204);

    const searchInput = wrapper.getByLabelText("Search");

    fireEvent.change(searchInput, { target: { value: "nonexistent" } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    await waitFor(() => {
      expect(
        wrapper.getByText(
          "There is no application based on your search criteria."
        )
      ).toBeTruthy();
    });
  });

  it("should display loading state when searching", async () => {
    // Mock a delayed response
    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=example.com")
      .delay(100)
      .reply(200, {
        app: {
          id: "1",
          displayName: "Test App",
          createdAt: "2024-01-01T00:00:00Z",
        },
        user: {
          displayName: "Test User",
          email: "test@example.com",
        },
      });

    const searchInput = wrapper.getByLabelText("Search");

    fireEvent.change(searchInput, { target: { value: "example.com" } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    // Should show loading spinner
    expect(
      wrapper.container.querySelector(".MuiCircularProgress-root")
    ).toBeTruthy();
  });

  it("should successfully fetch and display app details", async () => {
    const mockApp = {
      id: "123",
      displayName: "My Test App",
      createdAt: "2024-01-15T10:30:00Z",
    };

    const mockUser = {
      displayName: "John Doe",
      email: "john@example.com",
    };

    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=myapp.com")
      .reply(200, {
        app: mockApp,
        user: mockUser,
      });

    const searchInput = wrapper.getByLabelText("Search");

    fireEvent.change(searchInput, { target: { value: "myapp.com" } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    await waitFor(() => {
      expect(wrapper.getByText("123")).toBeTruthy();
      expect(wrapper.getByText("My Test App")).toBeTruthy();
      expect(wrapper.getByText("John Doe")).toBeTruthy();
      expect(wrapper.getByText("john@example.com")).toBeTruthy();
      expect(wrapper.getByText("ID")).toBeTruthy();
      expect(wrapper.getByText("Display name")).toBeTruthy();
      expect(wrapper.getByText("Created at")).toBeTruthy();
      expect(wrapper.getByText("User display")).toBeTruthy();
      expect(wrapper.getByText("Email")).toBeTruthy();
    });
  });

  it("should show error when app fetch fails", async () => {
    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=error.com")
      .reply(500);

    const searchInput = wrapper.getByLabelText("Search");

    fireEvent.change(searchInput, { target: { value: "error.com" } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    await waitFor(() => {
      expect(
        wrapper.getByText("Something went wrong while fetching app")
      ).toBeTruthy();
    });
  });

  it("should only trigger search on Enter key", () => {
    const searchInput = wrapper.getByLabelText("Search");

    // Test other keys don't trigger search
    fireEvent.change(searchInput, { target: { value: "test" } });
    fireEvent.keyUp(searchInput, { key: "Tab" });

    // Should not show any loading or results
    expect(
      wrapper.container.querySelector(".MuiCircularProgress-root")
    ).toBeFalsy();

    fireEvent.keyUp(searchInput, { key: "Space" });
    expect(
      wrapper.container.querySelector(".MuiCircularProgress-root")
    ).toBeFalsy();
  });

  describe("App deletion", () => {
    beforeEach(async () => {
      const mockApp = {
        id: "456",
        displayName: "App to Delete",
        createdAt: "2024-01-01T00:00:00Z",
      };

      const mockUser = {
        displayName: "Jane Smith",
        email: "jane@example.com",
      };

      nock(process.env.API_DOMAIN || "")
        .get("/admin/cloud/app?url=delete-me.com")
        .reply(200, {
          app: mockApp,
          user: mockUser,
        });

      const searchInput = wrapper.getByLabelText("Search");
      fireEvent.change(searchInput, { target: { value: "delete-me.com" } });
      fireEvent.keyUp(searchInput, { key: "Enter" });

      await waitFor(() => {
        expect(wrapper.getByText("App to Delete")).toBeTruthy();
      });
    });

    it("should open confirm modal when delete button is clicked", async () => {
      openMenuAndClick("Delete");

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "This will delete the application and soft delete the user."
          )
        ).toBeTruthy();
      });
    });

    it("should successfully delete app", async () => {
      nock(process.env.API_DOMAIN || "")
        .delete("/admin/cloud/app?appId=456")
        .reply(200);

      openMenuAndClick("Delete");

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "This will delete the application and soft delete the user."
          )
        ).toBeTruthy();
      });

      fireEvent.click(
        wrapper.getByRole("button", {
          name: "Yes, continue",
        })
      );

      await waitFor(() => {
        expect(
          wrapper.getByText("The app has been deleted successfully.")
        ).toBeTruthy();
      });
    });

    it("should show error when deletion fails", async () => {
      nock(process.env.API_DOMAIN || "")
        .delete("/admin/cloud/app?appId=456")
        .reply(500);

      openMenuAndClick("Delete");

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "This will delete the application and soft delete the user."
          )
        ).toBeTruthy();
      });

      fireEvent.click(
        wrapper.getByRole("button", {
          name: "Yes, continue",
        })
      );

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "Something went wrong while deleting the app and user."
          )
        ).toBeTruthy();
      });
    });

    it("should close modal when cancel is clicked", async () => {
      openMenuAndClick("Delete");

      await waitFor(() => {
        expect(
          wrapper.getByText(
            "This will delete the application and soft delete the user."
          )
        ).toBeTruthy();
      });

      const cancelButton =
        wrapper.getByRole("button", { name: /cancel/i }) ||
        wrapper.getByRole("button", { name: /no/i });

      fireEvent.click(cancelButton);

      await waitFor(() => {
        expect(
          wrapper.queryByText(
            "This will delete the application and soft delete the user."
          )
        ).toBeFalsy();
      });
    });
  });

  it("should properly encode URLs in API requests", async () => {
    const encodedUrl = "https://example.com/path?param=value";

    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=" + encodeURIComponent(encodedUrl))
      .reply(200, {
        app: {
          id: "789",
          displayName: "URL Test App",
          createdAt: "2024-01-01T00:00:00Z",
        },
        user: {
          displayName: "URL Test User",
          email: "url@example.com",
        },
      });

    const searchInput = wrapper.getByLabelText("Search");

    fireEvent.change(searchInput, { target: { value: encodedUrl } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    await waitFor(() => {
      expect(wrapper.getByText("URL Test App")).toBeTruthy();
    });
  });

  it("should clear previous results when new search is performed", async () => {
    // First search
    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=first.com")
      .reply(200, {
        app: {
          id: "1",
          displayName: "First App",
          createdAt: "2024-01-01T00:00:00Z",
        },
        user: {
          displayName: "First User",
          email: "first@example.com",
        },
      });

    const searchInput = wrapper.getByLabelText("Search");

    fireEvent.change(searchInput, { target: { value: "first.com" } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    await waitFor(() => {
      expect(wrapper.getByText("First App")).toBeTruthy();
    });

    // Second search that fails
    nock(process.env.API_DOMAIN || "")
      .get("/admin/cloud/app?url=second.com")
      .reply(404);

    fireEvent.change(searchInput, { target: { value: "second.com" } });
    fireEvent.keyUp(searchInput, { key: "Enter" });

    await waitFor(() => {
      expect(wrapper.queryByText("First App")).toBeFalsy();
      expect(
        wrapper.getByText(
          "There is no application based on your search criteria."
        )
      ).toBeTruthy();
    });
  });

  describe("Visit user action", () => {
    beforeEach(async () => {
      const mockApp = {
        id: "999",
        displayName: "App to Visit",
        createdAt: "2024-01-01T00:00:00Z",
      };

      const mockUser = {
        id: "user-999",
        displayName: "Visit User",
        email: "visit@example.com",
      };

      nock(process.env.API_DOMAIN || "")
        .get("/admin/cloud/app?url=999")
        .reply(200, {
          app: mockApp,
          user: mockUser,
        });

      const searchInput = wrapper.getByLabelText("Search");

      fireEvent.change(searchInput, { target: { value: "999" } });
      fireEvent.keyUp(searchInput, { key: "Enter" });

      await waitFor(() => {
        expect(wrapper.getByText("App to Visit")).toBeTruthy();
      });
    });

    it("should call visit endpoint and reload page", async () => {
      nock(process.env.API_DOMAIN || "")
        .post("/admin/cloud/impersonate", { userId: "user-999" })
        .reply(200, { token: "impersonation-token" });

      // Mock window.location.reload
      window.location = { assign: vi.fn() } as any;

      openMenuAndClick("Impersonate");

      await waitFor(() => {
        expect(window.location.assign).toHaveBeenCalledWith("/");
      });
    });
  });
});
