import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { ActivityIndicator, Platform, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { createGame, loadGame, type Coordinates, type GameResponse } from '@/api/client';
import PlaygroundMap from '@/components/playground-map';
import { DEFAULT_CELL_SIZE_METERS, parseCellSizeKm, previewPlayground } from '@/playground';

export default function HomeScreen() {
  const [selected, setSelected] = useState<Coordinates | null>(null);
  const [cellSizeText, setCellSizeText] = useState(String(DEFAULT_CELL_SIZE_METERS / 1000));
  const [game, setGame] = useState<GameResponse | null>(null);
  const [sessionId, setSessionId] = useState('');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('Pan and zoom, then tap to choose the center.');
  const [error, setError] = useState(false);
  const cellSize = parseCellSizeKm(cellSizeText);
  let playground = game?.playground ?? null;
  let sizeError = !game && cellSize === null ? 'Enter 0.1–5 km in steps of 0.1 km.' : '';
  if (!game && selected && cellSize !== null) {
    try {
      playground = previewPlayground(selected, cellSize);
    } catch (err) {
      sizeError = err instanceof Error ? err.message : 'Invalid playground size.';
    }
  }
  const displayedSize = game?.playground.cellSizeMeters ?? cellSize;
  const cannotCreate = !playground || !!sizeError || Platform.OS === 'web';

  function selectCenter(center: Coordinates) {
    try {
      if (cellSize === null) throw new Error('Enter a valid cell size first.');
      previewPlayground(center, cellSize);
      setSelected(center);
      setError(false);
      setMessage('Check the entire grid is suitable and permitted for outdoor play, then create the game.');
    } catch (err) {
      setError(true);
      setMessage(err instanceof Error ? err.message : 'Invalid playground center.');
    }
  }

  async function saveOrLoad(action: 'save' | 'load') {
    if (busy || (action === 'save' && (!selected || game || cannotCreate))) return;
    setBusy(true);
    setError(false);
    setMessage(action === 'save' ? 'Saving playground...' : 'Loading playground...');
    try {
      const saved = action === 'save' ? await createGame(selected!, cellSize!) : await loadGame(sessionId.trim());
      setGame(saved);
      setSessionId(saved.sessionId);
      setMessage('Playground saved. Keep the session ID to load it again.');
    } catch (err) {
      setError(true);
      setMessage(err instanceof Error ? err.message : 'Could not reach the backend.');
    } finally {
      setBusy(false);
    }
  }

  function startNew() {
    setGame(null);
    setSelected(null);
    setCellSizeText(String(DEFAULT_CELL_SIZE_METERS / 1000));
    setError(false);
    setMessage('Pan and zoom, then tap to choose the center.');
  }

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.title}>Catch me!</Text>
        <Text style={styles.heading}>{game ? 'Saved playground' : 'Choose your playground'}</Text>
        <Text style={styles.text}>
          6 × 6 cells{displayedSize !== null ? ` · approximately ${displayedSize / 1000} km × ${displayedSize / 1000} km each · ${(displayedSize * 6 / 1000).toFixed(1)} km × ${(displayedSize * 6 / 1000).toFixed(1)} km total` : ''}
        </Text>
        {!game && <View style={styles.card}>
          <Text style={styles.heading}>Cell side length (km)</Text>
          <TextInput
            accessibilityLabel="Cell side length in kilometers"
            value={cellSizeText}
            onChangeText={setCellSizeText}
            editable={!busy}
            keyboardType="decimal-pad"
            autoCorrect={false}
            maxLength={8}
            style={styles.input}
          />
          <Text style={styles.hint}>0.1–5 km, in steps of 0.1 km. Both comma and dot decimals work.</Text>
          {!!sizeError && <Text accessibilityLiveRegion="polite" style={styles.error}>{sizeError}</Text>}
        </View>}
        <View style={styles.mapContainer}>
          <PlaygroundMap key={game?.sessionId ?? 'draft'} playground={playground} editable={!game && !busy} onSelect={selectCenter} />
        </View>
        {playground && <Text style={styles.text}>
          Center: {playground.center.latitude.toFixed(6)}, {playground.center.longitude.toFixed(6)} · Columns A–F, rows 1–6 from north to south
        </Text>}
        <Text accessibilityLiveRegion="polite" style={[styles.text, error && styles.error]}>{message}</Text>
        {busy && <ActivityIndicator color="#a7e9bd" />}
        <Pressable
          accessibilityRole="button"
          accessibilityState={{ disabled: busy || (!game && cannotCreate) }}
          disabled={busy || (!game && cannotCreate)}
          onPress={() => { if (game) startNew(); else void saveOrLoad('save'); }}
          style={({ pressed }) => [styles.button, (pressed || busy || (!game && cannotCreate)) && styles.dimmed]}
        >
          <Text style={styles.buttonText}>{game ? 'Choose a new playground' : 'Create game'}</Text>
        </Pressable>
        {game && <View style={styles.card}>
          <Text style={styles.heading}>Session ID</Text>
          <Text selectable style={styles.sessionId}>{game.sessionId}</Text>
          <Text style={styles.text}>Created: {new Date(game.createdAt).toLocaleString()}</Text>
        </View>}
        <View style={styles.card}>
          <Text style={styles.heading}>Load an existing playground</Text>
          <TextInput
            accessibilityLabel="Game session ID"
            placeholder="32-character session ID"
            placeholderTextColor="#8ba996"
            value={sessionId}
            onChangeText={setSessionId}
            editable={!busy}
            autoCapitalize="none"
            autoCorrect={false}
            maxLength={32}
            style={styles.input}
          />
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ disabled: busy || !sessionId.trim() }}
            disabled={busy || !sessionId.trim()}
            onPress={() => void saveOrLoad('load')}
            style={({ pressed }) => [styles.button, (pressed || busy || !sessionId.trim()) && styles.dimmed]}
          >
            <Text style={styles.buttonText}>Load playground</Text>
          </Pressable>
        </View>
        <Text style={styles.hint}>Local development only. No player tracking or game rules yet. A session ID is not a team code or an authentication credential.</Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#10251c' },
  content: { padding: 20, gap: 16, maxWidth: 680, width: '100%', alignSelf: 'center' },
  title: { color: '#ffffff', fontSize: 36, fontWeight: '800' },
  heading: { color: '#ffffff', fontSize: 20, fontWeight: '700' },
  text: { color: '#d1dfd6', fontSize: 15, lineHeight: 22 },
  error: { color: '#ffb8b8' },
  mapContainer: { borderRadius: 16, overflow: 'hidden' },
  card: { backgroundColor: '#1d372a', borderRadius: 16, padding: 20, gap: 12 },
  button: { backgroundColor: '#a7e9bd', borderRadius: 12, padding: 16, minHeight: 52 },
  dimmed: { opacity: 0.65 },
  buttonText: { color: '#10251c', fontSize: 16, fontWeight: '700', textAlign: 'center' },
  input: { backgroundColor: '#10251c', color: '#ffffff', padding: 14, borderRadius: 8, fontSize: 15 },
  sessionId: { color: '#a7e9bd', fontSize: 16 },
  hint: { color: '#b9d1c2', fontSize: 13, lineHeight: 20 },
});
