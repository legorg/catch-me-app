import { router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { ActivityIndicator, Platform, ScrollView, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { MAX_NAME_LENGTH, createGame, isValidPlayerName, joinGame, type Coordinates, type CreateGameResponse } from '@/api/client';
import Button from '@/components/button';
import PlaygroundMap from '@/components/playground-map';
import { playerSession } from '@/player-session';
import { DEFAULT_CELL_SIZE_METERS, parseCellSizeKm, previewPlayground } from '@/playground';
import { colors, styles } from '@/styles';

export default function CreateScreen() {
  const [selected, setSelected] = useState<Coordinates | null>(null);
  const [cellSizeText, setCellSizeText] = useState(String(DEFAULT_CELL_SIZE_METERS / 1000));
  const [created, setCreated] = useState<CreateGameResponse | null>(null);
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('Pan and zoom, then tap to choose the center.');
  const [error, setError] = useState(false);
  const cellSize = parseCellSizeKm(cellSizeText);
  let playground = created?.game.playground ?? null;
  let sizeError = !created && cellSize === null ? 'Enter 0.1–5 km in steps of 0.1 km.' : '';
  if (!created && selected && cellSize !== null) {
    try {
      playground = previewPlayground(selected, cellSize);
    } catch (err) {
      sizeError = err instanceof Error ? err.message : 'Invalid playground size.';
    }
  }
  const displayedSize = created?.game.playground.cellSizeMeters ?? cellSize;
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

  async function run(action: () => Promise<void>, progress: string) {
    if (busy) return;
    setBusy(true);
    setError(false);
    setMessage(progress);
    try {
      await action();
    } catch (err) {
      setError(true);
      setMessage(err instanceof Error ? err.message : 'Could not reach the backend.');
    } finally {
      setBusy(false);
    }
  }

  const create = () => run(async () => {
    if (!selected || cellSize === null || cannotCreate) return;
    setCreated(await createGame(selected, cellSize));
    setMessage('Game created. Share the codes, then join a team yourself.');
  }, 'Creating game...');

  const joinAs = (code: string) => run(async () => {
    const joined = await joinGame(code, name);
    playerSession.set(joined.playerToken);
    router.replace('/lobby');
  }, 'Joining...');

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.heading}>{created ? 'Game created' : 'Choose your playground'}</Text>
        <Text style={styles.text}>
          6 × 6 cells{displayedSize !== null ? ` · approximately ${displayedSize / 1000} km × ${displayedSize / 1000} km each · ${(displayedSize * 6 / 1000).toFixed(1)} km × ${(displayedSize * 6 / 1000).toFixed(1)} km total` : ''}
        </Text>
        {!created && <View style={styles.card}>
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
          {!!sizeError && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{sizeError}</Text>}
        </View>}
        <View style={styles.mapContainer}>
          <PlaygroundMap key={created?.game.sessionId ?? 'draft'} playground={playground} editable={!created && !busy} onSelect={selectCenter} />
        </View>
        {playground && <Text style={styles.text}>
          Center: {playground.center.latitude.toFixed(6)}, {playground.center.longitude.toFixed(6)} · Columns A–F, rows 1–6 from north to south
        </Text>}
        <Text accessibilityLiveRegion="polite" style={[styles.text, error && styles.error]}>{message}</Text>
        {busy && <ActivityIndicator color={colors.accent} />}
        {!created && <Button label="Create game" onPress={() => void create()} disabled={busy || cannotCreate} />}
        {created && <>
          <View style={styles.card}>
            <Text style={styles.heading}>Join codes</Text>
            <View style={styles.row}>
              <View style={{ flex: 1 }}>
                <Text style={styles.text}>Runners</Text>
                <Text selectable style={styles.code}>{created.joinCodes.runner}</Text>
              </View>
              <View style={{ flex: 1 }}>
                <Text style={styles.text}>Hunters</Text>
                <Text selectable style={styles.code}>{created.joinCodes.hunter}</Text>
              </View>
            </View>
            <Text style={styles.hint}>Give each player the code of their team. Players can join until the game starts.</Text>
          </View>
          <View style={styles.card}>
            <Text style={styles.heading}>Join yourself</Text>
            <TextInput
              accessibilityLabel="Your name"
              placeholder="Your name"
              placeholderTextColor={colors.placeholder}
              value={name}
              onChangeText={setName}
              editable={!busy}
              autoCorrect={false}
              maxLength={MAX_NAME_LENGTH}
              style={styles.input}
            />
            <View style={styles.row}>
              <Button label="Join as runner" onPress={() => void joinAs(created.joinCodes.runner)} disabled={busy || !isValidPlayerName(name)} />
              <Button label="Join as hunter" onPress={() => void joinAs(created.joinCodes.hunter)} disabled={busy || !isValidPlayerName(name)} />
            </View>
          </View>
        </>}
      </ScrollView>
    </SafeAreaView>
  );
}
