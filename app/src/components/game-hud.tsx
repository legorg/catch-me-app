import { Text, View } from 'react-native';

import type { RunnerStatus } from '@/api/types';
import { useT } from '@/i18n';
import { formatDuration, SPECTATOR_COLORS } from '@/lobby';
import { overlay, styles } from '@/styles';

type GameHudProps = {
  /** Seconds until the next reveal, if one is scheduled. */
  nextReveal: number | null;
  outside: boolean;
  drawing: boolean;
  runnerStatus?: RunnerStatus;
  spectator: boolean;
  testMode: boolean;
};

/** Top card: test override, runner outcome, next reveal, playground warning and drawing hint. */
export default function GameHud({ nextReveal, outside, drawing, runnerStatus, spectator, testMode }: GameHudProps) {
  const t = useT();
  return (
    <View pointerEvents="none" style={overlay.hud}>
      {testMode && <>
        <Text style={[styles.heading, styles.error]}>{t('lobby.testMode')}</Text>
        <Text style={styles.hint}>{t('lobby.testModeText')}</Text>
      </>}
      {spectator && <>
        <Text accessibilityLiveRegion="polite" style={styles.heading}>{t('spectator.title')}</Text>
        {(runnerStatus === 'caught' || runnerStatus === 'finished') && <Text style={styles.text}>{t(runnerStatus === 'caught' ? 'catch.status' : 'finish.status')}</Text>}
        <View style={styles.row}>
          <Text style={[styles.text, { color: SPECTATOR_COLORS.runner }]}>{t('spectator.runnerLegend')}</Text>
          <Text style={[styles.text, { color: SPECTATOR_COLORS.hunter }]}>{t('spectator.hunterLegend')}</Text>
        </View>
        <Text style={styles.hint}>{t('spectator.outcomeLegend')}</Text>
      </>}
      {!spectator && (runnerStatus === 'caught' || runnerStatus === 'finished') && <>
        <Text accessibilityLiveRegion="polite" style={styles.heading}>{t(runnerStatus === 'caught' ? 'catch.status' : 'finish.status')}</Text>
        <Text style={styles.hint}>{t(runnerStatus === 'caught' ? 'catch.spectating' : 'finish.spectating')}</Text>
      </>}
      {nextReveal !== null && <Text accessibilityLiveRegion="polite" style={styles.heading}>{t('lobby.nextReveal', { time: formatDuration(nextReveal) })}</Text>}
      {outside && <Text style={[styles.heading, styles.error]}>{t('lobby.outside')}</Text>}
      {drawing && <Text style={styles.hint}>{t('sketch.hint')}</Text>}
    </View>
  );
}
