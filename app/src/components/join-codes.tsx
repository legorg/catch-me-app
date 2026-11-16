import { Text, View } from 'react-native';

import type { JoinCodes as Codes } from '@/api/client';
import { useT } from '@/i18n';
import { styles } from '@/styles';

/** Card with the runner and hunter join codes, for inviting players to either team. */
export default function JoinCodes({ codes }: { codes: Codes }) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('joinCodes.heading')}</Text>
      <View style={styles.row}>
        <View style={styles.flex}>
          <Text style={styles.text}>{t('team.runners')}</Text>
          <Text selectable style={styles.code}>{codes.runner}</Text>
        </View>
        <View style={styles.flex}>
          <Text style={styles.text}>{t('team.hunters')}</Text>
          <Text selectable style={styles.code}>{codes.hunter}</Text>
        </View>
      </View>
      <Text style={styles.hint}>{t('joinCodes.hint')}</Text>
    </View>
  );
}
