import { Pressable, Text, View } from 'react-native';

import type { JokerSettings } from '@/api/types';
import { useT } from '@/i18n';
import { HUNTER_JOKER_KINDS, JOKER_ALLOWANCES, JOKER_TITLE_KEYS, RUNNER_JOKER_KINDS, type JokerKind } from '@/joker-settings';
import { picker, styles } from '@/styles';

/** Card for choosing how often each joker may be used in the new game. */
export default function JokerSettingsCard({ settings, onChange, disabled }: {
  settings: JokerSettings;
  onChange: (kind: JokerKind, allowance: number) => void;
  disabled: boolean;
}) {
  const t = useT();
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('create.jokers')}</Text>
      <Text style={styles.hint}>{t('create.jokersHint')}</Text>
      <Text style={[styles.text, styles.accentText]}>{t('create.hunterJokers')}</Text>
      {HUNTER_JOKER_KINDS.map((kind) => <AllowanceRow key={kind} kind={kind} settings={settings} onChange={onChange} disabled={disabled} />)}
      <Text style={[styles.text, styles.accentText]}>{t('create.runnerJokers')}</Text>
      {RUNNER_JOKER_KINDS.map((kind) => <AllowanceRow key={kind} kind={kind} settings={settings} onChange={onChange} disabled={disabled} />)}
    </View>
  );
}

function AllowanceRow({ kind, settings, onChange, disabled }: {
  kind: JokerKind;
  settings: JokerSettings;
  onChange: (kind: JokerKind, allowance: number) => void;
  disabled: boolean;
}) {
  const t = useT();
  const joker = t(JOKER_TITLE_KEYS[kind]);
  return (
    <View style={styles.gapTiny}>
      <Text style={styles.text}>{joker}</Text>
      <View style={picker.row}>
        {JOKER_ALLOWANCES.map((allowance) => {
          const selected = settings[kind].allowance === allowance;
          return (
            <Pressable
              key={allowance}
              accessibilityRole="button"
              accessibilityLabel={t('create.jokerAllowance', { joker, allowance })}
              accessibilityState={{ selected, disabled }}
              disabled={disabled}
              onPress={() => onChange(kind, allowance)}
              style={[picker.cell, selected && picker.selected]}>
              <Text style={[picker.label, selected && picker.selectedLabel]}>{allowance}</Text>
            </Pressable>
          );
        })}
      </View>
    </View>
  );
}
