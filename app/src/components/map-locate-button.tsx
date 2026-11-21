import { Ionicons } from '@expo/vector-icons';
import { Pressable, Text } from 'react-native';

import { useT } from '../i18n';
import { colors, mapStyles as styles } from '../styles';

type Props = {
  locating: boolean;
  error: string;
  onLocate: () => void;
};

/** The "my location" button with its error; placed in the map's control column. */
export default function MapLocateButton({ locating, error, onLocate }: Props) {
  const t = useT();
  return (
    <>
      {!!error && <Text style={styles.locateError}>{error}</Text>}
      <Pressable accessibilityRole="button" accessibilityLabel={t('map.myLocation')} disabled={locating} onPress={onLocate} style={[styles.locateButton, locating && styles.locating]}>
        <Ionicons name="locate" size={22} color={colors.background} />
      </Pressable>
    </>
  );
}
