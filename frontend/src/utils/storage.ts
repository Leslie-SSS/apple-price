/**
 * Local storage utilities for Bark Key management
 * The Bark Key is stored locally to avoid frequent input
 */

const BARK_KEY_KEY = "apple-price-bark-key";

export const storage = {
  getBarkKey: (): string => {
    try {
      return localStorage.getItem(BARK_KEY_KEY) || "";
    } catch {
      return "";
    }
  },

  setBarkKey: (key: string): void => {
    try {
      localStorage.setItem(BARK_KEY_KEY, key);
    } catch {
      console.error("Failed to save Bark Key to localStorage");
    }
  },

  hasBarkKey: (): boolean => {
    return storage.getBarkKey().length > 0;
  },

  clearBarkKey: (): void => {
    try {
      localStorage.removeItem(BARK_KEY_KEY);
    } catch {
      console.error("Failed to clear Bark Key from localStorage");
    }
  },
};

/**
 * Mask a Bark Key for display (shows first 4 and last 4 chars)
 */
export function maskBarkKey(key: string): string {
  if (!key) return "";
  if (key.length <= 8) return "****";
  return key.slice(0, 4) + "****" + key.slice(-4);
}
