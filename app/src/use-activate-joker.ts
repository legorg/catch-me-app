import { router } from 'expo-router';

import { activateJoker, type GameStateResponse, type JokerKind } from '@/api/client';

/** Activates a joker of the player's team and closes the jokers sheet; failures go to `fail`. */
export function useActivateJoker(
  token: string | null,
  show: (state: GameStateResponse) => void,
  fail: (err: unknown) => void,
) {
  return async (kind: JokerKind, cell?: string) => {
    if (!token) return;
    try {
      show(await activateJoker(token, kind, cell));
      // The map behind shows the result: the game screen reloads the state when the sheet closes.
      router.back();
    } catch (err) {
      fail(err);
    }
  };
}
