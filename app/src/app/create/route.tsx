import { ActivityIndicator } from 'react-native';

import Button from '@/components/button';
import CreateMessage from '@/components/create-message';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import RouteChoiceCard from '@/components/route-choice-card';
import { useCreateGameContext } from '@/create-game-context';
import { useT } from '@/i18n';
import { colors } from '@/styles';

/** Step 3: where the runners start and stop; then the game is created. */
export default function CreateRouteScreen() {
  const t = useT();
  const game = useCreateGameContext();
  const { busy } = game;
  return (
    <KeyboardScrollView>
      <RouteChoiceCard mode={game.routeMode} onModeChange={game.setRouteMode} playground={game.playground}
        points={game.routePoints} tapMissed={game.routeTapMissed} onTap={game.tapRoutePoint} disabled={busy} />
      <CreateMessage />
      {busy && <ActivityIndicator color={colors.accent} />}
      <Button label={t('create.create')} onPress={game.create} disabled={busy || game.cannotCreate} disabledReason={game.createBlocker} />
    </KeyboardScrollView>
  );
}
