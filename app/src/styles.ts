import { StyleSheet, type ViewStyle } from 'react-native';

import palette from './palette.json';

const { camouflage, ...scheme } = palette;

/** Whether screens show the camouflage pattern; set in palette.json. */
export const showCamouflage = camouflage;

/** The app's colour scheme lives in palette.json (app.config.js reads it too). */
export const colors = {
  ...scheme,
  start: '#2f80ed',
  target: '#eb5757',
  /** Start and destination pins; the same value as ROUTE_PIN_COLOR. */
  route: '#00bcd4',
  white: '#ffffff',
  scrim: 'rgba(0,0,0,0.6)',
  /** The playground's fill on the map: the accent at 15% opacity. */
  playgroundFill: `${palette.accent}26`,
};

/** Android's header: a dark bar with a khaki line, as iOS keeps its native blurred large title. */
export const header = StyleSheet.create({
  background: { flex: 1, backgroundColor: colors.card, borderBottomWidth: 0, borderBottomColor: colors.accent },
});

export const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.background },
  content: { padding: 20, gap: 16, maxWidth: 680, width: '100%', alignSelf: 'center' },
  heading: { color: colors.white, fontSize: 20, fontWeight: '700' },
  text: { color: colors.text, fontSize: 15, lineHeight: 22 },
  error: { color: colors.error },
  mapContainer: { borderRadius: 16, overflow: 'hidden' },
  card: { backgroundColor: colors.card, borderRadius: 16, padding: 20, gap: 12 },
  row: { flexDirection: 'row', gap: 12, flexWrap: "wrap" },
  button: { backgroundColor: colors.accent, borderRadius: 12, padding: 16, minHeight: 52, flexGrow: 1 },
  secondaryButton: { borderColor: colors.accent, borderWidth: 2, borderRadius: 12, padding: 14, minHeight: 52, flexGrow: 1 },
  dimmed: { opacity: 0.65 },
  buttonText: { color: colors.background, fontSize: 16, fontWeight: '700', textAlign: 'center' },
  secondaryButtonText: { color: colors.accent, fontSize: 16, fontWeight: '700', textAlign: 'center' },
  input: { backgroundColor: colors.background, color: colors.white, padding: 14, borderRadius: 8, fontSize: 15 },
  code: { color: colors.accent, fontSize: 28, fontWeight: '800', letterSpacing: 4 },
  hint: { color: colors.hint, fontSize: 13, lineHeight: 20 },
  flex: { flex: 1 },
  gapSmall: { gap: 8 },
  gapTiny: { gap: 4 },
  gapMedium: { gap: 12 },
  alignEnd: { alignSelf: 'flex-end' },
  centerText: { textAlign: 'center' },
  boldText: { fontWeight: '700' },
  accentText: { color: colors.accent },
  startText: { color: colors.start },
  targetText: { color: colors.target },
  routeText: { color: colors.route },
  /** iOS: reaches past the scroll content, so the pattern also fills short screens, overscroll and wide iPads. */
  camouflage: { position: 'absolute', top: -1000, bottom: -1000, left: -400, right: -400 },
  centeredScreen: { justifyContent: 'center', padding: 20 },
  countdownScreen: { alignItems: 'center', justifyContent: 'center', padding: 32, gap: 16 },
  countdown: { color: colors.accent, fontSize: 160, fontWeight: '800' },
  modalBackdrop: { flex: 1, backgroundColor: colors.scrim, justifyContent: 'center', padding: 20 },
  hudZone: { flex: 1, justifyContent: 'flex-end', alignItems: 'flex-start', gap: 8 },
  jokersFooterBase: { paddingHorizontal: 20, paddingTop: 8 },
});

/** Dynamic styles: functions of props/state/insets. */
type Insets = { top: number; bottom: number; left: number; right: number };

export const dynamicStyles = {
  hudFill: (insets: Insets): ViewStyle => ({
    paddingTop: insets.top + 8,
    paddingBottom: insets.bottom + 12,
    paddingLeft: insets.left + 12,
    paddingRight: insets.right + 12,
  }),
  jokersFooter: (bottomInset: number): ViewStyle => ({ paddingBottom: Math.max(bottomInset, 16) }),
  fullscreenToggle: (insets?: Pick<Insets, 'top' | 'right'>): ViewStyle => ({
    top: (insets?.top ?? 0) + 12,
    right: (insets?.right ?? 0) + 12,
  }),
  pinHead: (color: string): ViewStyle => ({ backgroundColor: color }),
  pinTail: (color: string): ViewStyle => ({ borderTopColor: color }),
  crosshairAt: (anchor: { x: number; y: number }): ViewStyle => ({
    left: anchor.x - 12, top: anchor.y - 12, right: undefined, bottom: undefined, width: 24, height: 24,
  }),
  locate: (inset?: { right?: number; bottom?: number }): ViewStyle => ({
    right: 12 + (inset?.right ?? 0),
    bottom: 12 + (inset?.bottom ?? 0),
  }),
  /** Turns the north arrow against the map, so it keeps pointing north. */
  northArrow: (heading: number) => ({ transform: [{ rotate: `${-heading}deg` }] }),
  replayProgress: (share: number): ViewStyle => ({ width: `${share * 100}%` }),
  /** Android: the content's padding plus the navigation bar, so the last element clears it. */
  scrollBottom: (bottomInset: number): ViewStyle => ({ paddingBottom: (styles.content.padding as number) + bottomInset }),
  sliderShare: (share: number): ViewStyle => ({ width: `${share * 100}%` }),
  /** The thumb's center sits on the value; its margin pulls it back by half its width. */
  sliderThumb: (share: number): ViewStyle => ({ left: `${share * 100}%` }),
  camouflageTiles: (width: number): ViewStyle => ({ width }),
  navigationBar: (height: number): ViewStyle => ({ position: 'absolute', left: 0, right: 0, bottom: 0, height, backgroundColor: colors.background }),
};

/** Camouflage tiles, laid out by hand. */
export const camouflageStyles = StyleSheet.create({
  clip: { overflow: 'hidden' },
  tiles: { flexDirection: 'row', flexWrap: 'wrap' },
  tile: { width: 320, height: 320 },
});

/** Game screen overlay (HUD, info panel, pills, icon buttons). */
export const overlay = StyleSheet.create({
  fill: { ...StyleSheet.absoluteFill, gap: 8 },
  hud: { backgroundColor: colors.card, borderRadius: 12, padding: 12, gap: 2, opacity: 0.95 },
  info: { backgroundColor: colors.card, borderRadius: 12, padding: 16, maxHeight: '50%', alignSelf: 'stretch', flexGrow: 0 },
  pill: { backgroundColor: colors.card, borderRadius: 12, paddingHorizontal: 16, paddingVertical: 8, minHeight: 40, justifyContent: 'center', opacity: 0.95 },
  bar: { flexDirection: 'row', gap: 8 },
  icon: { backgroundColor: colors.card, borderRadius: 12, width: 44, height: 44, alignItems: 'center', justifyContent: 'center', opacity: 0.95 },
});

/** Language switch in the start screen's header. */
export const languageSwitch = StyleSheet.create({
  button: { paddingHorizontal: 6, paddingVertical: 4 },
  code: { color: colors.white, fontSize: 16, fontWeight: '700' },
});

/** Joker cell picker. */
export const picker = StyleSheet.create({
  row: { flexDirection: 'row', gap: 6 },
  rowTight: { gap: 3 },
  cell: { flex: 1, aspectRatio: 1, maxWidth: 56, minHeight: 40, borderRadius: 8, borderWidth: 1, borderColor: colors.accent, alignItems: 'center', justifyContent: 'center' },
  cellTight: { minHeight: 32, borderRadius: 6 },
  /** A cell with a word instead of a number: shares the row's whole width instead of being square. */
  cellWide: { aspectRatio: undefined, maxWidth: undefined, minHeight: 44, paddingHorizontal: 8 },
  selected: { backgroundColor: colors.accent },
  label: { color: colors.accent, fontSize: 14, fontWeight: '700' },
  labelSmall: { fontSize: 10 },
  selectedLabel: { color: colors.background },
  list: { gap: 6 },
});

/** Playground map, markers, crosshair, locate button. */
export const mapStyles = StyleSheet.create({
  map: { height: 360, width: '100%' },
  fullscreen: { flex: 1, width: '100%' },
  square: { width: '100%', aspectRatio: 1 },
  // Marks where the grid lands when the map comes to rest.
  crosshair: { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, alignItems: 'center', justifyContent: 'center' },
  crossH: { position: 'absolute', width: 24, height: 2, backgroundColor: colors.start },
  crossV: { position: 'absolute', width: 2, height: 24, backgroundColor: colors.start },
  locate: { position: 'absolute', right: 12, bottom: 12, alignItems: 'flex-end', gap: 6 },
  locateButton: { backgroundColor: colors.white, borderRadius: 8, width: 40, height: 40, alignItems: 'center', justifyContent: 'center' },
  locating: { opacity: 0.6 },
  locateError: { color: colors.white, backgroundColor: colors.target, borderRadius: 6, paddingHorizontal: 8, paddingVertical: 4, fontSize: 12 },
  numberedPin: { width: 32, height: 40 },
  pinHead: { position: 'absolute', top: 0, left: 0, width: 32, height: 32, borderRadius: 16, borderWidth: 2, borderColor: colors.white, alignItems: 'center', justifyContent: 'center' },
  pinTailBorder: { position: 'absolute', top: 26, left: 8, width: 0, height: 0, borderLeftWidth: 8, borderRightWidth: 8, borderTopWidth: 14, borderLeftColor: 'transparent', borderRightColor: 'transparent', borderTopColor: colors.white },
  pinTail: { position: 'absolute', top: 28, left: 11, width: 0, height: 0, borderLeftWidth: 5, borderRightWidth: 5, borderTopWidth: 10, borderLeftColor: 'transparent', borderRightColor: 'transparent' },
  pinNumber: { fontSize: 14, fontWeight: '800', color: colors.white },
  dot: { width: 18, height: 18, borderRadius: 9, borderWidth: 2, borderColor: colors.white, alignItems: 'center', justifyContent: 'center' },
  dotLabel: { fontSize: 9, fontWeight: '800', color: colors.white },
  label: { fontSize: 11, color: colors.background, backgroundColor: colors.white, padding: 2, borderRadius: 3, fontWeight: '700' },
});

/** Titled slider: a thin rail filled up to the value, with a round thumb, inside a tall touch area. */
export const sliderStyles = StyleSheet.create({
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'baseline', gap: 8 },
  touch: { height: 36, justifyContent: 'center' },
  rail: { height: 6, borderRadius: 3, backgroundColor: colors.background, overflow: 'hidden' },
  fill: { height: '100%', backgroundColor: colors.accent },
  thumb: { position: 'absolute', width: 22, height: 22, marginLeft: -11, borderRadius: 11, backgroundColor: colors.accent, borderWidth: 2, borderColor: colors.white },
});

/** Fullscreen map wrapper. */
export const fullscreenStyles = StyleSheet.create({
  full: { flex: 1, backgroundColor: colors.background },
  toggle: { position: 'absolute', backgroundColor: colors.background, borderRadius: 8, width: 40, height: 40, alignItems: 'center', justifyContent: 'center', opacity: 0.92 },
  footer: { maxHeight: '50%', backgroundColor: colors.card },
  footerScroll: { flexGrow: 0 },
  footerContent: { padding: 12 },
});

/** Result replay: a tall touch area around a thin rail. */
export const replayStyles = StyleSheet.create({
  scrubber: { height: 32, justifyContent: 'center' },
  rail: { height: 6, borderRadius: 3, backgroundColor: colors.background, overflow: 'hidden' },
  progress: { height: '100%', backgroundColor: colors.accent },
  legend: { flexDirection: 'row', flexWrap: 'wrap', columnGap: 16, rowGap: 6 },
  legendItem: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  legendDot: { width: 12, height: 12, borderRadius: 6 },
});
