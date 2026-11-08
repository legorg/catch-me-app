// The player token lives in memory only, so it is lost when the app restarts.
// Never log it; it is the player's credential for this game.
let token: string | null = null;

export const playerSession = {
  get: () => token,
  set: (value: string) => { token = value; },
  clear: () => { token = null; },
};
