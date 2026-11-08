import { StyleSheet, Text, View } from 'react-native';

import type { Coordinates, Playground } from '../api/generated/types.gen';

export type CellHighlight = { cell: string; title: string; color: string };

export type MapMarker = {
  id: string;
  coordinate: Coordinates;
  title: string;
  description: string;
  color: string;
};

export type PlaygroundMapProps = {
  playground: Playground | null;
  editable: boolean;
  onSelect: (center: Coordinates) => void;
  highlights?: CellHighlight[];
  markers?: MapMarker[];
};

// Metro resolves playground-map.native.tsx on Android/iOS. Never import the
// native maps library into the optional browser preview.
export default function PlaygroundMap({ playground, highlights = [], markers = [] }: PlaygroundMapProps) {
  return (
    <View style={styles.fallback}>
      <Text style={styles.text}>Playground maps are available in the Android and iOS app.</Text>
      {playground && <Text style={styles.text}>Saved grid: {playground.rows} × {playground.columns}</Text>}
      {highlights.map(({ cell, title }) => <Text key={cell + title} style={styles.text}>{title}: {cell}</Text>)}
      {markers.map(({ id, title, description }) => <Text key={id} style={styles.text}>{title}: {description}</Text>)}
    </View>
  );
}

const styles = StyleSheet.create({
  fallback: { height: 320, padding: 24, justifyContent: 'center', backgroundColor: '#1d372a', gap: 12 },
  text: { color: '#d1dfd6', textAlign: 'center' },
});
