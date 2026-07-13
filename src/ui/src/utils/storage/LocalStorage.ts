import SessionStorage, { fallbackStore } from "./SessionStorage";

const w = typeof window !== "undefined";

if (w && "localStorage" in window === false) {
  throw new Error("Browser unsupported. Please upgrade your browser");
}

const storage = w ? window.localStorage : fallbackStore;

const LocalStorage = {
  // Set a key-value pair. The value will be automatically JSON.stringify-ed.
  set<T>(key: string, value: T): T {
    try {
      storage.setItem(key, JSON.stringify(value));
    } catch (e) {
      LocalStorage.fallbackStorage.set(key, value);
    }

    return value;
  },

  // Return the stored value. The return value will be JSON.parse-d automatically.
  get<T>(key: string, retVal?: T): T | undefined {
    try {
      const value = storage.getItem(key);
      return (JSON.parse(value) as T) || retVal;
    } catch (e) {
      return retVal;
    }
  },

  // Delete the key from localStorage.
  del(key: string) {
    storage.removeItem(key);
  },

  // Clear the local storage.
  clear() {
    storage.clear();
  },

  fallbackStorage: SessionStorage,
};

export default LocalStorage;
