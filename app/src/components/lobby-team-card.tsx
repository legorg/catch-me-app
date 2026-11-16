import { Text, View } from 'react-native';

import { type PlayerResponse } from '@/api/types';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** One team's roster with each player's readiness. */
export default function LobbyTeamCard({ team, players, playerId }: {
  team: 'runner' | 'hunter';
  players: PlayerResponse[];
  playerId: string;
}) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>
        {t('lobby.teamCount', { team: t(team === 'runner' ? 'team.runners' : 'team.hunters'), count: players.length })}
      </Text>
      {players.length === 0 && <Text style={styles.hint}>{t('lobby.nobody')}</Text>}
      {players.map((p) => (
        <Text key={p.playerId} style={styles.text}>
          {p.ready ? '✓' : '○'} {p.playerId === playerId ? t('common.you', { name: p.name }) : p.name}
          {team === 'hunter' ? ` · ${p.hunterTeam ? t('lobby.hunterTeam', { name: p.hunterTeam }) : t('lobby.noTeam')}` : ''}
          {` · ${t(p.ready ? 'lobby.ready' : 'lobby.notReady')}`}
        </Text>
      ))}
    </View>
  );
}
