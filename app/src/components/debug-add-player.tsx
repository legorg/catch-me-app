import { Modal, Pressable, Text, TextInput, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { MAX_NAME_LENGTH, isValidPlayerName, type JoinCodes } from '@/api/client';
import Button from '@/components/button';
import { noop } from '@/press';
import { colors, styles } from '@/styles';
import { useDebugAddPlayer } from '@/use-debug-add-player';

/**
 * Development only: adds another player to the game from this device, so a game can be tested alone.
 * The bot reports a position inside the grid right away (a runner at its route start) and then walks
 * on a wandering course every 5 seconds, so reveals have something to show. It never polls the game.
 */
export default function DebugAddPlayer({ codes }: { codes: JoinCodes }) {
  const { open, name, team, hunterTeam, ready, busy, message, setName, setTeam, setHunterTeamName, show, close, toggleReady, add } = useDebugAddPlayer(codes);
  if (!__DEV__) return null;

  return (
    <>
      <Button label="Debug: add player" variant="secondary" onPress={show} />
      <Modal visible={open} transparent animationType="fade" onRequestClose={close}>
        <SafeAreaProvider>
          <Pressable style={styles.modalBackdrop} onPress={close}>
            <Pressable style={styles.card} onPress={noop}>
              <Text style={styles.heading}>Debug: add player</Text>
              <TextInput value={name} onChangeText={setName} maxLength={MAX_NAME_LENGTH} autoCorrect={false} placeholder="Name" placeholderTextColor={colors.placeholder} style={styles.input} />
              <View style={styles.row}>
                <Button label="Runner" variant={team === 'runner' ? 'primary' : 'secondary'} onPress={() => setTeam('runner')} />
                <Button label="Hunter" variant={team === 'hunter' ? 'primary' : 'secondary'} onPress={() => setTeam('hunter')} />
              </View>
              {team === 'hunter' && <TextInput value={hunterTeam} onChangeText={setHunterTeamName} maxLength={MAX_NAME_LENGTH} autoCorrect={false} placeholder="Hunter team" placeholderTextColor={colors.placeholder} style={styles.input} />}
              <Button label={ready ? '✓ Ready after joining' : '○ Not ready after joining'} variant="secondary" onPress={toggleReady} />
              {!!message && <Text style={styles.text}>{message}</Text>}
              <View style={styles.row}>
                <Button label="Add" onPress={add} disabled={busy || !isValidPlayerName(name) || (team === 'hunter' && !hunterTeam.trim())} />
                <Button label="Close" variant="secondary" onPress={close} />
              </View>
            </Pressable>
          </Pressable>
        </SafeAreaProvider>
      </Modal>
    </>
  );
}
