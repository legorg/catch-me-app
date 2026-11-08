import { useEffect, useRef, useState } from 'react';
import { Platform, StyleSheet, Text } from 'react-native';
import MapView, { Marker, Polygon, Polyline, PROVIDER_GOOGLE } from 'react-native-maps';

import { boundaryCoordinates, cellBounds, cellLabels, gridLines } from '../playground';
import type { PlaygroundMapProps } from './playground-map';

export default function PlaygroundMap({ playground, editable, onSelect, highlights = [], markers = [] }: PlaygroundMapProps) {
  const map = useRef<MapView>(null);
  const [ready, setReady] = useState(false);
  const center = playground?.center ?? { latitude: 50.9375, longitude: 6.9603 }; //Cologne
  const bounds = playground?.bounds;
  const north = bounds?.north;
  const south = bounds?.south;
  const east = bounds?.east;
  const west = bounds?.west;
  useEffect(() => {
    if (ready && north !== undefined && south !== undefined && east !== undefined && west !== undefined) {
      map.current?.fitToCoordinates(boundaryCoordinates({ north, south, east, west }), {
        edgePadding: { top: 24, bottom: 24, left: 24, right: 24 }, animated: true,
      });
    }
  }, [ready, north, south, east, west]);
  return (
    <MapView
      ref={map}
      onMapReady={() => setReady(true)}
      style={styles.map}
      // iOS uses Apple Maps, which works in Expo Go without a Google key.
      provider={Platform.OS === 'android' ? PROVIDER_GOOGLE : undefined}
      initialRegion={{
        ...center,
        latitudeDelta: bounds ? (bounds.north - bounds.south) * 2 : 0.03,
        longitudeDelta: bounds ? (bounds.east - bounds.west) * 2 : 0.03,
      }}
      rotateEnabled={false}
      pitchEnabled={false}
      showsUserLocation={false}
      showsMyLocationButton={false}
      toolbarEnabled={false}
      onPress={(event) => { if (editable) onSelect(event.nativeEvent.coordinate); }}
    >
      {playground && <>
        <Polygon coordinates={boundaryCoordinates(playground.bounds)} strokeColor="#10251c" strokeWidth={3} fillColor="rgba(167,233,189,0.15)" />
        {highlights.map(({ cell, color }) => (
          <Polygon key={cell + color} coordinates={boundaryCoordinates(cellBounds(playground, cell))} strokeColor={color} strokeWidth={3} fillColor={`${color}55`} tappable={false} />
        ))}
        {gridLines(playground).map((coordinates, index) => (
          <Polyline key={index} coordinates={coordinates} strokeColor="#10251c" strokeWidth={1} />
        ))}
        {cellLabels(playground).map(({ label, coordinate }) => (
          <Marker key={label} coordinate={coordinate} anchor={{ x: 0.5, y: 0.5 }} tracksViewChanges={false} stopPropagation onPress={() => { if (editable) onSelect(coordinate); }}>
            <Text style={styles.label}>{label}</Text>
          </Marker>
        ))}
      </>}
      {markers.map(({ id, coordinate, title, description, color }) => (
        <Marker key={id} coordinate={coordinate} title={title} description={description} pinColor={color} />
      ))}
    </MapView>
  );
}

const styles = StyleSheet.create({
  map: { height: 360, width: '100%' },
  label: { fontSize: 11, color: '#10251c', backgroundColor: '#ffffff', padding: 2, borderRadius: 3, fontWeight: '700' },
});
