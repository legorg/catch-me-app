import { useEffect } from 'react';

import { type GameStateResponse } from '@/api/client';
import { stopBackgroundSharing } from '@/background-location';
import { canShareGameLocation, isActivePlayer, isSpectator, ownPlayer } from '@/game-state';
import { useLocationSharing } from '@/use-location-sharing';

type Options = {
  token: string | null | undefined;
  state: GameStateResponse | undefined;
};

/**
 * Shares the player's position with the game while it is on screen (and in the
 * background); the state poll reports it (see pending-location).
 */
export function useGameLocation({ token, state }: Options) {
  const active = !!token && canShareGameLocation(state);
  const me = ownPlayer(state);
  const stopped = state?.status === 'ended' || isSpectator(state) || (!!me && !isActivePlayer(me));
  // Also stop a task left registered by an earlier app run when the restored
  // session is already retired or ended (there may be no active effect to clean up).
  useEffect(() => {
    if (stopped) void stopBackgroundSharing();
  }, [stopped]);

  // Not tied to focus: the jokers sheet opens over this screen and sharing must not stop meanwhile.
  return useLocationSharing(active, true);
}
