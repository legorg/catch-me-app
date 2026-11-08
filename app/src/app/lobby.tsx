import { router, useFocusEffect } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useCallback, useState } from 'react';
import { ActivityIndicator, ScrollView, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ApiError, getGameState, leaveGame, setReady, type GameStateResponse } from '@/api/client';
import Button from '@/components/button';
import PlaygroundMap from '@/components/playground-map';
import { TEAM_LABELS, playersByTeam, secondsUntilStart } from '@/lobby';
import { playerSession } from '@/player-session';
import { colors, styles } from '@/styles';

const POLL_INTERVAL_MS = 2000;

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

  if (!token || !snapshot) {
    return (
      <SafeAreaView style={styles.screen}>
        <View style={styles.content}>
          {error ? <Text style={[styles.text, styles.error]}>{error}</Text> : <ActivityIndicator color={colors.accent} />}
        </View>
      </SafeAreaView>
    );
  }

  const { state, receivedAt } = snapshot;
  const me = state.players.find((p) => p.playerId === state.playerId);
  const countdown = secondsUntilStart(state, receivedAt, now);
  const running = state.status === 'running' || countdown === 0;
  const teams = playersByTeam(state.players);

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>{running ? 'Game running' : countdown !== null ? `Starting in ${countdown}` : 'Lobby'}</Text>
        <Text accessibilityLiveRegion="polite" style={styles.text}>
          {running
            ? 'The game has started. Location sharing and the game rules arrive in the next milestones.'
            : countdown !== null
              ? 'Everyone is ready. Joining, leaving, or un-readying cancels the countdown.'
              : 'The game starts 10 seconds after all players are ready and both teams have at least one player.'}
        </Text>
        <View style={styles.card}>
          <Text style={styles.text}>You are a <Text style={styles.heading}>{state.team}</Text>{me ? ` · ${me.name}` : ''}</Text>
          {!running && <>
            <Text style={styles.text}>Invite teammates with your team&apos;s code:</Text>
            <Text selectable style={styles.code}>{state.joinCode}</Text>
          </>}
        </View>
        {(['runner', 'hunter'] as const).map((team) => (
          <View key={team} style={styles.card}>
            <Text style={styles.heading}>{TEAM_LABELS[team]} ({teams[team].length})</Text>
            {teams[team].length === 0 && <Text style={styles.hint}>Nobody yet.</Text>}
            {teams[team].map((p) => (
              <Text key={p.playerId} style={styles.text}>
                {p.ready ? '✓' : '○'} {p.name}{p.playerId === state.playerId ? ' (you)' : ''}{running ? '' : p.ready ? ' · ready' : ' · not ready'}
              </Text>
            ))}
          </View>
        ))}
        {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
        {!running && me && <View style={styles.row}>
          <Button
            label={me.ready ? 'Not ready' : 'Ready'}
            variant={me.ready ? 'secondary' : 'primary'}
            disabled={busy}
            onPress={() => void act(async () => show(await setReady(token, !me.ready)))}
          />
          <Button label="Leave" variant="secondary" disabled={busy} onPress={() => void act(async () => {
            await leaveGame(token);
            goHome();
          })} />
        </View>}
        {running && <Button label="Exit to home" variant="secondary" onPress={goHome} />}
        <View style={styles.mapContainer}>
          <PlaygroundMap playground={state.playground} editable={false} onSelect={() => {}} />
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
