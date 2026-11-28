import { router } from 'expo-router';

import Button from '@/components/button';
import JokerSettingsCard from '@/components/joker-settings-card';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import SettingCard from '@/components/setting-card';
import { useCreateGameContext } from '@/create-game-context';
import { useT } from '@/i18n';

/** Step 2: how often runners are revealed and which jokers the hunters get. */
export default function CreateRulesScreen() {
  const t = useT();
  const game = useCreateGameContext();
  return (
    <KeyboardScrollView>
      <SettingCard title={t('create.revealInterval')} label={t('create.revealIntervalLabel')} hint={t('create.revealIntervalHint')}
        value={game.revealIntervalText} onChange={game.setRevealIntervalText} editable={!game.busy} error={game.revealIntervalError} maxLength={2} />
      <JokerSettingsCard settings={game.jokers} onChange={game.setJokerAllowance} onZoneAlertChange={game.setZoneAlert} disabled={game.busy} />
      <Button label={t('create.next')} onPress={() => router.push('/create/route')} disabled={game.rulesBlocker !== null} disabledReason={game.rulesBlocker} />
    </KeyboardScrollView>
  );
}
