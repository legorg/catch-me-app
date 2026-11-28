import { createContext, use } from 'react';

import type { useCreateGame } from './use-create-game';

type CreateGame = ReturnType<typeof useCreateGame>;

/** The draft game, shared by the create steps; provided by app/create/_layout.tsx. */
export const CreateGameContext = createContext<CreateGame | null>(null);

export function useCreateGameContext(): CreateGame {
  const game = use(CreateGameContext);
  if (!game) throw new Error('useCreateGameContext is only available inside the create steps');
  return game;
}
