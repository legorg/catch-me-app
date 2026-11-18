import { Text, View } from 'react-native';

import type { GameResultResponse, ResultPlayerResponse } from '@/api/types';
import TimelineEntries from '@/components/timeline-entries';
import { formatResultTime } from '@/game-result';
import { useT } from '@/i18n';
import { formatDuration } from '@/lobby';
import { styles } from '@/styles';

function PlayerResult({ player }: { player: ResultPlayerResponse }) {
  const t = useT();
  return (
    <View style={styles.gapTiny}>
      <Text style={[styles.text, styles.boldText]}>{player.name}</Text>
      {player.runner
        ? <Text style={styles.text}>{t(player.runner.status === 'caught' ? 'result.caughtAt' : 'result.finishedAt', { time: formatResultTime(player.runner.endedAt) })}</Text>
        : <Text style={styles.text}>{player.team === 'spectator' ? t('result.spectator') : player.hunterTeam ? t('result.hunterTeam', { name: player.hunterTeam }) : t('result.noHunterTeam')}</Text>}
    </View>
  );
}

export default function GameResultSummary({ result }: { result: GameResultResponse }) {
  const t = useT();
  return (
    <>
      <View style={styles.card}>
        <Text accessibilityRole="header" style={styles.heading}>{t('result.ended')}</Text>
        <Text style={styles.text}>{t('result.startedAt', { time: formatResultTime(result.startsAt) })}</Text>
        <Text style={styles.text}>{t('result.endedAt', { time: formatResultTime(result.endedAt) })}</Text>
        <Text style={styles.text}>{t('result.duration', { time: formatDuration(result.durationSeconds) })}</Text>
      </View>
      <View style={styles.card}>
        <Text accessibilityRole="header" style={styles.heading}>{t('team.runners')}</Text>
        {result.players.filter((p) => p.team === 'runner').map((p) => <PlayerResult key={p.playerId} player={p} />)}
      </View>
      <View style={styles.card}>
        <Text accessibilityRole="header" style={styles.heading}>{t('team.hunters')}</Text>
        {result.players.filter((p) => p.team === 'hunter').map((p) => <PlayerResult key={p.playerId} player={p} />)}
      </View>
      {result.players.some((player) => player.team === 'spectator') && <View style={styles.card}>
        <Text accessibilityRole="header" style={styles.heading}>{t('team.spectators')}</Text>
        {result.players.filter((p) => p.team === 'spectator').map((p) => <PlayerResult key={p.playerId} player={p} />)}
      </View>}
      {/* Oldest first: a finished game reads from start to end. */}
      <View style={styles.card}>
        <Text accessibilityRole="header" style={styles.heading}>{t('timeline.title')}</Text>
        <TimelineEntries events={result.events} />
      </View>
      <Text style={styles.hint}>{t('result.summaryOnly')}</Text>
    </>
  );
}
