import { useCallback } from 'react';

import { updatePosition, type GameStateResponse } from '@/api/client';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

type Options = {
  token: string | null | undefined;
  state: GameStateResponse | undefined;
  show: (state: GameStateResponse) => void;
  fail: (err: unknown) => void;
};

/** Shares the player's position with the game while it is on screen (and in the background). */
export function useGameLocation({ token, state, show, fail }: Options) {
  const send = useCallback(async (fix: Fix) => {
    if (!token) return;
    try {
      show(await updatePosition(token, fix));
    } catch (err) {
      fail(err);
    }
  }, [token, show, fail]);

  // Not tied to focus: the jokers sheet opens over this screen and sharing must not stop meanwhile.
  return useLocationSharing(!!token && !!state, true, send);
}
