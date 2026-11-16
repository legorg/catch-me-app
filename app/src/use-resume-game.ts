import { router } from 'expo-router';
import { useEffect, useState } from 'react';

import { ApiError, getGameState } from '@/api/client';
import { playerSession } from '@/player-session';

/**
 * On app start, rejoins the game saved from the last run: the game screen while it is running,
 * otherwise the lobby. A token the backend no longer knows is forgotten.
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
          if (current) router.replace(state.status === 'running' ? '/game' : '/lobby');
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
