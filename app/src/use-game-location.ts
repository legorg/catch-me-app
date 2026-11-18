import { useCallback, useEffect } from 'react';

import { ApiError, getGameState, updatePosition, type GameStateResponse } from '@/api/client';
import { stopBackgroundSharing } from '@/background-location';
import { canShareGameLocation, isActivePlayer, ownPlayer } from '@/game-state';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

type Options = {
  token: string | null | undefined;
  state: GameStateResponse | undefined;
  show: (state: GameStateResponse) => void;
  fail: (err: unknown) => void;
};

/** Shares the player's position with the game while it is on screen (and in the background). */
export function useGameLocation({ token, state, show, fail }: Options) {
  const active = !!token && canShareGameLocation(state);
  const me = ownPlayer(state);
  const retired = !!me && !isActivePlayer(me);
  // Also stop a task left registered by an earlier app run when the restored
  // session is already retired (there may be no active effect to clean up).
  useEffect(() => {
    if (retired) void stopBackgroundSharing();
  }, [retired]);
  const send = useCallback(async (fix: Fix) => {
    if (!token || !active) return;
    try {
      show(await updatePosition(token, fix));
    } catch (err) {
      // The catch may have succeeded even if its response was lost. Refresh
      // instead of repeatedly reporting a position the server will reject.
      if (err instanceof ApiError && err.status === 409) {
        try { show(await getGameState(token)); } catch (refreshError) { fail(refreshError); }
        return;
      }
      fail(err);
    }
  }, [token, active, show, fail]);

  // Not tied to focus: the jokers sheet opens over this screen and sharing must not stop meanwhile.
  return useLocationSharing(active, true, send);
}
