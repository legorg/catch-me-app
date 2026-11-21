import { Ionicons } from '@expo/vector-icons';
import { Pressable } from 'react-native';

import { useT } from '../i18n';
import { colors, mapStyles as styles } from '../styles';

/** Shown on a map whose grid is turned: a tap turns the map so that row 1 is on top and the grid looks straight. */
export default function MapAlignButton({ onPress }: { onPress: () => void }) {
  const t = useT();
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={t('map.alignGrid')} onPress={onPress} style={styles.locateButton}>
      <Ionicons name="grid-outline" size={20} color={colors.background} />
    </Pressable>
  );
}
