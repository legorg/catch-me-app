import { router, useFocusEffect, useIsFocused } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';

import { ApiError, getGameState, type GameStateResponse } from '@/api/client';
import { translate } from '@/i18n';
import { gameDestination, shouldAcceptGameState } from '@/game-state';
import { gameResultRoute } from '@/game-result';
import { stopBackgroundSharing } from '@/background-location';
import { secondsUntilStart } from '@/lobby';
import { playerSession } from '@/player-session';

const POLL_INTERVAL_MS = 2000;

export function goHome() {
  playerSession.clear();
  router.replace('/');
}

/** Replace the entire gameplay stack, including an open joker modal. */
export function goToGameResult(sessionId: string) {
  void stopBackgroundSharing();
  playerSession.clear(); // Do not resume the ended game when returning home.
  // Joining replaces the lobby/game screen, so the stack may already be at its root.
  if (router.canDismiss()) router.dismissAll();
  router.replace(gameResultRoute(sessionId));
}

/**
 * Polls the game state of the current player while the screen is focused.
 * `now` ticks every 250 ms so countdowns can be derived from the snapshot.
 */
export function useGameState() {
  const [snapshot, setSnapshot] = useState<{ state: GameStateResponse; receivedAt: number } | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [error, setError] = useState('');
  const token = playerSession.get();
  const focused = useIsFocused();

  const show = useCallback((state: GameStateResponse) => {
    const receivedAt = Date.now();
    setSnapshot((previous) => shouldAcceptGameState(previous?.state, state) ? { state, receivedAt } : previous);
    setError('');
  }, []);

  const fail = useCallback((err: unknown) => {
    // The token is unknown (left, or the backend data was reset): start over.
    if (err instanceof ApiError && err.status === 401) {
      goHome();
      return;
    }
    setError(err instanceof Error ? err.message : translate('common.backendUnreachable'));
  }, []);

  useFocusEffect(useCallback(() => {
    if (!token) {
      router.replace('/');
      return;
    }
    let active = true;
    const poll = () => getGameState(token).then(
      (state) => { if (active) show(state); },
      (err: unknown) => { if (active) fail(err); },
    );
    void poll();
    const pollTimer = setInterval(() => void poll(), POLL_INTERVAL_MS);
    const clock = setInterval(() => setNow(Date.now()), 250);
    return () => {
      active = false;
      clearInterval(pollTimer);
      clearInterval(clock);
    };
  }, [token, show, fail]));

  const state = snapshot?.state;
  const countdown = state && snapshot ? secondsUntilStart(state, snapshot.receivedAt, now) : null;
  const running = !!state && gameDestination(state, countdown) === '/game';

  useEffect(() => {
    if (focused && state?.status === 'ended') goToGameResult(state.sessionId);
  }, [focused, state?.status, state?.sessionId]);

  return { token, snapshot, state, now, error, focused, show, fail, countdown, running };
}

/** Back in the lobby if the game is not running (e.g. a countdown that was cancelled). */
export function useLobbyUnlessRunning(state: GameStateResponse | undefined, running: boolean) {
  useEffect(() => {
    if (state && state.status !== 'ended' && !running) router.replace('/lobby');
  }, [state, running]);
}
