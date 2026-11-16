import { Ionicons } from '@expo/vector-icons';
import { Pressable, Text, View } from 'react-native';

import { useT } from '../i18n';
import { dynamicStyles, mapStyles as styles } from '../styles';

type Props = {
  locating: boolean;
  error: string;
  onLocate: () => void;
  inset?: { bottom?: number; right?: number };
};

export default function MapLocateButton({ locating, error, onLocate, inset }: Props) {
  const t = useT();
  return (
    <View style={[styles.locate, dynamicStyles.locate(inset)]}>
      {!!error && <Text style={styles.locateError}>{error}</Text>}
      <Pressable accessibilityRole="button" accessibilityLabel={t('map.myLocation')} disabled={locating} onPress={onLocate} style={[styles.locateButton, locating && styles.locating]}>
        <Ionicons name="locate" size={22} color="#10251c" />
      </Pressable>
    </View>
  );
}
