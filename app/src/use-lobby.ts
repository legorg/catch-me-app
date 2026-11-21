import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';

import { ApiError, leaveGame, regenerateRoute, setHunterTeam, setReady, updatePosition } from '@/api/client';
import { stopBackgroundSharing } from '@/background-location';
import { inform } from '@/confirm';
import { canShareLobbyLocation, isSpectator } from '@/game-state';
import { translate } from '@/i18n';
import { goHome, useGameState } from '@/use-game-state';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

/** State and actions of the lobby screen: the game state, location sharing, readiness and the hunter team. */
export function useLobby() {
  const game = useGameState();
  const { token, state, focused, show, fail, running } = game;
  const [busy, setBusy] = useState(false);
  const [changingRoute, setChangingRoute] = useState(false);
  const [newTeam, setNewTeam] = useState('');

  // The game has its own screen; joining it replaces the lobby.
  useEffect(() => {
    if (running) router.replace('/game');
  }, [running]);

  const spectator = isSpectator(state);
  // Also clear any task left registered before restoring a spectator session.
  useEffect(() => {
    if (spectator) void stopBackgroundSharing();
  }, [spectator]);

  // Playing participants share from joining; spectators never report locations.
  const participating = canShareLobbyLocation(state);
  const sharing = focused && !!token && participating;
  // The lobby shares in the foreground only; background sharing belongs to the game screen.
  const location = useLocationSharing(sharing, false, useCallback(async (fix: Fix) => {
    if (!token || !sharing) return;
    try {
      show(await updatePosition(token, fix));
    } catch (err) {
      fail(err);
    }
  }, [token, sharing, show, fail]));

  // A refused action is shown in a system alert: the next poll would clear the screen's error at once.
  async function act(action: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    try {
      await action();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) fail(err);
      else inform(translate('lobby.actionFailed'), err instanceof Error ? err.message : translate('common.backendUnreachable'));
    } finally {
      setBusy(false);
    }
  }

  const me = state?.players.find((p) => p.playerId === state.playerId);
  const { permission, background } = location;
  // Installed builds need "Always"; Expo Go cannot share in the background, so foreground access must do there.
  // ("Always" can only be granted on top of foreground access, so it covers both.)
  const needsBackground = participating && background.available && background.permission?.granted !== true;
  const needsForeground = participating && !background.available && permission?.granted !== true;
  const needsHunterTeam = state?.team === 'hunter' && !me?.hunterTeam;

  const changeHunterTeam = (name: string) => {
    void act(async () => {
      if (!token || !participating || state?.team !== 'hunter') return;
      show(await setHunterTeam(token, name));
      setNewTeam('');
    });
  };

  const createTeam = () => changeHunterTeam(newTeam);

  const toggleReady = () => {
    void act(async () => {
      if (!token || !me || !participating) return;
      show(await setReady(token, !me.ready));
    });
  };

  const changeRoute = () => {
    void act(async () => {
      if (!token || !participating || state?.team !== 'runner') return;
      setChangingRoute(true);
      try {
        show(await regenerateRoute(token));
      } finally {
        setChangingRoute(false);
      }
    });
  };

  const leave = () => {
    void act(async () => {
      if (!token) return;
      await leaveGame(token);
      goHome();
    });
  };

  const requestLocation = () => {
    if (participating) void location.requestPermission();
  };

  const requestBackground = () => {
    if (participating) void background.request();
  };

  return {
    ...game, busy, changingRoute, newTeam, setNewTeam, sharing, location, me, spectator,
    needsBackground, needsForeground, needsHunterTeam,
    changeHunterTeam, createTeam, toggleReady, changeRoute, leave, requestLocation, requestBackground,
  };
}
