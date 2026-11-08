import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { API_URL, loadDemoGreeting } from '@/api/client';

type ConnectionState = 'idle' | 'checking' | 'connected' | 'error';

export default function HomeScreen() {
  const [connection, setConnection] = useState<ConnectionState>('idle');
  const [message, setMessage] = useState('Start the Go server, then check your connection.');

  async function handleCheckConnection() {
    setConnection('checking');
    setMessage('Connecting to the backend...');

    try {
      const greeting = await loadDemoGreeting('Runner');
      setConnection('connected');
      setMessage(greeting.message);
    } catch (error) {
      setConnection('error');
      setMessage(error instanceof Error ? error.message : 'Could not reach the backend.');
    }
  }

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.intro}>
          <Text style={styles.eyebrow}>HUNTERS VS. RUNNERS</Text>
          <Text style={styles.title}>Catch me!</Text>
          <Text style={styles.subtitle}>Your next outdoor adventure starts here.</Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardTitle}>Generated API connection</Text>
          <Text selectable style={styles.url}>{API_URL}</Text>
          <Text
            accessibilityLiveRegion="polite"
            style={[
              styles.message,
              connection === 'connected' && styles.success,
              connection === 'error' && styles.error,
            ]}
          >
            {message}
          </Text>
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ disabled: connection === 'checking', busy: connection === 'checking' }}
            disabled={connection === 'checking'}
            onPress={handleCheckConnection}
            style={({ pressed }) => [
              styles.button,
              (pressed || connection === 'checking') && styles.buttonDimmed,
            ]}
          >
            {connection === 'checking' && <ActivityIndicator color="#10251c" />}
            <Text style={styles.buttonText}>
              {connection === 'checking' ? 'Checking...' : 'Check backend connection'}
            </Text>
          </Pressable>
        </View>

        <Text style={styles.hint}>
          Playing on a phone? Set your computer&apos;s Wi-Fi IP address in the root .env for Docker, or app/.env.local when running locally.
        </Text>
        <Text style={styles.footer}>Starter setup only. Game creation, joining, and live maps come next.</Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#10251c' },
  content: { flexGrow: 1, justifyContent: 'center', padding: 24, gap: 24, maxWidth: 560, width: '100%', alignSelf: 'center' },
  intro: { gap: 12 },
  eyebrow: { color: '#a7e9bd', fontSize: 12, fontWeight: '700', letterSpacing: 2 },
  title: { color: '#ffffff', fontSize: 48, fontWeight: '800' },
  subtitle: { color: '#d1dfd6', fontSize: 18, lineHeight: 26 },
  card: { backgroundColor: '#1d372a', borderRadius: 20, padding: 24, gap: 16 },
  cardTitle: { color: '#ffffff', fontSize: 20, fontWeight: '700' },
  url: { color: '#b9d1c2', fontSize: 14 },
  message: { color: '#d1dfd6', fontSize: 16, lineHeight: 24 },
  success: { color: '#a7e9bd' },
  error: { color: '#ffb8b8' },
  button: { backgroundColor: '#a7e9bd', borderRadius: 12, padding: 16, minHeight: 52, flexDirection: 'row', justifyContent: 'center', alignItems: 'center', gap: 10 },
  buttonDimmed: { opacity: 0.65 },
  buttonText: { color: '#10251c', fontSize: 16, fontWeight: '700', textAlign: 'center' },
  hint: { color: '#b9d1c2', fontSize: 14, lineHeight: 22 },
  footer: { color: '#b9d1c2', fontSize: 13, lineHeight: 20 },
});
