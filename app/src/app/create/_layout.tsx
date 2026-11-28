import { Stack } from 'expo-router';

import { CreateGameContext } from '@/create-game-context';
import { useT } from '@/i18n';
import { stackScreenOptions } from '@/stack-options';
import { useCreateGame } from '@/use-create-game';

/** Creating a game, one step per screen; the draft lives here, so going back and forth keeps it. */
export default function CreateLayout() {
  const t = useT();
  const game = useCreateGame();
  return (
    <CreateGameContext value={game}>
      <Stack screenOptions={stackScreenOptions}>
        <Stack.Screen name="index" options={{ title: t('create.chooseTitle') }} />
        <Stack.Screen name="rules" options={{ title: t('create.rulesTitle') }} />
        <Stack.Screen name="route" options={{ title: t('create.routeTitle') }} />
        <Stack.Screen name="created" options={{ title: t('create.createdTitle') }} />
      </Stack>
    </CreateGameContext>
  );
}
