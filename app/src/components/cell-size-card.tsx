import { Text, TextInput, View } from 'react-native';

import { useT } from '@/i18n';
import { styles } from '@/styles';

/** Card for entering the grid cell size in kilometers, with its validation error. */
export default function CellSizeCard({ value, onChange, editable, error }: { value: string; onChange: (text: string) => void; editable: boolean; error: string }) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('create.cellSize')}</Text>
      <TextInput
        accessibilityLabel={t('create.cellSizeLabel')}
        value={value}
        onChangeText={onChange}
        editable={editable}
        keyboardType="decimal-pad"
        autoCorrect={false}
        maxLength={8}
        style={styles.input}
      />
      <Text style={styles.hint}>{t('create.cellSizeHint')}</Text>
      {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
    </View>
  );
}
