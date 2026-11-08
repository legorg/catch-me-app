import { router, useFocusEffect, useIsFocused } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useCallback, useState } from 'react';
import { ActivityIndicator, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import {
  ApiError, getGameState, leaveGame, regenerateRoute, setReady, updatePosition, type GameStateResponse,
} from '@/api/client';
import Button from '@/components/button';
import PlaygroundMap, { type CellHighlight } from '@/components/playground-map';
import {
  TEAM_LABELS, formatDuration, playerMarkers, playersByTeam, secondsUntil, secondsUntilStart,
} from '@/lobby';
import { playerSession } from '@/player-session';
import { cellAt, edgeName } from '@/playground';
import { colors, styles } from '@/styles';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

const POLL_INTERVAL_MS = 2000;
const START_COLOR = '#2f80ed';
const TARGET_COLOR = '#eb5757';

function goHome() {
  playerSession.clear();
  router.replace('/');
}

export default function LobbyScreen() {
  const [snapshot, setSnapshot] = useState<{ state: GameStateResponse; receivedAt: number } | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const token = playerSession.get();
  const focused = useIsFocused();

  const show = useCallback((state: GameStateResponse) => {
    setSnapshot({ state, receivedAt: Date.now() });
    setError('');
  }, []);

  const fail = useCallback((err: unknown) => {
    // The token is unknown (left, or the backend data was reset): start over.
    if (err instanceof ApiError && err.status === 401) {
      goHome();
      return;
    }
    setError(err instanceof Error ? err.message : 'Could not reach the backend.');
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
  const running = state?.status === 'running' || countdown === 0;
  const skipStartPositionCheck = state?.startPositionCheckDisabled === true;
  // Runners need their position before the start (start cell check); everyone shares while running.
  const sharing = focused && !!token && !!state && (state.team === 'runner' || running);
  const location = useLocationSharing(sharing, useCallback(async (fix: Fix) => {
    if (!token) return;
    try {
      show(await updatePosition(token, fix));
    } catch (err) {
      fail(err);
    }
  }, [token, show, fail]));

  async function act(action: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    try {
      await action();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  if (!token || !state || !snapshot) {
    return (
      <SafeAreaView style={styles.screen}>
        <View style={styles.content}>
          {error ? <Text style={[styles.text, styles.error]}>{error}</Text> : <ActivityIndicator color={colors.accent} />}
        </View>
      </SafeAreaView>
    );
  }

  const me = state.players.find((p) => p.playerId === state.playerId);
  const teams = playersByTeam(state.players);
  const route = state.route;
  const myCell = me?.position ? cellAt(state.playground, me.position) : undefined;
  const inStartCell = !!route && myCell === route.start;
  const nextReveal = secondsUntil(state.nextRevealAt, state, snapshot.receivedAt, now);
  const highlights: CellHighlight[] = route
    ? [{ cell: route.start, title: 'Start', color: START_COLOR }, { cell: route.end, title: 'Target', color: TARGET_COLOR }]
    : [];
  const permission = location.permission;

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>{running ? 'Game running' : countdown !== null ? `Starting in ${countdown}` : 'Lobby'}</Text>
        <Text accessibilityLiveRegion="polite" style={styles.text}>
        {skipStartPositionCheck && <View style={styles.card}>
          <Text style={[styles.heading, styles.error]}>TEST MODE ÔÇö start position check disabled</Text>
          <Text style={styles.text}>Ready and the countdown do not depend on your position. Your real GPS location is still shared normally.</Text>
        </View>}
          {running
            ? 'Catching and reaching the target arrive in the next milestone.'
            : countdown !== null
              ? skipStartPositionCheck ? 'Everyone is ready. Joining, leaving, or un-readying cancels the countdown.' : 'Everyone is ready. Joining, leaving, un-readying, or a runner leaving their start cell cancels the countdown.'
              : skipStartPositionCheck ? 'The game starts 10 seconds after all players are ready and both teams have at least one player.' : 'The game starts 10 seconds after all players are ready and both teams have at least one player. Runners must stand in their start cell.'}
        </Text>
        {running && nextReveal !== null && <View style={styles.card}>
          <Text style={styles.heading}>Next reveal in {formatDuration(nextReveal)}</Text>
          <Text style={styles.hint}>
            {state.team === 'hunter'
              ? 'Every 5 minutes you see where each runner was at that moment.'
              : 'Every 5 minutes the hunters see where you are at that moment.'}
          </Text>
        </View>}
        {running && myCell === null && <Text style={[styles.heading, styles.error]}>You are outside the playground.</Text>}
        <View style={styles.mapContainer}>
          <PlaygroundMap playground={state.playground} editable={false} onSelect={() => {}} highlights={highlights} markers={playerMarkers(state)} />
        </View>
        {running && <Text style={styles.hint}>
          ● You (green){state.team === 'runner' ? ' · teammates (blue)' : ' · teammates (blue) · revealed runners (red)'}. Runners never see hunters.
        </Text>}
        {sharing && permission && !permission.granted && <View style={styles.card}>
          <Text style={styles.heading}>Share your location</Text>
          <Text style={styles.text}>
            {state.team === 'runner' && !running && !skipStartPositionCheck
              ? 'Runners must prove they stand in their start cell before getting ready.'
              : 'Your teammates see your position while the game runs. It is only shared while this screen is open.'}
          </Text>
          {permission.canAskAgain
            ? <Button label="Allow location" onPress={() => void location.requestPermission()} />
            : <Text style={[styles.text, styles.error]}>Location access is blocked. Allow it in the system settings.</Text>}
        </View>}
        {!!location.error && <Text style={[styles.text, styles.error]}>{location.error}</Text>}
        <View style={styles.card}>
          <Text style={styles.text}>You are a <Text style={styles.heading}>{state.team}</Text>{me ? ` · ${me.name}` : ''}</Text>
          {!running && <>
            <Text style={styles.text}>Invite teammates with your team&apos;s code:</Text>
            <Text selectable style={styles.code}>{state.joinCode}</Text>
          </>}
        </View>
        {route && <View style={styles.card}>
          <Text style={styles.heading}>Your route</Text>
          <Text style={styles.text}>
            Start in <Text style={[styles.heading, { color: START_COLOR }]}>{route.start}</Text> ({edgeName(route.start)}) and reach{' '}
            <Text style={[styles.heading, { color: TARGET_COLOR }]}>{route.end}</Text> ({edgeName(route.end)}) without being caught.
          </Text>
          {!running && <Text style={[styles.text, !inStartCell && !skipStartPositionCheck && styles.error]}>
            {skipStartPositionCheck
              ? myCell ? `You are in ${myCell}. The start-position check is disabled for testing.` : 'The start-position check is disabled for testing.'
              : inStartCell
              ? '✓ You are in your start cell.'
              : myCell
                ? `You are in ${myCell}. Go to ${route.start} to get ready.`
                : me?.position ? 'You are outside the playground.' : 'Waiting for your location...'}
          </Text>}
          <Text style={styles.hint}>Only runners see routes. Hunters start anywhere.</Text>
          {!running && <Button
            label={`New route (${route.regenerationsLeft} left)`}
            variant="secondary"
            disabled={busy || me?.ready || route.regenerationsLeft === 0}
            onPress={() => void act(async () => show(await regenerateRoute(token)))}
          />}
          {!running && me?.ready && route.regenerationsLeft > 0 && <Text style={styles.hint}>Set yourself not ready to change your route.</Text>}
        </View>}
        {(['runner', 'hunter'] as const).map((team) => (
          <View key={team} style={styles.card}>
            <Text style={styles.heading}>{TEAM_LABELS[team]} ({teams[team].length})</Text>
            {teams[team].length === 0 && <Text style={styles.hint}>Nobody yet.</Text>}
            {teams[team].map((p) => (
              <Text key={p.playerId} style={styles.text}>
                {p.ready ? '✓' : '○'} {p.name}{p.playerId === state.playerId ? ' (you)' : ''}
                {running ? '' : p.ready ? ' · ready' : ' · not ready'}
                {p.route && p.playerId !== state.playerId ? ` · ${p.route.start} → ${p.route.end}` : ''}
              </Text>
            ))}
          </View>
        ))}
        {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
        {!running && me && <View style={styles.row}>
          <Button
            label={me.ready ? 'Not ready' : 'Ready'}
            variant={me.ready ? 'secondary' : 'primary'}
            disabled={busy || (!skipStartPositionCheck && !me.ready && !!route && !inStartCell)}
            onPress={() => void act(async () => show(await setReady(token, !me.ready)))}
          />
          <Button label="Leave" variant="secondary" disabled={busy} onPress={() => void act(async () => {
            await leaveGame(token);
            goHome();
          })} />
        </View>}
        {running && <Button label="Exit to home" variant="secondary" onPress={goHome} />}
      </ScrollView>
    </SafeAreaView>
  );
}
