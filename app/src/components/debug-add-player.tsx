import { useState } from 'react';
import { Modal, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import {
  ApiError, MAX_NAME_LENGTH, isValidPlayerName, joinGame, setHunterTeam, setReady, updatePosition, type JoinCodes,
} from '@/api/client';
import type { Bounds, Coordinates } from '@/api/types';
import Button from '@/components/button';
import { colors, styles } from '@/styles';

const MOVE_INTERVAL_MS = 5000;
// About 120 m per step, i.e. a brisk jog.
const STEP_DEGREES = 0.0011;
const MAX_TURN = 0.6;

type Bot = { token: string; position: Coordinates; bounds: Bounds; heading: number };

// Bots outlive the lobby, so they keep reporting after the game screen replaces it.
const bots = new Map<string, Bot>();
let mover: ReturnType<typeof setInterval> | undefined;

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

function randomIn({ north, south, east, west }: Bounds): Coordinates {
  return { latitude: south + Math.random() * (north - south), longitude: west + Math.random() * (east - west) };
}

function report(name: string, bot: Bot) {
  updatePosition(bot.token, { ...bot.position, accuracyMeters: 5 }).catch((err: unknown) => {
    // The game is gone (or the data was reset): stop moving this bot.
    if (err instanceof ApiError && err.status === 401) bots.delete(name);
  });
}

function addBot(name: string, bot: Bot) {
  bots.set(name, bot);
  report(name, bot);
  mover ??= setInterval(() => {
    for (const [botName, b] of bots) {
      const { north, south, east, west } = b.bounds;
      // Keep going roughly the same way, turning a little each step.
      b.heading += (Math.random() - 0.5) * 2 * MAX_TURN;
      const latitude = b.position.latitude + Math.sin(b.heading) * STEP_DEGREES;
      const longitude = b.position.longitude + Math.cos(b.heading) * STEP_DEGREES;
      // Turn around at the edge of the playground.
      if (latitude < south || latitude > north || longitude < west || longitude > east) b.heading += Math.PI;
      b.position = { latitude: clamp(latitude, south, north), longitude: clamp(longitude, west, east) };
      report(botName, b);
    }
    if (bots.size === 0) {
      clearInterval(mover);
      mover = undefined;
    }
  }, MOVE_INTERVAL_MS);
}

/**
 * Development only: adds another player to the game from this device, so a game can be tested alone.
 * The bot reports a position inside the grid right away (a runner at its route start) and then walks
 * on a wandering course every 5 seconds, so reveals have something to show. It never polls the game.
 */
export default function DebugAddPlayer({ codes }: { codes: JoinCodes }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('Bot');
  const [team, setTeam] = useState<'runner' | 'hunter'>('hunter');
  const [hunterTeam, setHunterTeamName] = useState('Bots');
  const [ready, setReadyState] = useState(true);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  if (!__DEV__) return null;

  const add = async () => {
    setBusy(true);
    setMessage('');
    try {
      const { playerToken, state } = await joinGame(codes[team], name);
      if (team === 'hunter') await setHunterTeam(playerToken, hunterTeam);
      if (ready) await setReady(playerToken, true);
      const { bounds } = state.playground;
      addBot(name.trim(), { token: playerToken, bounds, position: state.route?.start.coordinates ?? randomIn(bounds), heading: Math.random() * 2 * Math.PI });
      setMessage(`Added ${name.trim()}; it reports a position every 5 seconds.`);
      setName(nextName(name));
    } catch (err) {
      setMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Button label="Debug: add player" variant="secondary" onPress={() => setOpen(true)} />
      <Modal visible={open} transparent animationType="fade" onRequestClose={() => setOpen(false)}>
        <SafeAreaProvider>
          <Pressable style={{ flex: 1, backgroundColor: 'rgba(0,0,0,0.6)', justifyContent: 'center', padding: 20 }} onPress={() => setOpen(false)}>
            <Pressable style={styles.card} onPress={() => {}}>
              <Text style={styles.heading}>Debug: add player</Text>
              <TextInput value={name} onChangeText={setName} maxLength={MAX_NAME_LENGTH} autoCorrect={false} placeholder="Name" placeholderTextColor={colors.placeholder} style={styles.input} />
              <View style={styles.row}>
                <Button label="Runner" variant={team === 'runner' ? 'primary' : 'secondary'} onPress={() => setTeam('runner')} />
                <Button label="Hunter" variant={team === 'hunter' ? 'primary' : 'secondary'} onPress={() => setTeam('hunter')} />
              </View>
              {team === 'hunter' && <TextInput value={hunterTeam} onChangeText={setHunterTeamName} maxLength={MAX_NAME_LENGTH} autoCorrect={false} placeholder="Hunter team" placeholderTextColor={colors.placeholder} style={styles.input} />}
              <Button label={ready ? '✓ Ready after joining' : '○ Not ready after joining'} variant="secondary" onPress={() => setReadyState(!ready)} />
              {!!message && <Text style={styles.text}>{message}</Text>}
              <View style={styles.row}>
                <Button label="Add" onPress={() => void add()} disabled={busy || !isValidPlayerName(name) || (team === 'hunter' && !hunterTeam.trim())} />
                <Button label="Close" variant="secondary" onPress={() => setOpen(false)} />
              </View>
            </Pressable>
          </Pressable>
        </SafeAreaProvider>
      </Modal>
    </>
  );
}

/** "Bot" -> "Bot 2" -> "Bot 3", so names stay unique per game. */
function nextName(name: string): string {
  const match = /^(.*?)(?: (\d+))?$/.exec(name.trim());
  return `${match?.[1] ?? name} ${Number(match?.[2] ?? 1) + 1}`.slice(0, MAX_NAME_LENGTH);
}
