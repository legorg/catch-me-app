import { StyleSheet } from 'react-native';

export const colors = {
  background: '#10251c',
  card: '#1d372a',
  accent: '#a7e9bd',
  text: '#d1dfd6',
  hint: '#b9d1c2',
  placeholder: '#8ba996',
  error: '#ffb8b8',
};

export const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.background },
  content: { padding: 20, gap: 16, maxWidth: 680, width: '100%', alignSelf: 'center' },
  title: { color: '#ffffff', fontSize: 36, fontWeight: '800' },
  heading: { color: '#ffffff', fontSize: 20, fontWeight: '700' },
  text: { color: colors.text, fontSize: 15, lineHeight: 22 },
  error: { color: colors.error },
  mapContainer: { borderRadius: 16, overflow: 'hidden' },
  card: { backgroundColor: colors.card, borderRadius: 16, padding: 20, gap: 12 },
  row: { flexDirection: 'row', gap: 12 },
  button: { backgroundColor: colors.accent, borderRadius: 12, padding: 16, minHeight: 52, flexGrow: 1 },
  secondaryButton: { borderColor: colors.accent, borderWidth: 2, borderRadius: 12, padding: 14, minHeight: 52, flexGrow: 1 },
  dimmed: { opacity: 0.65 },
  buttonText: { color: colors.background, fontSize: 16, fontWeight: '700', textAlign: 'center' },
  secondaryButtonText: { color: colors.accent, fontSize: 16, fontWeight: '700', textAlign: 'center' },
  input: { backgroundColor: colors.background, color: '#ffffff', padding: 14, borderRadius: 8, fontSize: 15 },
  code: { color: colors.accent, fontSize: 28, fontWeight: '800', letterSpacing: 4 },
  hint: { color: colors.hint, fontSize: 13, lineHeight: 20 },
});
