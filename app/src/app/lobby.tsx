import { Stack, router, useFocusEffect, useIsFocused } from 'expo-router';
import { useCallback, useState } from 'react';
import { ActivityIndicator, Text, TextInput, View } from 'react-native';

import {
  ApiError, MAX_NAME_LENGTH, getGameState, leaveGame, setHunterTeam, setReady, updatePosition,
  type GameStateResponse,
} from '@/api/client';
import Button from '@/components/button';
import JoinCodes from '@/components/join-codes';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import FullscreenMap from '@/components/fullscreen-map';
import { type CellHighlight, type MapMarker } from '@/components/playground-map';
import { translate, translateParts, useT } from '@/i18n';
import {
  formatDuration, hunterTeams, playerMarkers, playersByTeam, secondsUntil, secondsUntilStart,
} from '@/lobby';
import { playerSession } from '@/player-session';
import { cellAt, distanceMeters, START_RADIUS_METERS } from '@/playground';
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
  const t = useT();
  const [snapshot, setSnapshot] = useState<{ state: GameStateResponse; receivedAt: number } | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [newTeam, setNewTeam] = useState('');
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
  const running = state?.status === 'running' || countdown === 0;
  const skipStartPositionCheck = state?.startPositionCheckDisabled === true;
  // Everyone shares from joining: runners for the start cell check, hunters for their hunter team.
  const sharing = focused && !!token && !!state;
  // Background sharing only while the game runs; the lobby stays foreground-only.
  const location = useLocationSharing(sharing, running, useCallback(async (fix: Fix) => {
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
    // A ScrollView must exist from the first render, or the large header never links to it.
    return (
      <>
        <KeyboardScrollView>
          {error ? <Text style={[styles.text, styles.error]}>{error}</Text> : <ActivityIndicator color={colors.accent} />}
        </KeyboardScrollView>
      </>
    );
  }

  const me = state.players.find((p) => p.playerId === state.playerId);
  const teams = playersByTeam(state.players);
  const route = state.route;
  const myCell = me?.position ? cellAt(state.playground, me.position) : undefined;
  const isRunner = state.team === 'runner';
  // The server only sends the route to runners.
  const metersToStart = route && me?.position ? distanceMeters(me.position, route.start.coordinates) : null;
  const atStart = metersToStart !== null && metersToStart <= START_RADIUS_METERS;
  const nextReveal = secondsUntil(state.nextRevealAt, state, snapshot.receivedAt, now);
  const startCell = route ? cellAt(state.playground, route.start.coordinates) : null;
  const stopCell = route ? cellAt(state.playground, route.stop.coordinates) : null;
  const highlights: CellHighlight[] = route ? [
    ...(startCell ? [{ cell: startCell, title: route.start.title || t('lobby.start'), color: START_COLOR }] : []),
    ...(stopCell ? [{ cell: stopCell, title: route.stop.title || t('lobby.target'), color: TARGET_COLOR }] : []),
  ] : [];
  const routeMarkers: MapMarker[] = route ? [
    { id: 'route-start', coordinate: route.start.coordinates, title: route.start.title || t('lobby.start'), description: t('lobby.start'), color: START_COLOR },
    { id: 'route-stop', coordinate: route.stop.coordinates, title: route.stop.title || t('lobby.target'), description: t('lobby.target'), color: TARGET_COLOR },
  ] : [];
  const permission = location.permission;
  const needsHunterTeam = state.team === 'hunter' && !me?.hunterTeam;
  const changeHunterTeam = (name: string) => void act(async () => {
    show(await setHunterTeam(token, name));
    setNewTeam('');
  });

  return (
    <>
      <KeyboardScrollView>
        {skipStartPositionCheck && <View style={styles.card}>
          <Text style={[styles.heading, styles.error]}>{t('lobby.testMode')}</Text>
          <Text style={styles.text}>{t('lobby.testModeText')}</Text>
        </View>}
        <Text accessibilityLiveRegion="polite" style={styles.text}>
          {running
            ? t('lobby.runningInfo')
            : countdown !== null
              ? t(skipStartPositionCheck ? 'lobby.countdownInfoTest' : 'lobby.countdownInfo')
              : t(skipStartPositionCheck ? 'lobby.startInfoTest' : 'lobby.startInfo')}
        </Text>
        {running && nextReveal !== null && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.nextReveal', { time: formatDuration(nextReveal) })}</Text>
          <Text style={styles.hint}>{t(state.team === 'hunter' ? 'lobby.revealHunter' : 'lobby.revealRunner')}</Text>
        </View>}
        {running && myCell === null && <Text style={[styles.heading, styles.error]}>{t('lobby.outside')}</Text>}
        <View style={styles.mapContainer}>
          <FullscreenMap playground={state.playground} editable={false} onSelect={() => {}} highlights={highlights} markers={[...routeMarkers, ...playerMarkers(state).filter((marker) => marker.id !== state.playerId)]} />
        </View>
        {(running || state.team === 'hunter') && <Text style={styles.hint}>
          {t(state.team === 'runner' ? 'lobby.legendRunner' : 'lobby.legendHunter')}
        </Text>}
        {sharing && permission && !permission.granted && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.shareHeading')}</Text>
          <Text style={styles.text}>
            {t(state.team === 'runner' && !running && !skipStartPositionCheck
              ? 'lobby.shareRunnerStart'
              : state.team === 'hunter' && !running
                ? 'lobby.shareHunterLobby'
                : location.background.available ? 'lobby.shareRunning' : 'lobby.shareRunningForeground')}
          </Text>
          {permission.canAskAgain
            ? <Button label={t('lobby.allowLocation')} onPress={() => void location.requestPermission()} />
            : <Text style={[styles.text, styles.error]}>{t('lobby.locationBlocked')}</Text>}
        </View>}
        {running && permission?.granted && <View style={styles.card}>
          <Text style={styles.heading}>{t(location.background.active ? 'lobby.sharingBackground' : 'lobby.sharingForeground')}</Text>
          {location.background.active
            ? <Text style={styles.text}>{t('lobby.sharingBackgroundText')}</Text>
            : !location.background.available
              ? <Text style={styles.hint}>{t('lobby.backgroundUnavailable')}</Text>
              : location.background.permission?.canAskAgain !== false
                ? <>
                  <Text style={styles.text}>{t('lobby.allowAlways')}</Text>
                  <Button label={t('lobby.shareInBackground')} variant="secondary" onPress={() => void location.background.request()} />
                </>
                : <Text style={styles.hint}>{t('lobby.backgroundBlocked')}</Text>}
        </View>}
        {!!location.error && <Text style={[styles.text, styles.error]}>{location.error}</Text>}
        <View style={styles.card}>
          <Text style={styles.text}>
            {translateParts('lobby.youAre', {
              team: <Text key="team" style={styles.heading}>{t(state.team === 'runner' ? 'team.runner' : 'team.hunter')}</Text>,
            })}
            {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
          </Text>
        </View>
        {!running && <JoinCodes codes={state.joinCodes} />}
        {route && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.routeHeading')}</Text>
          <Text style={styles.text}>
            {translateParts('lobby.route', {
              start: <Text key="start" style={[styles.heading, { color: START_COLOR }]}>{route.start.title || t('lobby.start')}</Text>,
              stop: <Text key="stop" style={[styles.heading, { color: TARGET_COLOR }]}>{route.stop.title || t('lobby.target')}</Text>,
              distance: (route.distanceMeters / 1000).toFixed(1),
            })}
          </Text>
          {!running && <Text style={[styles.text, !atStart && !skipStartPositionCheck && styles.error]}>
            {skipStartPositionCheck
              ? t('lobby.checkDisabled')
              : atStart
                ? t('lobby.atStart')
                : metersToStart !== null
                  ? t('lobby.goToStart', { meters: Math.round(metersToStart), radius: START_RADIUS_METERS })
                  : t('lobby.waitingForLocation')}
          </Text>}
        </View>}
        {state.team === 'hunter' && !running && me && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.hunterTeamHeading')}</Text>
          <Text style={[styles.text, needsHunterTeam && styles.error]}>
            {needsHunterTeam ? t('lobby.needHunterTeam') : t('lobby.huntWith', { name: me.hunterTeam ?? '' })}
          </Text>
          <Text style={styles.hint}>{t('lobby.hunterTeamHint')}</Text>
          {hunterTeams(state.players).filter((team) => team !== me.hunterTeam).map((team) => (
            <Button key={team} label={t('lobby.joinTeam', { name: team })} variant="secondary" disabled={busy || me.ready} onPress={() => changeHunterTeam(team)} />
          ))}
          <TextInput
            accessibilityLabel={t('lobby.newTeamName')}
            placeholder={t('lobby.newTeamName')}
            placeholderTextColor={colors.placeholder}
            value={newTeam}
            onChangeText={setNewTeam}
            editable={!busy && !me.ready}
            autoCorrect={false}
            maxLength={MAX_NAME_LENGTH}
            style={styles.input}
          />
          <Button label={t('lobby.createTeam')} variant="secondary" disabled={busy || me.ready || !newTeam.trim()} onPress={() => changeHunterTeam(newTeam)} />
          {me.ready && <Text style={styles.hint}>{t('lobby.unreadyForTeam')}</Text>}
        </View>}
        {(['runner', 'hunter'] as const).map((team) => (
          <View key={team} style={styles.card}>
            <Text style={styles.heading}>
              {t('lobby.teamCount', { team: t(team === 'runner' ? 'team.runners' : 'team.hunters'), count: teams[team].length })}
            </Text>
            {teams[team].length === 0 && <Text style={styles.hint}>{t('lobby.nobody')}</Text>}
            {teams[team].map((p) => (
              <Text key={p.playerId} style={styles.text}>
                {p.ready ? '✓' : '○'} {p.playerId === state.playerId ? t('common.you', { name: p.name }) : p.name}
                {team === 'hunter' ? ` · ${p.hunterTeam ? t('lobby.hunterTeam', { name: p.hunterTeam }) : t('lobby.noTeam')}` : ''}
                {running ? '' : ` · ${t(p.ready ? 'lobby.ready' : 'lobby.notReady')}`}
              </Text>
            ))}
          </View>
        ))}
        {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
        {!running && me && <View style={styles.row}>
          <Button
            label={t(me.ready ? 'lobby.notReadyButton' : 'lobby.readyButton')}
            variant={me.ready ? 'secondary' : 'primary'}
            disabled={busy || (!me.ready && needsHunterTeam) || (!skipStartPositionCheck && !me.ready && isRunner && !atStart)}
            onPress={() => void act(async () => show(await setReady(token, !me.ready)))}
          />
          <Button label={t('lobby.leave')} variant="secondary" disabled={busy} onPress={() => void act(async () => {
            await leaveGame(token);
            goHome();
          })} />
        </View>}
        {running && <Button label={t('lobby.exit')} variant="secondary" onPress={goHome} />}
      </KeyboardScrollView>
    </>
  );
}
