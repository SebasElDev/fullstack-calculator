type ClassValue = string | false | null | undefined;

/** Joins conditional class names; the whole of our styling toolkit. */
export function cn(...values: ClassValue[]): string {
  return values.filter(Boolean).join(" ");
}
