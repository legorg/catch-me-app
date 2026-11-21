import { Text, View } from 'react-native';

import { type GameStateRunnerResponse } from '@/api/types';
import Button from '@/components/button';
import { translateParts, useT } from '@/i18n';
import { routeChangeBlocker, routeTitles, startCheckMessage } from '@/lobby';
import { styles } from '@/styles';

/** The runners' shared route, whether the player stands at its start, and a way to get a new one. */
export default function LobbyRouteCard({ state, skipStartPositionCheck, atStart, metersToStart, busy, changingRoute, onRegenerate }: {
  state: GameStateRunnerResponse;
  skipStartPositionCheck: boolean;
  atStart: boolean;
  metersToStart: number | null;
  busy: boolean;
  changingRoute: boolean;
  onRegenerate: () => void;
}) {
  const t = useT();
  const { route } = state;
  const titles = routeTitles(route);
  const blocker = routeChangeBlocker(state);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('lobby.routeHeading')}</Text>
      <Text style={styles.text}>
        {translateParts('lobby.route', {
          start: <Text key="start" style={[styles.heading, styles.routeText]}>{titles.start}</Text>,
          stop: <Text key="stop" style={[styles.heading, styles.routeText]}>{titles.stop}</Text>,
          distance: (route.distanceMeters / 1000).toFixed(1),
        })}
      </Text>
      <Text style={[styles.text, !atStart && !skipStartPositionCheck && styles.error]}>
        {startCheckMessage(skipStartPositionCheck, atStart, metersToStart)}
      </Text>
      <Button
        label={t('lobby.newRoute', { count: state.routeRegenerationsLeft })}
        variant="secondary"
        disabled={busy || blocker !== null}
        onPress={onRegenerate}
      />
      <Text accessibilityLiveRegion="polite" style={styles.hint}>
        {changingRoute ? t('lobby.findingRoute')
          : blocker === 'noneLeft' ? t('lobby.noRouteChangesLeft')
            : blocker === 'runnerReady' ? t('lobby.unreadyForRoute')
              : t('lobby.newRouteHint')}
      </Text>
    </View>
  );
}
