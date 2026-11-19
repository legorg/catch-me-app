import { Text, View } from 'react-native';

import type { GameResultResponse } from '@/api/types';
import Button from '@/components/button';
import PlaygroundMap from '@/components/playground-map';
import { useT } from '@/i18n';
import { formatDuration, noop } from '@/lobby';
import { REPLAY_SPEEDS } from '@/replay';
import { dynamicStyles, replayStyles, styles } from '@/styles';
import { useReplay } from '@/use-replay';
import { useScrubber } from '@/use-scrubber';

function Scrubber({ value, max, onSeek }: { value: number; max: number; onSeek: (value: number) => void }) {
  const t = useT();
  const handlers = useScrubber(value, max, onSeek);
  const share = max > 0 ? value / max : 0;
  return (
    <View
      style={replayStyles.scrubber}
      accessible
      accessibilityRole="adjustable"
      accessibilityLabel={t('replay.scrubber')}
      accessibilityValue={{ min: 0, max: 100, now: Math.round(share * 100) }}
      {...handlers}
    >
      <View pointerEvents="none" style={replayStyles.rail}>
        <View style={[replayStyles.progress, dynamicStyles.replayProgress(share)]} />
      </View>
    </View>
  );
}

/** Plays the ended game back on the map: every runner and hunter moving with a short tail. */
export default function GameReplay({ result }: { result: GameResultResponse }) {
  const t = useT();
  const { at, duration, running, speed, setSpeed, toggle, seek, markers, lines, highlights, legend } = useReplay(result);
  return (
    <View style={styles.card}>
      <Text accessibilityRole="header" style={styles.heading}>{t('replay.title')}</Text>
      <View style={styles.mapContainer}>
        <PlaygroundMap playground={result.playground} editable={false} onSelect={noop} highlights={highlights} markers={markers} lines={lines} locationEnabled={false} />
      </View>
      <View style={replayStyles.legend}>
        {legend.map((entry) => (
          <View key={entry.key} style={replayStyles.legendItem}>
            <View style={[replayStyles.legendDot, dynamicStyles.pinHead(entry.color)]} />
            <Text style={styles.text}>{entry.team === 'runner' ? t('team.runners') : entry.hunterTeam ?? t('result.noHunterTeam')}</Text>
          </View>
        ))}
      </View>
      <Scrubber value={at} max={duration} onSeek={seek} />
      <Text style={styles.text}>{t('replay.time', { at: formatDuration(Math.floor(at / 1000)), total: formatDuration(Math.floor(duration / 1000)) })}</Text>
      <Button label={t(running ? 'replay.pause' : 'replay.play')} onPress={toggle} />
      <View style={styles.row}>
        {REPLAY_SPEEDS.map((option) => (
          <Button key={option} label={t('replay.speed', { speed: option })} variant={option === speed ? 'primary' : 'secondary'} onPress={() => setSpeed(option)} />
        ))}
      </View>
      <Text style={styles.hint}>{t('replay.legend')}</Text>
    </View>
  );
}
