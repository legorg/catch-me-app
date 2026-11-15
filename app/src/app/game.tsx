import { router } from 'expo-router';
import { useCallback, useEffect } from 'react';
import { ActivityIndicator, ScrollView, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { updatePosition } from '@/api/client';
import Button from '@/components/button';
import PlaygroundMap, { type CellHighlight, type MapMarker } from '@/components/playground-map';
import { translateParts, useT } from '@/i18n';
import { formatDuration, playerMarkers, secondsUntil } from '@/lobby';
import { cellAt } from '@/playground';
import { colors, styles } from '@/styles';
import { goHome, useGameState } from '@/use-game-state';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

const START_COLOR = '#2f80ed';
const TARGET_COLOR = '#eb5757';

export default function GameScreen() {
  const t = useT();
  const insets = useSafeAreaInsets();
  const { token, snapshot, state, now, error, focused, show, fail, running } = useGameState();

  // Back in the lobby if the game is not running (e.g. a countdown that was cancelled).
  useEffect(() => {
    if (state && !running) router.replace('/lobby');
  }, [state, running]);

  const location = useLocationSharing(focused && !!token && !!state, true, useCallback(async (fix: Fix) => {
    if (!token) return;
    try {
      show(await updatePosition(token, fix));
    } catch (err) {
      fail(err);
    }
  }, [token, show, fail]));

  if (!token || !state || !snapshot) {
    return (
      <View style={[styles.screen, { justifyContent: 'center', padding: 20 }]}>
        {error ? <Text style={[styles.text, styles.error]}>{error}</Text> : <ActivityIndicator color={colors.accent} />}
      </View>
    );
  }

  const me = state.players.find((p) => p.playerId === state.playerId);
  const route = state.route;
  const isRunner = state.team === 'runner';
  const myCell = me?.position ? cellAt(state.playground, me.position) : undefined;
  const nextReveal = secondsUntil(state.nextRevealAt, state, snapshot.receivedAt, now);
  const startCell = route ? cellAt(state.playground, route.start.coordinates) : null;
  const stopCell = route ? cellAt(state.playground, route.stop.coordinates) : null;
  const startTitle = route?.start.title || t('lobby.start');
  const stopTitle = route?.stop.title || t('lobby.target');
  const highlights: CellHighlight[] = route ? [
    ...(startCell ? [{ cell: startCell, title: startTitle, color: START_COLOR }] : []),
    ...(stopCell ? [{ cell: stopCell, title: stopTitle, color: TARGET_COLOR }] : []),
  ] : [];
  const markers: MapMarker[] = [
    ...(route ? [
      { id: 'route-start', coordinate: route.start.coordinates, title: startTitle, description: t('lobby.start'), color: START_COLOR },
      { id: 'route-stop', coordinate: route.stop.coordinates, title: stopTitle, description: t('lobby.target'), color: TARGET_COLOR },
    ] : []),
    ...playerMarkers(state).filter((marker) => marker.id !== state.playerId),
  ];
  const permission = location.permission;

  return (
    <View style={styles.screen}>
      <View style={[styles.card, { borderRadius: 0, gap: 4 }]}>
        <Text style={styles.text}>
          {translateParts('lobby.youAre', {
            team: <Text key="team" style={styles.heading}>{t(isRunner ? 'team.runner' : 'team.hunter')}</Text>,
          })}
          {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
        </Text>
        {route && isRunner && <Text style={styles.text}>
          {translateParts('lobby.route', {
            start: <Text key="start" style={[styles.heading, { color: START_COLOR }]}>{startTitle}</Text>,
            stop: <Text key="stop" style={[styles.heading, { color: TARGET_COLOR }]}>{stopTitle}</Text>,
            distance: (route.distanceMeters / 1000).toFixed(1),
          })}
        </Text>}
        {nextReveal !== null && <>
          <Text accessibilityLiveRegion="polite" style={styles.heading}>{t('lobby.nextReveal', { time: formatDuration(nextReveal) })}</Text>
          <Text style={styles.hint}>{t(isRunner ? 'lobby.revealRunner' : 'lobby.revealHunter')}</Text>
        </>}
        {myCell === null && <Text style={[styles.heading, styles.error]}>{t('lobby.outside')}</Text>}
      </View>
      <View style={{ flex: 1 }}>
        <PlaygroundMap playground={state.playground} editable={false} onSelect={() => {}} highlights={highlights} markers={markers} fullscreen />
      </View>
      <ScrollView style={{ flexGrow: 0, maxHeight: '40%' }} contentContainerStyle={[styles.content, { paddingBottom: insets.bottom + 16 }]}>
        <Text style={styles.hint}>{t(isRunner ? 'lobby.legendRunner' : 'lobby.legendHunter')}</Text>
        {permission && !permission.granted && <View style={styles.card}>
          <Text style={styles.heading}>{t('lobby.shareHeading')}</Text>
          <Text style={styles.text}>{t(location.background.available ? 'lobby.shareRunning' : 'lobby.shareRunningForeground')}</Text>
          {permission.canAskAgain
            ? <Button label={t('lobby.allowLocation')} onPress={() => void location.requestPermission()} />
            : <Text style={[styles.text, styles.error]}>{t('lobby.locationBlocked')}</Text>}
        </View>}
        {permission?.granted && <View style={styles.card}>
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
        {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
        <Button label={t('lobby.exit')} variant="secondary" onPress={goHome} />
      </ScrollView>
    </View>
  );
}
