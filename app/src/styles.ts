import { StyleSheet, type ViewStyle } from 'react-native';

export const colors = {
  background: '#10251c',
  card: '#1d372a',
  accent: '#a7e9bd',
  text: '#d1dfd6',
  hint: '#b9d1c2',
  placeholder: '#8ba996',
  error: '#ffb8b8',
  start: '#2f80ed',
  target: '#eb5757',
  white: '#ffffff',
  scrim: 'rgba(0,0,0,0.6)',
};

export const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.background },
  content: { padding: 20, gap: 16, maxWidth: 680, width: '100%', alignSelf: 'center' },
  heading: { color: '#ffffff', fontSize: 20, fontWeight: '700' },
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
  input: { backgroundColor: colors.background, color: '#ffffff', padding: 14, borderRadius: 8, fontSize: 15 },
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
  languageMenu: (anchor: { top: number; right: number }): ViewStyle => ({ top: anchor.top, right: anchor.right }),
  /** Android draws edge-to-edge and the start screen has no header, so its title must clear the status bar itself. */
  homeTitle: (topInset: number): ViewStyle => ({ paddingTop: topInset }),
  replayProgress: (share: number): ViewStyle => ({ width: `${share * 100}%` }),
};

/** Game screen overlay (HUD, info panel, pills, icon buttons). */
export const overlay = StyleSheet.create({
  fill: { ...StyleSheet.absoluteFill, gap: 8 },
  hud: { backgroundColor: colors.card, borderRadius: 12, padding: 12, gap: 2, opacity: 0.95 },
  info: { backgroundColor: colors.card, borderRadius: 12, padding: 16, maxHeight: '50%', alignSelf: 'stretch', flexGrow: 0 },
  pill: { backgroundColor: colors.card, borderRadius: 12, paddingHorizontal: 16, paddingVertical: 8, minHeight: 40, justifyContent: 'center', opacity: 0.95 },
  bar: { flexDirection: 'row', gap: 8 },
  icon: { backgroundColor: colors.card, borderRadius: 12, width: 44, height: 44, alignItems: 'center', justifyContent: 'center', opacity: 0.95 },
});

/** Start screen title row and its language dropdown. */
export const languageSwitch = StyleSheet.create({
  titleRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: 12 },
  title: { color: colors.white, fontSize: 34, fontWeight: '700', flexShrink: 1 },
  button: { flexDirection: 'row', alignItems: 'center', gap: 4, backgroundColor: colors.card, borderRadius: 12, paddingHorizontal: 10, paddingVertical: 6 },
  flag: { fontSize: 24 },
  caret: { color: colors.accent, fontSize: 14 },
  backdrop: { ...StyleSheet.absoluteFill },
  menu: { position: 'absolute', backgroundColor: colors.card, borderRadius: 12, paddingVertical: 6, minWidth: 160, elevation: 8, shadowColor: '#000', shadowOpacity: 0.4, shadowRadius: 12, shadowOffset: { width: 0, height: 4 } },
  item: { flexDirection: 'row', alignItems: 'center', gap: 12, paddingHorizontal: 16, paddingVertical: 10 },
  pressed: { backgroundColor: colors.background },
  name: { color: colors.text, fontSize: 16 },
  current: { color: colors.accent, fontWeight: '700' },
});

/** Joker cell picker. */
export const picker = StyleSheet.create({
  row: { flexDirection: 'row', gap: 6 },
  rowTight: { gap: 3 },
  cell: { flex: 1, aspectRatio: 1, maxWidth: 56, minHeight: 40, borderRadius: 8, borderWidth: 1, borderColor: colors.accent, alignItems: 'center', justifyContent: 'center' },
  cellTight: { minHeight: 32, borderRadius: 6 },
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

/** Fullscreen map wrapper. */
export const fullscreenStyles = StyleSheet.create({
  full: { flex: 1, backgroundColor: colors.background },
  toggle: { position: 'absolute', backgroundColor: colors.background, borderRadius: 8, width: 40, height: 40, alignItems: 'center', justifyContent: 'center', opacity: 0.92 },
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

