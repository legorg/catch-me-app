import { router } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, Vibration, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { updatePosition } from '@/api/client';
import Button from '@/components/button';
import PlaygroundMap, { type CellHighlight, type MapMarker } from '@/components/playground-map';
import { translateParts, useT } from '@/i18n';
import { formatDuration, jokerMarkers, playerMarkers, revealMarkers, secondsUntil } from '@/lobby';
import { cellAt } from '@/playground';
import { colors, styles } from '@/styles';
import { goHome, useGameState } from '@/use-game-state';
import { useLocationSharing, type Fix } from '@/use-location-sharing';

const START_COLOR = '#2f80ed';
const TARGET_COLOR = '#eb5757';

export default function GameScreen() {
  const t = useT();
  const insets = useSafeAreaInsets();
  const { token, snapshot, state, now, error, show, fail, running } = useGameState();
  const [infoOpen, setInfoOpen] = useState(false);

  // The game screen only opens when the countdown ends, so this buzzes once at the start.
  // (iOS ignores the pattern and vibrates once for a fixed time.)
  useEffect(() => {
    Vibration.vibrate([0, 400, 200, 400]);
  }, []);

  // A new nextRevealAt means the previous reveal just happened. The first value seen is not a reveal.
  const revealAt = state?.nextRevealAt;
  const lastRevealAt = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (lastRevealAt.current && revealAt && revealAt !== lastRevealAt.current) Vibration.vibrate(300);
    if (revealAt) lastRevealAt.current = revealAt;
  }, [revealAt]);

  // Back in the lobby if the game is not running (e.g. a countdown that was cancelled).
  useEffect(() => {
    if (state && !running) router.replace('/lobby');
  }, [state, running]);

  // Not tied to focus: the jokers sheet opens over this screen and sharing must not stop meanwhile.
  const location = useLocationSharing(!!token && !!state, true, useCallback(async (fix: Fix) => {
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
    ...revealMarkers(state),
    ...jokerMarkers(state),
    ...playerMarkers(state).filter((marker) => marker.id !== state.playerId),
  ];
  const permission = location.permission;

  return (
    <View style={styles.screen}>
      <View style={StyleSheet.absoluteFill}>
        <PlaygroundMap playground={state.playground} editable={false} onSelect={() => {}} highlights={highlights} markers={markers} fullscreen controlsInset={{ bottom: insets.bottom, right: insets.right }} />
      </View>
      {/* Floating over the map; touches outside the cards fall through to it. */}
      <View pointerEvents="box-none" style={[overlay.fill, { paddingTop: insets.top + 8, paddingBottom: insets.bottom + 12, paddingLeft: insets.left + 12, paddingRight: insets.right + 12 }]}>
        <View pointerEvents="none" style={overlay.hud}>
          {nextReveal !== null && <Text accessibilityLiveRegion="polite" style={styles.heading}>{t('lobby.nextReveal', { time: formatDuration(nextReveal) })}</Text>}
          {myCell === null && <Text style={[styles.heading, styles.error]}>{t('lobby.outside')}</Text>}
        </View>
        <View pointerEvents="box-none" style={{ flex: 1, justifyContent: 'flex-end', alignItems: 'flex-start', gap: 8 }}>
          {infoOpen && <ScrollView style={overlay.info} contentContainerStyle={{ gap: 12 }}>
            <Text style={styles.text}>
              {translateParts('lobby.youAre', {
                team: <Text key="team" style={styles.heading}>{t(isRunner ? 'team.runner' : 'team.hunter')}</Text>,
              })}
              {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
            </Text>
            {nextReveal !== null && <Text style={styles.hint}>{t(isRunner ? 'lobby.revealRunner' : 'lobby.revealHunter')}</Text>}
            {route && isRunner && <Text style={styles.text}>
              {translateParts('lobby.route', {
                start: <Text key="start" style={[styles.heading, { color: START_COLOR }]}>{startTitle}</Text>,
                stop: <Text key="stop" style={[styles.heading, { color: TARGET_COLOR }]}>{stopTitle}</Text>,
                distance: (route.distanceMeters / 1000).toFixed(1),
              })}
            </Text>}
            <Text style={styles.hint}>{t(isRunner ? 'lobby.legendRunner' : 'lobby.legendHunter')}</Text>
            {permission && !permission.granted && <View style={{ gap: 8 }}>
              <Text style={styles.heading}>{t('lobby.shareHeading')}</Text>
              <Text style={styles.text}>{t(location.background.available ? 'lobby.shareRunning' : 'lobby.shareRunningForeground')}</Text>
              {permission.canAskAgain
                ? <Button label={t('lobby.allowLocation')} onPress={() => void location.requestPermission()} />
                : <Text style={[styles.text, styles.error]}>{t('lobby.locationBlocked')}</Text>}
            </View>}
            {permission?.granted && <View style={{ gap: 8 }}>
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
          </ScrollView>}
          {!!location.error && <Text style={[styles.text, styles.error, overlay.pill]}>{location.error}</Text>}
          {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error, overlay.pill]}>{error}</Text>}
          <View style={{ flexDirection: 'row', gap: 8 }}>
            <Pressable accessibilityRole="button" onPress={goHome} style={overlay.pill}>
              <Text style={styles.secondaryButtonText}>{t('lobby.exit')}</Text>
            </Pressable>
            <Pressable accessibilityRole="button" onPress={() => router.push('/jokers')} style={overlay.pill}>
              <Text style={styles.secondaryButtonText}>{t('joker.button')}</Text>
            </Pressable>
            <Pressable accessibilityRole="button" accessibilityState={{ expanded: infoOpen }} onPress={() => setInfoOpen(!infoOpen)} style={overlay.pill}>
              <Text style={styles.secondaryButtonText}>{infoOpen ? '✕' : 'ⓘ'}</Text>
            </Pressable>
          </View>
        </View>
      </View>
    </View>
  );
}

const overlay = StyleSheet.create({
  fill: { ...StyleSheet.absoluteFill, gap: 8 },
  hud: { backgroundColor: colors.card, borderRadius: 12, padding: 12, gap: 2, opacity: 0.95 },
  info: { backgroundColor: colors.card, borderRadius: 12, padding: 16, maxHeight: '50%', alignSelf: 'stretch', flexGrow: 0 },
  pill: { backgroundColor: colors.card, borderRadius: 12, paddingHorizontal: 16, paddingVertical: 8, minHeight: 40, justifyContent: 'center', opacity: 0.95 },
});
