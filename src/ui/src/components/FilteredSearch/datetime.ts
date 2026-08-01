/**
 * Helpers for the `datetime` filter kind. Values are stored as an ISO string
 * that carries the author's UTC offset (`YYYY-MM-DDTHH:mm+HH:MM`). A bare
 * `datetime-local` string would be read as the *reader's* local time, so a
 * shared link would silently denote a different absolute window for a
 * colleague in another timezone.
 */

const pad = (n: number) => String(n).padStart(2, "0");

/** The wall-clock half, in the shape a native `datetime-local` input takes. */
export const toLocalInputValue = (date: Date): string =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
  `T${pad(date.getHours())}:${pad(date.getMinutes())}`;

const utcOffset = (date: Date): string => {
  const minutes = -date.getTimezoneOffset();
  const abs = Math.abs(minutes);

  return `${minutes < 0 ? "-" : "+"}${pad(Math.floor(abs / 60))}:${pad(abs % 60)}`;
};

export const toStoredValue = (date: Date): string =>
  toLocalInputValue(date) + utcOffset(date);

/** Renders a stored value back into a `datetime-local` input. */
export const toInputValue = (value: string): string => {
  const date = new Date(value);

  return isNaN(date.getTime()) ? "" : toLocalInputValue(date);
};

/**
 * Pins a value the user typed or picked to the offset it was written in.
 * Unparseable input passes through untouched so `isValidDateTime` rejects it.
 */
export const normalizeDateTime = (value: string): string => {
  const date = new Date(value);

  return isNaN(date.getTime()) ? value : toStoredValue(date);
};

export const minutesAgo = (minutes: number): string =>
  toStoredValue(new Date(Date.now() - minutes * 60 * 1000));

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
  value: () => toStoredValue(new Date()),
};

export const isValidDateTime = (value: string): boolean =>
  !isNaN(new Date(value).getTime());

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
