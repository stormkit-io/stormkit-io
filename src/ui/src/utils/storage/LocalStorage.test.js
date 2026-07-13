import LocalStorage from "./LocalStorage";

describe("LocalStorage", () => {
  beforeEach(() => {
    LocalStorage.clear();
  });

  test.skip("should set/get an item", () => {
    LocalStorage.set("my", "key");
    expect(LocalStorage.get("my")).toBe("key");
    expect(LocalStorage("my")).toBe("key");

    LocalStorage("abc", { a: {} });
    expect(LocalStorage.get("abc")).toEqual({ a: {} });

    expect(LocalStorage.get("non-set", "defaultReturn")).toBe("defaultReturn");
    expect(LocalStorage.get(undefined, {})).toEqual({});
  });

  test.skip("when localStorage is full should fallback to session storage", () => {
    window.localStorage.isFull(true);
    expect(LocalStorage.set("my-2", "key-2")).toBe("key-2");
    window.localStorage.isFull(false);
  });

  test.skip("should remove an item", () => {
    LocalStorage.set("abc", { a: [1, 2, 3] });
    LocalStorage.del("abc");
    expect(LocalStorage.get("abc")).toBe(null);
  });
});
