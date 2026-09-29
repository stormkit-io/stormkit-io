import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import openPopup from "./popup";

describe("~/utils/helpers/popup.ts", () => {
  const popup = { close: vi.fn(), closed: false } as unknown as Window;
  let onClose: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.useFakeTimers();
    onClose = vi.fn();
    vi.spyOn(window, "open").mockReturnValue(popup);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  const post = (source: Window | null) => {
    window.dispatchEvent(
      new MessageEvent("message", {
        data: { success: true, sessionToken: "token" },
        source,
      })
    );

    vi.advanceTimersByTime(1000);
  };

  it("should accept the result from the popup it opened", () => {
    openPopup({
      url: "https://api.example.org/auth/github",
      title: "Login",
      onClose,
    });
    post(popup);

    expect(onClose).toHaveBeenCalledWith({
      success: true,
      sessionToken: "token",
    });
  });

  it("should ignore results from other windows", () => {
    openPopup({
      url: "https://api.example.org/auth/github",
      title: "Login",
      onClose,
    });
    post(window);

    expect(onClose).not.toHaveBeenCalled();
  });
});
