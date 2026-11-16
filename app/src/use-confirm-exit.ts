import { confirm } from '@/confirm';
import { useT } from '@/i18n';
import { goHome } from '@/use-game-state';

/** Asks before leaving a running game, then goes back to the start screen. */
export function useConfirmExit() {
  const t = useT();
  return () => confirm(t('game.exitTitle'), t('game.exitText'), t('game.exitYes'), goHome);
}
