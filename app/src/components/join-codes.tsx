import { Text, View } from 'react-native';

import type { JoinCodes as Codes } from '@/api/client';
import Button from '@/components/button';
import { useT } from '@/i18n';
import { TEAM_LABELS } from '@/lobby';
import { styles } from '@/styles';
import { useShareJoinCode } from '@/use-share-join-code';

const TEAMS = ['runner', 'hunter', 'spectator'] as const;

/** All three invitations; sharing one never includes the other invitation codes. */
export default function JoinCodes({ codes }: { codes: Codes }) {
  const t = useT();
  const { busy, error, share } = useShareJoinCode(codes);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('joinCodes.heading')}</Text>
      {TEAMS.map((team) => (
        <View key={team} style={styles.gapTiny}>
          <Text style={styles.text}>{t(TEAM_LABELS[team])}</Text>
          <View style={styles.row}>
            <Text selectable accessibilityLabel={`${t(TEAM_LABELS[team])}: ${codes[team]}`} style={styles.code}>{codes[team]}</Text>
            <Button label={t('joinCodes.share', { team: t(TEAM_LABELS[team]) })} variant="secondary" disabled={busy} onPress={() => void share(team)} />
          </View>
        </View>
      ))}
      <Text style={styles.hint}>{t('joinCodes.hint')}</Text>
      {!!error && <Text accessibilityLiveRegion="polite" style={styles.error}>{error}</Text>}
    </View>
  );
}
