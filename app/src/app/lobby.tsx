import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { ActivityIndicator, Modal, Text, TextInput, View } from 'react-native';

import {
  MAX_NAME_LENGTH, leaveGame, setHunterTeam, setReady, updatePosition,
} from '@/api/client';
import Button from '@/components/button';
import DebugAddPlayer from '@/components/debug-add-player';
import JoinCodes from '@/components/join-codes';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import FullscreenMap from '@/components/fullscreen-map';
import { type CellHighlight, type MapMarker } from '@/components/playground-map';
import { translateParts, useT } from '@/i18n';
import { isGameStateRunnerResponse } from '@/api/types';
import { hunterTeams, playerMarkers, playersByTeam } from '@/lobby';
import { cellAt, distanceMeters, START_RADIUS_METERS } from '@/playground';
import { colors, styles } from '@/styles';
import { goHome, useGameState } from '@/use-game-state';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

const START_COLOR = colors.start;
const TARGET_COLOR = colors.target;

export default function LobbyScreen() {
  const t = useT();
  const { token, snapshot, state, error, focused, show, fail, countdown, running } = useGameState();
  const [busy, setBusy] = useState(false);
  const [newTeam, setNewTeam] = useState('');

  // The game has its own screen; joining it replaces the lobby.
  useEffect(() => {
    if (running) router.replace('/game');
  }, [running]);

  const skipStartPositionCheck = state?.startPositionCheckDisabled === true;
  // Everyone shares from joining: runners for the start cell check, hunters for their hunter team.
  const sharing = focused && !!token && !!state;
  // The lobby shares in the foreground only; background sharing belongs to the game screen.
  const location = useLocationSharing(sharing, false, useCallback(async (fix: Fix) => {
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
  const route = isGameStateRunnerResponse(state) ? state.route : undefined;
  const isRunner = isGameStateRunnerResponse(state);
  // The server only sends the route to runners.
  const metersToStart = route && me?.position ? distanceMeters(me.position, route.start.coordinates) : null;
  const atStart = metersToStart !== null && metersToStart <= START_RADIUS_METERS;
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
  const background = location.background;
  // Installed builds need "Always"; Expo Go cannot share in the background, so foreground access must do there.
  // ("Always" can only be granted on top of foreground access, so it covers both.)
  const needsBackground = background.available && background.permission?.granted !== true;
  const needsForeground = !background.available && permission?.granted !== true;
  const needsHunterTeam = state.team === 'hunter' && !me?.hunterTeam;
  const changeHunterTeam = (name: string) => void act(async () => {
    show(await setHunterTeam(token, name));
    setNewTeam('');
  });

  return (
    <>
      {/* Covers everything until the game screen replaces the lobby; un-readying or leaving closes it. */}
      <Modal visible={countdown !== null && countdown > 0} animationType="fade" statusBarTranslucent onRequestClose={() => {}}>
        <View style={[styles.screen, styles.countdownScreen]}>
          <Text style={styles.heading}>{t('lobby.countdownTitle')}</Text>
          <Text accessibilityLiveRegion="polite" style={styles.countdown}>{countdown}</Text>
          <Text style={[styles.hint, styles.centerText]}>{t('lobby.countdownHint')}</Text>
        </View>
      </Modal>
      <KeyboardScrollView>
        {skipStartPositionCheck && <View style={styles.card}>
          <Text style={[styles.heading, styles.error]}>{t('lobby.testMode')}</Text>
          <Text style={styles.text}>{t('lobby.testModeText')}</Text>
        </View>}
        <Text accessibilityLiveRegion="polite" style={styles.text}>
          {countdown !== null
            ? t(skipStartPositionCheck ? 'lobby.countdownInfoTest' : 'lobby.countdownInfo')
            : t(skipStartPositionCheck ? 'lobby.startInfoTest' : 'lobby.startInfo')}
        </Text>
        <View style={styles.mapContainer}>
          <FullscreenMap playground={state.playground} editable={false} onSelect={() => {}} highlights={highlights} markers={[...routeMarkers, ...playerMarkers(state).filter((marker) => marker.id !== state.playerId)]} />
        </View>
        {state.team === 'hunter' && <Text style={styles.hint}>
          {t('lobby.legendHunter')}
        </Text>}
        {sharing && permission && !permission.granted && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.shareHeading')}</Text>
          <Text style={styles.text}>
            {t(state.team === 'runner' && !skipStartPositionCheck ? 'lobby.shareRunnerStart' : 'lobby.shareHunterLobby')}
          </Text>
          {needsForeground && <Text style={[styles.text, !me?.ready && styles.error]}>{t('lobby.locationRequired')}</Text>}
          {permission.canAskAgain
            ? <Button label={t('lobby.allowLocation')} onPress={() => void location.requestPermission()} />
            : <Text style={[styles.text, styles.error]}>{t('lobby.locationBlocked')}</Text>}
        </View>}
        {!!location.error && <Text style={[styles.text, styles.error]}>{location.error}</Text>}
        {sharing && needsBackground && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.backgroundHeading')}</Text>
          <Text style={[styles.text, !me?.ready && styles.error]}>{t('lobby.backgroundRequired')}</Text>
          {/* "Always" can only be asked for once foreground access is granted (see the card above). */}
          {permission?.granted && (background.permission?.canAskAgain !== false
            ? <Button label={t('lobby.shareInBackground')} variant="secondary" onPress={() => void background.request()} />
            : <Text style={[styles.text, styles.error]}>{t('lobby.backgroundRequiredBlocked')}</Text>)}
        </View>}
        <View style={styles.card}>
          <Text style={styles.text}>
            {translateParts('lobby.youAre', {
              team: <Text key="team" style={styles.heading}>{t(state.team === 'runner' ? 'team.runner' : 'team.hunter')}</Text>,
            })}
            {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
          </Text>
        </View>
        <JoinCodes codes={state.joinCodes} />
        {__DEV__ && <DebugAddPlayer codes={state.joinCodes} />}
        {route && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.routeHeading')}</Text>
          <Text style={styles.text}>
            {translateParts('lobby.route', {
              start: <Text key="start" style={[styles.heading, styles.startText]}>{route.start.title || t('lobby.start')}</Text>,
              stop: <Text key="stop" style={[styles.heading, styles.targetText]}>{route.stop.title || t('lobby.target')}</Text>,
              distance: (route.distanceMeters / 1000).toFixed(1),
            })}
          </Text>
          <Text style={[styles.text, !atStart && !skipStartPositionCheck && styles.error]}>
            {skipStartPositionCheck
              ? t('lobby.checkDisabled')
              : atStart
                ? t('lobby.atStart')
                : metersToStart !== null
                  ? t('lobby.goToStart', { meters: Math.round(metersToStart), radius: START_RADIUS_METERS })
                  : t('lobby.waitingForLocation')}
          </Text>
        </View>}
        {state.team === 'hunter' && me && <View style={styles.card}>
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
                {` · ${t(p.ready ? 'lobby.ready' : 'lobby.notReady')}`}
              </Text>
            ))}
          </View>
        ))}
        {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
        {me && <View style={styles.row}>
          <Button
            label={t(me.ready ? 'lobby.notReadyButton' : 'lobby.readyButton')}
            variant={me.ready ? 'secondary' : 'primary'}
            disabled={busy || (!me.ready && (needsHunterTeam || needsBackground || needsForeground)) || (!skipStartPositionCheck && !me.ready && isRunner && !atStart)}
            onPress={() => void act(async () => show(await setReady(token, !me.ready)))}
          />
          <Button label={t('lobby.leave')} variant="secondary" disabled={busy} onPress={() => void act(async () => {
            await leaveGame(token);
            goHome();
          })} />
        </View>}
      </KeyboardScrollView>
    </>
  );
}
