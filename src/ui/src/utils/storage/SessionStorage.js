const w = typeof window !== "undefined";

// istanbul ignore if
if (w && "sessionStorage" in window === false) {
  throw new Error("Browser unsupported. Please upgrade your browser");
}

// istanbul ignore next
export const fallbackStore = {
  _cache: {},
  /**
   * Return item with given key.
   *
   * @param key
   */
  getItem: key => fallbackStore._cache[key] || null,

  /**
   * Set key-value pair.
   *
   * @param key
   * @param value
   */
  setItem: (key, value) => {
    fallbackStore._cache[key] = value;
  },

  /**
   * Remove item.
   *
   * @param key
   */
  removeItem: key => {
    delete fallbackStore._cache[key];
  },

  /**
   * Clear the cache.
   */
  clear: () => {
    fallbackStore._cache = {};
  },
};

const UNDEF = void 0;
const storage = w
  ? window.sessionStorage
  : /* istanbul ignore next */ fallbackStore;

/**
 * The constructor. If value is provided, the function will be used as a setter,
 * otherwise a getter.
 *
 * @param {string} key
 * @param {*|=} value
 * @return {*}
 * @constructor
 */
const SessionStorage = (key, value) => {
  if (value !== UNDEF) {
    return SessionStorage.set(key, value);
  } else {
    return SessionStorage.get(key);
  }
};

/**
 * Set a key-value pair. The value will be automatically JSON.stringify-ed.
 *
 * @param {string} key
 * @param {*} value
 * @return {*} The value provided
 */
SessionStorage.set = (key, value) => {
  try {
    storage.setItem(key, JSON.stringify(value));
  } catch (e) {
    fallbackStore.setItem(key, JSON.stringify(value));
  }

  return value;
};

/**
 * Return the stored value. The return value will be JSON.parse-d automatically.
 *
 * @param {string} key
 * @param {*|=} retVal The default return value.
 * @return {*} A parsed json object or null if value does not exist
 */
SessionStorage.get = (key, retVal = null) => {
  try {
    const value = storage.getItem(key);
    return JSON.parse(value) || /* istanbul ignore next */ retVal;
  } catch (e) {
    return retVal;
  }
};

/**
 * Delete the key from sessionStorage.
 *
 * @param {string} key
 * @return {*} The deleted value
 */
SessionStorage.del = key => {
  const value = SessionStorage.get(key);
  storage.removeItem(key);
  return value;
};

/**
 * Clear the local storage.
 */
SessionStorage.clear = storage.clear;

export default SessionStorage;
