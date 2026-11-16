import { Text, View } from 'react-native';

import { useT } from '@/i18n';
import { formatDuration } from '@/lobby';
import { overlay, styles } from '@/styles';

type GameHudProps = {
  /** Seconds until the next reveal, if one is scheduled. */
  nextReveal: number | null;
  outside: boolean;
  drawing: boolean;
};

/** Top card: next reveal, a warning when the player left the playground, and the drawing hint. */
export default function GameHud({ nextReveal, outside, drawing }: GameHudProps) {
  const t = useT();
  return (
    <View pointerEvents="none" style={overlay.hud}>
      {nextReveal !== null && <Text accessibilityLiveRegion="polite" style={styles.heading}>{t('lobby.nextReveal', { time: formatDuration(nextReveal) })}</Text>}
      {outside && <Text style={[styles.heading, styles.error]}>{t('lobby.outside')}</Text>}
      {drawing && <Text style={styles.hint}>{t('sketch.hint')}</Text>}
    </View>
  );
}
