import { View } from 'react-native';

import { dynamicStyles, mapStyles as styles } from '../styles';

/** Marks where the grid lands when the map comes to rest. */
export default function MapCrosshair({ anchor }: { anchor: { x: number; y: number } | null }) {
  return (
    <View pointerEvents="none" style={[styles.crosshair, anchor && dynamicStyles.crosshairAt(anchor)]}>
      <View style={styles.crossH} />
      <View style={styles.crossV} />
    </View>
  );
}
