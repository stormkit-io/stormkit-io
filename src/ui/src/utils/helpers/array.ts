export function toArray<T>(v: T | Array<T>): Array<T> {
  return (Array.isArray(v) ? v : [v]).filter(i => typeof i !== "undefined");
}
