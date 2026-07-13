/**
 * Calculates the percentage change between two numbers.
 * Returns undefined if either number is not provided.
 */
export const calculateChange = (
  current?: number,
  previous?: number
): number | undefined => {
  if (!previous || !current) {
    return undefined;
  }

  return Math.round(((current - previous) / previous) * 100);
};
