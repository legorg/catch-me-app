import { router } from 'expo-router';
import { useEffect, useState } from 'react';

import { ApiError, getGameState } from '@/api/client';
import { playerSession } from '@/player-session';
import { gameDestination } from '@/game-state';
import { goToGameResult } from '@/use-game-state';

/**
 * On app start, rejoins the game saved from the last run: the game screen while it is running,
 * otherwise the lobby, or the result if it has ended. Unknown tokens are forgotten.
 * Returns true while it is still checking.
 */
export function useResumeGame() {
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    let current = true;
    void (async () => {
      const token = await playerSession.restore();
      if (token) {
        try {
          const state = await getGameState(token);
          if (current) {
            if (state.status === 'ended') goToGameResult(state.sessionId);
            else router.replace(gameDestination(state));
          }
          return;
        } catch (err) {
          if (err instanceof ApiError && err.status === 401) playerSession.clear();
        }
      }
      if (current) setChecking(false);
    })();
    return () => { current = false; };
  }, []);

  return checking;
}
