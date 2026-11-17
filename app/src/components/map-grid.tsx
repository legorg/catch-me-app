import { Text } from 'react-native';
import { Marker, Polygon, Polyline, type LatLng } from 'react-native-maps';

import type { Playground } from '../api/types';
import { boundaryCoordinates, cellBounds, cellLabels, gridLines } from '../playground';
import { mapStyles as styles } from '../styles';
import type { CellHighlight } from './playground-map';

type Props = {
  playground: Playground;
  highlights: CellHighlight[];
  onSelect: (coordinate: LatLng) => void;
};

/** The playground's boundary, highlighted cells, grid lines and cell labels. */
export default function MapGrid({ playground, highlights, onSelect }: Props) {
  return (
    <>
      <Polygon coordinates={boundaryCoordinates(playground.bounds)} strokeColor="#10251c" strokeWidth={3} fillColor="rgba(167,233,189,0.15)" />
      {highlights.map(({ cell, color }, index) => (
        <Polygon key={`${index}-${cell}`} coordinates={boundaryCoordinates(cellBounds(playground, cell))} strokeColor={color} strokeWidth={3} fillColor={`${color}55`} tappable={false} />
      ))}
      {gridLines(playground).map((coordinates, index) => (
        <Polyline key={index} coordinates={coordinates} strokeColor="#10251c" strokeWidth={1} />
      ))}
      {cellLabels(playground).map(({ label, coordinate }) => (
        <Marker key={label} coordinate={coordinate} anchor={{ x: 0.5, y: 0.5 }} tracksViewChanges={false} stopPropagation onPress={() => onSelect(coordinate)}>
          <Text style={styles.label}>{label}</Text>
        </Marker>
      ))}
    </>
  );
}
