import SessionStorage, { fallbackStore } from "./SessionStorage";

describe("SessionStorage", () => {
  beforeEach(() => {
    SessionStorage.clear();
  });

  test.skip("should set/get an item", () => {
    SessionStorage.set("my", "key");
    expect(SessionStorage.get("my")).toBe("key");
    expect(SessionStorage("my")).toBe("key");

    SessionStorage("abc", { a: {} });
    expect(SessionStorage.get("abc")).toEqual({ a: {} });

    expect(SessionStorage.get("non-set")).toBe(null);
    expect(SessionStorage.get("non-set", "abc")).toBe("abc");
  });

  test.skip("when sessionStorage is full should fallback to global store", () => {
    window.sessionStorage.isFull(true);
    expect(SessionStorage.set("my-2", "key-2"));
    expect(fallbackStore.getItem("my-2")).toBe(JSON.stringify("key-2"));
    window.sessionStorage.isFull(false);
  });

  test.skip("should remove an item", () => {
    SessionStorage.set("abc", { a: [1, 2, 3] });
    SessionStorage.del("abc");
    expect(SessionStorage.get("abc")).toBe(null);
  });

  test.skip("[fallbackStorage] should work", () => {
    expect(fallbackStore.getItem("some-key")).toBe(null);
    fallbackStore.setItem("some-key", "some-value");
    expect(fallbackStore.getItem("some-key")).toBe("some-value");
    fallbackStore.removeItem("some-key");
    expect(fallbackStore.getItem("some-key")).toBe(null);
    fallbackStore.setItem("some-key", "some-value");
    expect(fallbackStore.getItem("some-key")).toBe("some-value");
    fallbackStore.clear();
    expect(fallbackStore.getItem("some-key")).toBe(null);
  });
});
