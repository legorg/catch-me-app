import { ActivityIndicator } from 'react-native';

import CreateMessage from '@/components/create-message';
import JoinAsCard from '@/components/join-as-card';
import JoinCodes from '@/components/join-codes';
import MapScrollView from '@/components/map-scroll-view';
import { useCreateGameContext } from '@/create-game-context';
import { noop } from '@/press';
import { colors } from '@/styles';

/** After creating: the game's grid, its codes, and joining it yourself. */
export default function CreatedScreen() {
  const game = useCreateGameContext();
  const { created, busy } = game;
  if (!created) return null;
  return (
    <MapScrollView map={{ playground: created.game.playground, editable: false, onSelect: noop, alignToGrid: true }}>
      <CreateMessage />
      {busy && <ActivityIndicator color={colors.accent} />}
      <JoinCodes codes={created.joinCodes} />
      <JoinAsCard codes={created.joinCodes} name={game.name} onNameChange={game.setName} busy={busy} onJoin={game.joinAs} />
    </MapScrollView>
  );
}
