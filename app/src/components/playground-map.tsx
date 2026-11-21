import { Platform, StyleSheet, View } from 'react-native';
import MapView, { Circle, Polyline, PROVIDER_GOOGLE } from 'react-native-maps';

import type { Coordinates, Playground } from '../api/types';
import { SKETCH_COLOR } from '../lobby';
import { DEFAULT_CENTER } from '../playground';
import { dynamicStyles, mapStyles as styles } from '../styles';
import { useMapDrawing } from '../use-map-drawing';
import { useMapHeading } from '../use-map-heading';
import { useMapLocate } from '../use-map-locate';
import { useMapRegion } from '../use-map-region';
import MapAlignButton from './map-align-button';
import MapCrosshair from './map-crosshair';
import MapDrawingOverlay from './map-drawing-overlay';
import MapGrid from './map-grid';
import MapLocateButton from './map-locate-button';
import MapNorthButton from './map-north-button';
import MapPin from './map-pin';

export type CellHighlight = { cell: string; title: string; color: string };

export type MapMarker = {
  id: string;
  coordinate: Coordinates;
  title: string;
  description: string;
  color: string;
  /** A number or short text shown inside the pin. */
  label?: string;
  /** A small dot centered on the coordinate instead of a pin, e.g. a moving player in the replay. */
  dot?: boolean;
};

type MapLine = { id: string; coordinates: Coordinates[]; color: string };

type MapCircle = { id: string; center: Coordinates; radius: number; color: string };

export type PlaygroundMapProps = {
  playground: Playground | null;
  editable: boolean;
  onSelect: (center: Coordinates) => void;
  highlights?: CellHighlight[];
  markers?: MapMarker[];
  /** Lines drawn over the grid, e.g. the team's sketches. */
  lines?: MapLine[];
  /** Circles drawn over the grid, e.g. the team's trap; radius in meters. */
  circles?: MapCircle[];
  /**
   * While set, finger moves draw a line instead of moving the map. Called with the line (at least two points)
   * when the finger lifts; the line stays visible until the returned promise settles.
   */
  onDraw?: (points: Coordinates[]) => Promise<void>;
  /** While set, the grid stays where it is and the point under the crosshair is picked whenever the map comes to rest. */
  onPick?: (point: Coordinates) => void;
  /** While set, a tap on the map reports the tapped point instead of moving the grid. */
  onTap?: (point: Coordinates) => void;
  /** Fill the parent instead of using the default height. */
  fullscreen?: boolean;
  /** Extra space around the location button, e.g. safe-area insets under a fullscreen map. */
  controlsInset?: { bottom?: number; right?: number };
  /** Disable the device location dot and locate button for caught spectators. */
  locationEnabled?: boolean;
  /** Allow turning the map with two fingers; a north button then turns it back. */
  rotatable?: boolean;
  /** Keep the map turned so that the grid's row 1 is on top, also after it refits; implies rotatable. */
  alignToGrid?: boolean;
  /** Called when the user turns the map, with where its top now points (degrees clockwise from north). */
  onHeadingChange?: (heading: number) => void;
};

export default function PlaygroundMap({ playground, editable, onSelect, highlights = [], markers = [], lines = [], circles = [], onDraw, onPick, onTap, fullscreen = false, controlsInset, locationEnabled = true, rotatable: rotatableProp = false, alignToGrid = false, onHeadingChange }: PlaygroundMapProps) {
  const center = playground?.center ?? DEFAULT_CENTER;
  const bounds = playground?.bounds;
  const rotatable = rotatableProp || alignToGrid;
  // On a turned grid, turning the map by the grid's rotation puts row 1 on top.
  const gridRotation = playground?.rotationDegrees ?? 0;
  const { map, anchor, select, onPress, onLayout, onMapReady, onRegionChangeComplete, isMuted } = useMapRegion({ playground, editable, onSelect, onPick, heading: alignToGrid ? gridRotation : undefined });
  const { drawing, stroke, responder } = useMapDrawing(map, onDraw);
  const { locationGranted, locating, locateError, locate } = useMapLocate(map, editable, select);
  const heading = useMapHeading(map, { rotatable, onRegionChangeComplete, isMuted, onHeadingChange });
  const showNorth = rotatable && !heading.pointsTo(0);
  const showAlign = rotatable && gridRotation !== 0 && !heading.pointsTo(gridRotation);
  return (
    <View style={fullscreen ? styles.fullscreen : styles.map} onLayout={onLayout}>
      <MapView
        ref={map}
        onMapReady={onMapReady}
        style={StyleSheet.absoluteFill}
        // iOS uses Apple Maps, which works in Expo Go without a Google key.
        provider={Platform.OS === 'android' ? PROVIDER_GOOGLE : undefined}
        initialRegion={{
          ...center,
          latitudeDelta: bounds ? (bounds.north - bounds.south) * 2 : 0.03,
          longitudeDelta: bounds ? (bounds.east - bounds.west) * 2 : 0.03,
        }}
        // The grid colors are made for a light map, so ignore the phone's dark mode.
        userInterfaceStyle="light"
        rotateEnabled={rotatable && !drawing}
        // Our own north button looks the same on both platforms.
        showsCompass={false}
        pitchEnabled={false}
        // While drawing, the finger draws and the map stays where it is.
        scrollEnabled={!drawing}
        zoomEnabled={!drawing}
        // The blue dot, once the user has allowed location.
        showsUserLocation={locationEnabled && locationGranted}
        showsMyLocationButton={false}
        toolbarEnabled={false}
        // Panning moves the grid: it stays centered on the map.
        onRegionChangeComplete={heading.onRegionChangeComplete}
        onPress={onTap ? (event) => onTap(event.nativeEvent.coordinate) : onPress}
      >
        {playground && <MapGrid playground={playground} highlights={highlights} onSelect={onTap ?? select} />}
        {circles.map(({ id, center: circleCenter, radius, color }) => (
          // The key includes the color so a status change redraws the native circle.
          <Circle key={`${id}-${color}`} center={circleCenter} radius={radius} strokeColor={color} strokeWidth={3} fillColor={`${color}40`} />
        ))}
        {lines.map(({ id, coordinates, color }) => (
          <Polyline key={id} coordinates={coordinates} strokeColor={color} strokeWidth={4} lineCap="round" lineJoin="round" />
        ))}
        {stroke.length >= 2 && <Polyline coordinates={stroke} strokeColor={SKETCH_COLOR} strokeWidth={4} lineCap="round" lineJoin="round" />}
        {markers.map(({ id, ...marker }) => (
          <MapPin key={marker.label === undefined ? id : `${id}-${marker.label}`} {...marker} />
        ))}
      </MapView>
      {(editable || onPick) && <MapCrosshair anchor={anchor} />}
      {drawing && <MapDrawingOverlay {...responder} />}
      {(showAlign || showNorth || locationEnabled) && (
        <View style={[styles.locate, dynamicStyles.locate(controlsInset)]}>
          {showAlign && <MapAlignButton onPress={() => heading.turnTo(gridRotation)} />}
          {showNorth && <MapNorthButton heading={heading.heading} onPress={() => heading.turnTo(0)} />}
          {locationEnabled && <MapLocateButton locating={locating} error={locateError} onLocate={locate} />}
        </View>
      )}
    </View>
  );
}
