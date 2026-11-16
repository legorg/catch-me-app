import { Ionicons } from '@expo/vector-icons';
import * as Location from 'expo-location';
import { useEffect, useRef, useState } from 'react';
import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import { dynamicStyles, mapStyles as styles } from '../styles';
import MapView, { Marker, Polygon, Polyline, PROVIDER_GOOGLE, type LatLng, type Region } from 'react-native-maps';

import type { Coordinates, Playground } from '../api/types';
import { useT } from '../i18n';
import { SKETCH_COLOR } from '../lobby';
import { DEFAULT_CENTER, boundaryCoordinates, cellBounds, cellLabels, gridLines } from '../playground';

// Apple Maps centers the pin on the coordinate; lift it by half its height so the tip lands there. Google Maps already does.
const PIN_OFFSET = Platform.OS === 'ios' ? { x: 0, y: -20 } : undefined;

// A numbered pin is 40 high with its tip at the bottom. iOS works out `anchor` from the view size before the view
// is laid out, which leaves the tip too low, so it gets an explicit offset (half the height) instead.
const NUMBERED_PIN_ANCHOR = Platform.OS === 'ios' ? undefined : { x: 0.5, y: 1 };
const NUMBERED_PIN_OFFSET = Platform.OS === 'ios' ? { x: 0, y: -20 } : undefined;

// A finger must move this far (in points) before the line gets a new point, which keeps lines small.
const MIN_STROKE_STEP = 6;

export type CellHighlight = { cell: string; title: string; color: string };

export type MapMarker = {
  id: string;
  coordinate: Coordinates;
  title: string;
  description: string;
  color: string;
  /** A number or short text shown inside the pin. */
  label?: string;
};

type MapLine = { id: string; coordinates: Coordinates[]; color: string };

export type PlaygroundMapProps = {
  playground: Playground | null;
  editable: boolean;
  onSelect: (center: Coordinates) => void;
  highlights?: CellHighlight[];
  markers?: MapMarker[];
  /** Lines drawn over the grid, e.g. the team's sketches. */
  lines?: MapLine[];
  /**
   * While set, finger moves draw a line instead of moving the map. Called with the line (at least two points)
   * when the finger lifts; the line stays visible until the returned promise settles.
   */
  onDraw?: (points: Coordinates[]) => Promise<void>;
  /** Fill the parent instead of using the default height. */
  fullscreen?: boolean;
  /** Extra space around the location button, e.g. safe-area insets under a fullscreen map. */
  controlsInset?: { bottom?: number; right?: number };
};

export default function PlaygroundMap({ playground, editable, onSelect, highlights = [], markers = [], lines = [], onDraw, fullscreen = false, controlsInset }: PlaygroundMapProps) {
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
  // Drawing: each finger position becomes a map coordinate (asynchronously, so they are collected by index).
  const [stroke, setStroke] = useState<Coordinates[]>([]);
  const currentStroke = useRef<{ pending: Promise<LatLng | null>[]; resolved: (LatLng | null)[]; last?: { x: number; y: number } } | null>(null);
  const addPoint = (x: number, y: number) => {
    const current = currentStroke.current;
    if (!current || !map.current || (current.last && Math.hypot(x - current.last.x, y - current.last.y) < MIN_STROKE_STEP)) return;
    current.last = { x, y };
    const index = current.pending.length;
    const point = map.current.coordinateForPoint({ x, y }).catch(() => null);
    current.pending.push(point);
    void point.then((coordinate) => {
      current.resolved[index] = coordinate;
      if (currentStroke.current === current) setStroke(current.resolved.filter((c): c is LatLng => !!c));
    });
  };
  // Uses the onDraw of the moment the finger lifted, so switching drawing off right after still sends the line.
  const finishStroke = async () => {
    const send = onDraw;
    const current = currentStroke.current;
    currentStroke.current = null;
    if (!current) return;
    const points = (await Promise.all(current.pending)).filter((c): c is LatLng => !!c);
    try {
      if (points.length >= 2) await send?.(points.map(({ latitude, longitude }) => ({ latitude, longitude })));
    } finally {
      // A new line may have been started meanwhile; it shows its own preview.
      if (!currentStroke.current) setStroke([]);
    }
  };
  const startStroke = (x: number, y: number) => {
    currentStroke.current = { pending: [], resolved: [] };
    addPoint(x, y);
  };
  const drawing = !!onDraw;
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
        // While drawing, the finger draws and the map stays where it is.
        scrollEnabled={!drawing}
        zoomEnabled={!drawing}
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
        {lines.map(({ id, coordinates, color }) => (
          <Polyline key={id} coordinates={coordinates} strokeColor={color} strokeWidth={4} lineCap="round" lineJoin="round" />
        ))}
        {stroke.length >= 2 && <Polyline coordinates={stroke} strokeColor={SKETCH_COLOR} strokeWidth={4} lineCap="round" lineJoin="round" />}
        {markers.map(({ id, coordinate, title, description, color, label }) => (
          // The native pin cannot hold text, so a numbered marker is drawn: a round head with the number and a pointed tail.
          label === undefined
            ? <Marker key={id} coordinate={coordinate} title={title} description={description} pinColor={color} centerOffset={PIN_OFFSET} />
            : <Marker key={`${id}-${label}`} coordinate={coordinate} title={title} description={description} anchor={NUMBERED_PIN_ANCHOR} centerOffset={NUMBERED_PIN_OFFSET} tracksViewChanges={false}>
              <View style={styles.numberedPin}>
                {/* White triangle behind, colored one in front and over the head's border: together a bordered tail. */}
                <View style={styles.pinTailBorder} />
                <View style={[styles.pinHead, dynamicStyles.pinHead(color)]}>
                  <Text style={styles.pinNumber}>{label}</Text>
                </View>
                <View style={[styles.pinTail, dynamicStyles.pinTail(color)]} />
              </View>
            </Marker>
        ))}
      </MapView>
    {editable && <View pointerEvents="none" style={[styles.crosshair, anchor && dynamicStyles.crosshairAt(anchor)]}>
      <View style={styles.crossH} />
      <View style={styles.crossV} />
    </View>}
    {/* Over the map, under the location button: catches the finger while drawing. */}
    {drawing && <View
      style={StyleSheet.absoluteFill}
      onStartShouldSetResponder={() => true}
      onMoveShouldSetResponder={() => true}
      onResponderGrant={(event) => startStroke(event.nativeEvent.locationX, event.nativeEvent.locationY)}
      onResponderMove={(event) => addPoint(event.nativeEvent.locationX, event.nativeEvent.locationY)}
      onResponderRelease={() => void finishStroke()}
      onResponderTerminate={() => void finishStroke()}
    />}
    <View style={[styles.locate, dynamicStyles.locate(controlsInset)]}>
      {!!locateError && <Text style={styles.locateError}>{locateError}</Text>}
      <Pressable accessibilityRole="button" accessibilityLabel={t('map.myLocation')} disabled={locating} onPress={() => void locate()} style={[styles.locateButton, locating && styles.locating]}>
        <Ionicons name="locate" size={22} color="#10251c" />
      </Pressable>
    </View>
    </View>
  );
}

