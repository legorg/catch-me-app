import { Text, TextInput, View } from 'react-native';

import { MAX_NAME_LENGTH } from '@/api/client';
import { type PlayerResponse } from '@/api/types';
import Button from '@/components/button';
import { useT } from '@/i18n';
import { hunterTeams } from '@/lobby';
import { colors, styles } from '@/styles';

/** Lets a hunter join an existing hunter team or create a new one (until they are ready). */
export default function HunterTeamCard({ me, players, needsHunterTeam, busy, newTeam, onNewTeamChange, onJoin, onCreate }: {
  me: PlayerResponse;
  players: PlayerResponse[];
  needsHunterTeam: boolean;
  busy: boolean;
  newTeam: string;
  onNewTeamChange: (name: string) => void;
  onJoin: (name: string) => void;
  onCreate: () => void;
}) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('lobby.hunterTeamHeading')}</Text>
      <Text style={[styles.text, needsHunterTeam && styles.error]}>
        {needsHunterTeam ? t('lobby.needHunterTeam') : t('lobby.huntWith', { name: me.hunterTeam ?? '' })}
      </Text>
      <Text style={styles.hint}>{t('lobby.hunterTeamHint')}</Text>
      {hunterTeams(players).filter((team) => team !== me.hunterTeam).map((team) => (
        <Button key={team} label={t('lobby.joinTeam', { name: team })} variant="secondary" disabled={busy || me.ready}
          disabledReason={me.ready ? t('lobby.unreadyForTeam') : null} onPress={() => onJoin(team)} />
      ))}
      <TextInput
        accessibilityLabel={t('lobby.newTeamName')}
        placeholder={t('lobby.newTeamName')}
        placeholderTextColor={colors.placeholder}
        value={newTeam}
        onChangeText={onNewTeamChange}
        editable={!busy && !me.ready}
        autoCorrect={false}
        maxLength={MAX_NAME_LENGTH}
        style={styles.input}
      />
      <Button label={t('lobby.createTeam')} variant="secondary" disabled={busy || me.ready || !newTeam.trim()}
        disabledReason={me.ready ? t('lobby.unreadyForTeam') : !newTeam.trim() ? t('lobby.enterTeamName') : null} onPress={onCreate} />
      {me.ready && <Text style={styles.hint}>{t('lobby.unreadyForTeam')}</Text>}
    </View>
  );
}
