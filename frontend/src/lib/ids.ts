/**
 * Client-side identifier generation for list keys (history entries).
 *
 * `crypto.randomUUID` exists only in secure contexts, so a build served over
 * plain HTTP from a non-localhost host would throw `TypeError` on every call.
 * Identifiers are cosmetic — React keys and history rows — so a per-factory
 * counter is a perfectly good fallback, and it keeps a missing API from being
 * mistaken for a failed calculation.
 */

/** Separates the prefix from the sequence number in a fallback identifier. */
const ID_SEPARATOR = "-";

/**
 * Returns a generator of identifiers unique within this browsing session.
 *
 * @param prefix namespace for the fallback identifiers, e.g. `history`.
 */
export function createIdFactory(prefix: string): () => string {
  let sequence = 0;
  return () => {
    const uuid = globalThis.crypto?.randomUUID?.();
    if (uuid !== undefined) {
      return uuid;
    }
    sequence += 1;
    return `${prefix}${ID_SEPARATOR}${sequence}`;
  };
}
