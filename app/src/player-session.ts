import * as SecureStore from 'expo-secure-store';

// The token is kept in memory for synchronous reads and mirrored to the device's secure store,
// so the player can rejoin after an app restart. Never log it; it is the player's credential for this game.
const STORAGE_KEY = 'playerToken';
let token: string | null = null;

// Storage can fail (e.g. a locked keychain): the in-memory token still works for this app run.
const persist = (action: () => Promise<void>) => { action().catch(() => undefined); };

export const playerSession = {
  get: () => token,
  set: (value: string) => {
    token = value;
    persist(() => SecureStore.setItemAsync(STORAGE_KEY, value));
  },
  clear: () => {
    token = null;
    persist(() => SecureStore.deleteItemAsync(STORAGE_KEY));
  },
  /** Loads the token saved by an earlier app run, unless one was set in the meantime. */
  restore: async () => {
    try {
      token ??= await SecureStore.getItemAsync(STORAGE_KEY);
    } catch {
      // Nothing saved or unreadable: start fresh.
    }
    return token;
  },
};
