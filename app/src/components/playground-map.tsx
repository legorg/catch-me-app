import { Ionicons } from '@expo/vector-icons';
import * as Location from 'expo-location';
import { useEffect, useRef, useState } from 'react';
import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import MapView, { Marker, Polygon, Polyline, PROVIDER_GOOGLE, type LatLng, type Region } from 'react-native-maps';

import type { Coordinates, Playground } from '../api/types';
import { useT } from '../i18n';
import { DEFAULT_CENTER, boundaryCoordinates, cellBounds, cellLabels, gridLines } from '../playground';

// Apple Maps centers the pin on the coordinate; lift it by half its height so the tip lands there. Google Maps already does.
const PIN_OFFSET = Platform.OS === 'ios' ? { x: 0, y: -20 } : undefined;

export type CellHighlight = { cell: string; title: string; color: string };

export type MapMarker = {
  id: string;
  coordinate: Coordinates;
  title: string;
  description: string;
  color: string;
  /** 0 to 1, default 1. */
  opacity?: number;
};

export type PlaygroundMapProps = {
  playground: Playground | null;
  editable: boolean;
  onSelect: (center: Coordinates) => void;
  highlights?: CellHighlight[];
  markers?: MapMarker[];
  /** Fill the parent instead of using the default height. */
  fullscreen?: boolean;
  /** Extra space around the location button, e.g. safe-area insets under a fullscreen map. */
  controlsInset?: { bottom?: number; right?: number };
};

export default function PlaygroundMap({ playground, editable, onSelect, highlights = [], markers = [], fullscreen = false, controlsInset }: PlaygroundMapProps) {
  const t = useT();
  // Where the map's center (and so the grid) is on screen. Native maps shift it from the middle of the
  // view, e.g. by safe-area margins under the notch, so the crosshair is placed by asking the map.
  const [anchor, setAnchor] = useState<{ x: number; y: number } | null>(null);
  const [permission, requestPermission] = Location.useForegroundPermissions();
  const [locating, setLocating] = useState(false);
  const [locateError, setLocateError] = useState('');
  const map = useRef<MapView>(null);
  const [ready, setReady] = useState(false);
  const center = playground?.center ?? DEFAULT_CENTER;
  const bounds = playground?.bounds;
  const boundsRef = useRef(bounds);
  useEffect(() => { boundsRef.current = bounds; });
  // Apple Maps (iOS) does not report isGesture reliably, so moves we cause ourselves (initial fit, refit, tap)
  // are muted for a moment instead.
  const muted = useRef(false);
  const mute = useRef((ms: number) => {
    muted.current = true;
    setTimeout(() => { muted.current = false; }, ms);
  });
  // Refit when the grid size changes, not while it is dragged: its span stays the same.
  const span = bounds ? bounds.north - bounds.south : undefined;
  useEffect(() => {
    if (ready && boundsRef.current) {
      mute.current(1200);
      map.current?.fitToCoordinates(boundaryCoordinates(boundsRef.current), {
        edgePadding: { top: 24, bottom: 24, left: 24, right: 24 }, animated: true,
      });
    }
  }, [ready, span]);
  const measureAnchor = async () => {
    try {
      const camera = await map.current?.getCamera();
      if (camera) setAnchor(await map.current!.pointForCoordinate(camera.center));
    } catch {
      // keep the middle of the view
    }
  };
  const pan = (region: Region) => {
    if (!editable || muted.current) return;
    onSelect({ latitude: region.latitude, longitude: region.longitude });
  };
  // A tap places the grid there and scrolls the map to it, so a following pan does not jump.
  const select = (coordinate: LatLng) => {
    if (!editable) return;
    onSelect(coordinate);
    mute.current(800);
    map.current?.animateCamera({ center: coordinate }, { duration: 300 });
  };
  const locate = async () => {
    if (locating) return;
    setLocating(true);
    setLocateError('');
    try {
      const granted = permission?.granted || (await requestPermission()).granted;
      if (!granted) {
        setLocateError(t('map.locationDenied'));
        return;
      }
      const { coords } = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      const here = { latitude: coords.latitude, longitude: coords.longitude };
      // An editable map puts the grid there; any other map just shows where you are.
      if (editable) select(here);
      else map.current?.animateCamera({ center: here }, { duration: 300 });
    } catch {
      setLocateError(t('map.locationFailed'));
    } finally {
      setLocating(false);
    }
  };
  return (
    <View style={fullscreen ? styles.fullscreen : styles.map} onLayout={() => setTimeout(() => void measureAnchor(), 300)}>
      <MapView
        ref={map}
        onMapReady={() => { setReady(true); void measureAnchor(); }}
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
        rotateEnabled={false}
        pitchEnabled={false}
        // The blue dot, once the user has allowed location.
        showsUserLocation={permission?.granted === true}
        showsMyLocationButton={false}
        toolbarEnabled={false}
        // Panning moves the grid: it stays centered on the map.
        onRegionChangeComplete={pan}
        onPress={(event) => select(event.nativeEvent.coordinate)}
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
            <Marker key={label} coordinate={coordinate} anchor={{ x: 0.5, y: 0.5 }} tracksViewChanges={false} stopPropagation onPress={() => select(coordinate)}>
              <Text style={styles.label}>{label}</Text>
            </Marker>
          ))}
        </>}
        {markers.map(({ id, coordinate, title, description, color, opacity }) => (
          // The marker opacity prop is unreliable on the native pin, so a fading marker is a dot view whose own opacity is drawn into it.
          opacity === undefined
            ? <Marker key={id} coordinate={coordinate} title={title} description={description} pinColor={color} centerOffset={PIN_OFFSET} />
            : <Marker key={`${id}-${opacity}`} coordinate={coordinate} title={title} description={description} anchor={{ x: 0.5, y: 0.5 }} tracksViewChanges={false}>
              <View style={[styles.dot, { backgroundColor: color, opacity }]} />
            </Marker>
        ))}
      </MapView>
    {editable && <View pointerEvents="none" style={[styles.crosshair, anchor && { left: anchor.x - 12, top: anchor.y - 12, right: undefined, bottom: undefined, width: 24, height: 24 }]}>
      <View style={styles.crossH} />
      <View style={styles.crossV} />
    </View>}
    <View style={[styles.locate, { right: 12 + (controlsInset?.right ?? 0), bottom: 12 + (controlsInset?.bottom ?? 0) }]}>
      {!!locateError && <Text style={styles.locateError}>{locateError}</Text>}
      <Pressable accessibilityRole="button" accessibilityLabel={t('map.myLocation')} disabled={locating} onPress={() => void locate()} style={[styles.locateButton, locating && styles.locating]}>
        <Ionicons name="locate" size={22} color="#10251c" />
      </Pressable>
    </View>
    </View>
  );
}

const styles = StyleSheet.create({
  map: { height: 360, width: '100%' },
  fullscreen: { flex: 1, width: '100%' },
  // Marks where the grid lands when the map comes to rest.
  crosshair: { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, alignItems: 'center', justifyContent: 'center' },
  crossH: { position: 'absolute', width: 24, height: 2, backgroundColor: '#2f80ed' },
  crossV: { position: 'absolute', width: 2, height: 24, backgroundColor: '#2f80ed' },
  locate: { position: 'absolute', right: 12, bottom: 12, alignItems: 'flex-end', gap: 6 },
  locateButton: { backgroundColor: '#ffffff', borderRadius: 8, width: 40, height: 40, alignItems: 'center', justifyContent: 'center' },
  locating: { opacity: 0.6 },
  locateError: { color: '#ffffff', backgroundColor: '#eb5757', borderRadius: 6, paddingHorizontal: 8, paddingVertical: 4, fontSize: 12 },
  dot: { width: 18, height: 18, borderRadius: 9, borderWidth: 2, borderColor: '#ffffff' },
  label: { fontSize: 11, color: '#10251c', backgroundColor: '#ffffff', padding: 2, borderRadius: 3, fontWeight: '700' },
});
