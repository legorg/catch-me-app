import { router } from 'expo-router';

import { ApiError, activateJoker, type GameStateResponse, type JokerRequest } from '@/api/client';
import { inform } from '@/confirm';
import { translate } from '@/i18n';

/**
 * Activates a joker of the player's team and closes the jokers sheet. A refused joker is shown in a system alert,
 * since the next poll would clear the screen's error at once; an unknown token goes to `fail`.
 */
export function useActivateJoker(
  token: string | null,
  show: (state: GameStateResponse) => void,
  fail: (err: unknown) => void,
) {
  return async (joker: JokerRequest) => {
    if (!token) return;
    try {
      show(await activateJoker(token, joker));
      // The map behind shows the result: the game screen reloads the state when the sheet closes.
      router.back();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        fail(err);
        return;
      }
      inform(translate('joker.failed'), err instanceof Error ? err.message : translate('common.backendUnreachable'));
    }
  };
}
