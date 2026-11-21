import { Text, View } from 'react-native';

import type { GameResultResponse } from '@/api/types';
import Button from '@/components/button';
import FullscreenMap from '@/components/fullscreen-map';
import { useT } from '@/i18n';
import { formatDuration, noop } from '@/lobby';
import { formatReplayTime, REPLAY_SPEEDS } from '@/replay';
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

/** Shared controls; the replay clock stays in GameReplay when the fullscreen modal opens/closes. */
function ReplayControls({ replay, startsAt }: { replay: ReturnType<typeof useReplay>; startsAt: string }) {
  const t = useT();
  const { at, duration, running, speed, setSpeed, toggle, seek, legend } = replay;
  return (
    <View style={styles.gapSmall}>
      <View style={replayStyles.legend}>
        {legend.map((entry) => (
          <View key={entry.key} style={replayStyles.legendItem}>
            <View style={[replayStyles.legendDot, dynamicStyles.pinHead(entry.color)]} />
            <Text style={styles.text}>{entry.team === 'runner' ? t('team.runners') : entry.hunterTeam ?? t('result.noHunterTeam')}</Text>
          </View>
        ))}
      </View>
      <Scrubber value={at} max={duration} onSeek={seek} />
      <Text style={[styles.text, styles.boldText]}>{t('replay.clockTime', { time: formatReplayTime(startsAt, at) })}</Text>
      <Text style={styles.text}>{t('replay.time', { at: formatDuration(Math.floor(at / 1000)), total: formatDuration(Math.floor(duration / 1000)) })}</Text>
      <View style={styles.row}>
        <Button label={t(running ? 'replay.pause' : 'replay.play')} onPress={toggle} />
        {REPLAY_SPEEDS.map((option) => (
          <Button key={option} label={t('replay.speed', { speed: option })} variant={option === speed ? 'primary' : 'secondary'} onPress={() => setSpeed(option)} />
        ))}
      </View>
    </View>
  );
}

/** Plays the ended game back inline or fullscreen, using the same clock and controls. */
export default function GameReplay({ result }: { result: GameResultResponse }) {
  const t = useT();
  const replay = useReplay(result);
  const { markers, lines } = replay;
  const controls = <ReplayControls replay={replay} startsAt={result.startsAt} />;
  return (
    <View style={styles.card}>
      <Text accessibilityRole="header" style={styles.heading}>{t('replay.title')}</Text>
      <View style={styles.mapContainer}>
        <FullscreenMap playground={result.playground} editable={false} onSelect={noop} markers={markers} lines={lines} locationEnabled={false} rotatable fullscreenFooter={controls} />
      </View>
      {controls}
      <Text style={styles.hint}>{t('replay.legend')}</Text>
    </View>
  );
}
