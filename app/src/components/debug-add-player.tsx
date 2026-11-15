import { useState } from 'react';
import { Modal, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { MAX_NAME_LENGTH, isValidPlayerName, joinGame, setHunterTeam, setReady, type JoinCodes } from '@/api/client';
import Button from '@/components/button';
import { colors, styles } from '@/styles';

/**
 * Development only: adds another player to the game from this device, so a game can be tested alone.
 * The bot's token is dropped right away; it never polls or shares a position.
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
      const { playerToken } = await joinGame(codes[team], name);
      if (team === 'hunter') await setHunterTeam(playerToken, hunterTeam);
      if (ready) await setReady(playerToken, true);
      setMessage(`Added ${name.trim()}.`);
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
