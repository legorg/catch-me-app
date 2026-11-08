import { router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { ActivityIndicator, ScrollView, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { JOIN_CODE_PATTERN, MAX_NAME_LENGTH, isValidPlayerName, joinGame, normalizeJoinCode } from '@/api/client';
import Button from '@/components/button';
import { playerSession } from '@/player-session';
import { colors, styles } from '@/styles';

export default function HomeScreen() {
  const [code, setCode] = useState('');
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const canJoin = JOIN_CODE_PATTERN.test(normalizeJoinCode(code)) && isValidPlayerName(name);

  async function join() {
    if (busy || !canJoin) return;
    setBusy(true);
    setError('');
    try {
      const joined = await joinGame(code, name);
      playerSession.set(joined.playerToken);
      router.replace('/lobby');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not join the game.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text style={styles.title}>Catch me!</Text>
        <Text style={styles.text}>
          Runners try to reach their target, hunters try to catch them. The app is the referee, so everyone can play.
        </Text>
        <View style={styles.card}>
          <Text style={styles.heading}>Join a game</Text>
          <TextInput
            accessibilityLabel="Join code"
            placeholder="Join code"
            placeholderTextColor={colors.placeholder}
            value={code}
            onChangeText={setCode}
            editable={!busy}
            autoCapitalize="characters"
            autoCorrect={false}
            maxLength={8}
            style={styles.input}
          />
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
          <Text style={styles.hint}>The code decides your team. Ask the game creator or a teammate for it.</Text>
          {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
          {busy && <ActivityIndicator color={colors.accent} />}
          <Button label="Join game" onPress={() => void join()} disabled={busy || !canJoin} />
        </View>
        <View style={styles.card}>
          <Text style={styles.heading}>Start a new game</Text>
          <Text style={styles.text}>Choose a playground on the map and get one join code per team.</Text>
          <Button label="Create a game" variant="secondary" onPress={() => router.push('/create')} disabled={busy} />
        </View>
        <Text style={styles.hint}>Local development only. Location sharing and game rules are not implemented yet.</Text>
      </ScrollView>
    </SafeAreaView>
  );
}
