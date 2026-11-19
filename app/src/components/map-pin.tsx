import { Platform, Text, View } from 'react-native';
import { Marker } from 'react-native-maps';

import { dynamicStyles, mapStyles as styles } from '../styles';
import type { MapMarker } from './playground-map';

// Apple Maps centers the pin on the coordinate; lift it by half its height so the tip lands there. Google Maps already does.
const PIN_OFFSET = Platform.OS === 'ios' ? { x: 0, y: -20 } : undefined;

// A numbered pin is 40 high with its tip at the bottom. iOS works out `anchor` from the view size before the view
// is laid out, which leaves the tip too low, so it gets an explicit offset (half the height) instead.
const NUMBERED_PIN_ANCHOR = Platform.OS === 'ios' ? undefined : { x: 0.5, y: 1 };
const NUMBERED_PIN_OFFSET = Platform.OS === 'ios' ? { x: 0, y: -20 } : undefined;

// A dot's middle is the coordinate: Apple Maps centers custom views already, Google Maps needs the anchor.
const DOT_ANCHOR = Platform.OS === 'ios' ? undefined : { x: 0.5, y: 0.5 };

/** The native pin cannot hold text, so a numbered marker is drawn: a round head with the number and a pointed tail. */
export default function MapPin({ coordinate, title, description, color, label, dot }: Omit<MapMarker, 'id'>) {
  if (dot) {
    return (
      <Marker coordinate={coordinate} title={title} description={description} anchor={DOT_ANCHOR} tracksViewChanges={false}>
        <View style={[styles.dot, dynamicStyles.pinHead(color)]}>
          {label !== undefined && <Text style={styles.dotLabel}>{label}</Text>}
        </View>
      </Marker>
    );
  }
  if (label === undefined) {
    return <Marker coordinate={coordinate} title={title} description={description} pinColor={color} centerOffset={PIN_OFFSET} />;
  }
  return (
    <Marker coordinate={coordinate} title={title} description={description} anchor={NUMBERED_PIN_ANCHOR} centerOffset={NUMBERED_PIN_OFFSET} tracksViewChanges={false}>
      <View style={styles.numberedPin}>
        {/* White triangle behind, colored one in front and over the head's border: together a bordered tail. */}
        <View style={styles.pinTailBorder} />
        <View style={[styles.pinHead, dynamicStyles.pinHead(color)]}>
          <Text style={styles.pinNumber}>{label}</Text>
        </View>
        <View style={[styles.pinTail, dynamicStyles.pinTail(color)]} />
      </View>
    </Marker>
  );
}
