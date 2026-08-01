/**
 * Helpers for the `datetime` filter kind. Values are kept as the string a
 * native `datetime-local` input produces (`YYYY-MM-DDTHH:mm`, local time) so a
 * value read back from the URL can be rendered straight into the input.
 */

const pad = (n: number) => String(n).padStart(2, "0");

export const toLocalInputValue = (date: Date): string =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
  `T${pad(date.getHours())}:${pad(date.getMinutes())}`;

export const minutesAgo = (minutes: number): string =>
  toLocalInputValue(new Date(Date.now() - minutes * 60 * 1000));

export interface DateTimePreset {
  text: string;
  /** Resolved when the preset is picked, not when it is rendered. */
  value: () => string;
}

export const relativePresets: DateTimePreset[] = [
  { text: "1 hour ago", value: () => minutesAgo(60) },
  { text: "6 hours ago", value: () => minutesAgo(60 * 6) },
  { text: "24 hours ago", value: () => minutesAgo(60 * 24) },
  { text: "7 days ago", value: () => minutesAgo(60 * 24 * 7) },
  { text: "30 days ago", value: () => minutesAgo(60 * 24 * 30) },
];

export const nowPreset: DateTimePreset = {
  text: "Now",
  value: () => toLocalInputValue(new Date()),
};

export const formatDateTime = (value: string): string => {
  const ms = new Date(value).getTime();

  return isNaN(ms) ? value : new Date(ms).toLocaleString();
};

/**
 * The API takes time bounds as unix seconds. Anything unparseable is dropped
 * rather than sent as NaN, which the API would read as "no bound".
 */
export const toUnixSeconds = (value: string): string => {
  const ms = new Date(value).getTime();

  return isNaN(ms) ? "" : String(Math.floor(ms / 1000));
};
