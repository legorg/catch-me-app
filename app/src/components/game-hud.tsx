import { Text, View } from 'react-native';

import type { RunnerStatus } from '@/api/types';
import { useT } from '@/i18n';
import { formatDuration } from '@/lobby';
import { overlay, styles } from '@/styles';

type GameHudProps = {
  /** Seconds until the next reveal, if one is scheduled. */
  nextReveal: number | null;
  outside: boolean;
  drawing: boolean;
  runnerStatus?: RunnerStatus;
  testMode: boolean;
};

/** Top card: test override, runner outcome, next reveal, playground warning and drawing hint. */
export default function GameHud({ nextReveal, outside, drawing, runnerStatus, testMode }: GameHudProps) {
  const t = useT();
  return (
    <View pointerEvents="none" style={overlay.hud}>
      {testMode && <>
        <Text style={[styles.heading, styles.error]}>{t('lobby.testMode')}</Text>
        <Text style={styles.hint}>{t('lobby.testModeText')}</Text>
      </>}
      {(runnerStatus === 'caught' || runnerStatus === 'finished') && <>
        <Text accessibilityLiveRegion="polite" style={styles.heading}>{t(runnerStatus === 'caught' ? 'catch.status' : 'finish.status')}</Text>
        <Text style={styles.hint}>{t(runnerStatus === 'caught' ? 'catch.spectating' : 'finish.spectating')}</Text>
      </>}
      {nextReveal !== null && <Text accessibilityLiveRegion="polite" style={styles.heading}>{t('lobby.nextReveal', { time: formatDuration(nextReveal) })}</Text>}
      {outside && <Text style={[styles.heading, styles.error]}>{t('lobby.outside')}</Text>}
      {drawing && <Text style={styles.hint}>{t('sketch.hint')}</Text>}
    </View>
  );
}
