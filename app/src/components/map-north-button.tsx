import { Ionicons } from '@expo/vector-icons';
import { Pressable } from 'react-native';

import { useT } from '../i18n';
import { colors, dynamicStyles, mapStyles as styles } from '../styles';

type Props = {
  /** Where the top of the map points, in degrees clockwise from north. */
  heading: number;
  onPress: () => void;
};

/** Shown on a turned map: the arrow points north, a tap turns the map back to north-up. */
export default function MapNorthButton({ heading, onPress }: Props) {
  const t = useT();
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={t('map.north')} onPress={onPress} style={styles.locateButton}>
      <Ionicons name="arrow-up" size={22} color={colors.target} style={dynamicStyles.northArrow(heading)} />
    </Pressable>
  );
}
