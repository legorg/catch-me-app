import { Text, View } from 'react-native';

import { type GameRoute } from '@/api/types';
import { translateParts, useT } from '@/i18n';
import { routeTitles, startCheckMessage } from '@/lobby';
import { styles } from '@/styles';

/** The runner's route and whether they stand at the start. */
export default function LobbyRouteCard({ route, skipStartPositionCheck, atStart, metersToStart }: {
  route: GameRoute;
  skipStartPositionCheck: boolean;
  atStart: boolean;
  metersToStart: number | null;
}) {
  const t = useT();
  const titles = routeTitles(route);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('lobby.routeHeading')}</Text>
      <Text style={styles.text}>
        {translateParts('lobby.route', {
          start: <Text key="start" style={[styles.heading, styles.startText]}>{titles.start}</Text>,
          stop: <Text key="stop" style={[styles.heading, styles.targetText]}>{titles.stop}</Text>,
          distance: (route.distanceMeters / 1000).toFixed(1),
        })}
      </Text>
      <Text style={[styles.text, !atStart && !skipStartPositionCheck && styles.error]}>
        {startCheckMessage(skipStartPositionCheck, atStart, metersToStart)}
      </Text>
    </View>
  );
}
