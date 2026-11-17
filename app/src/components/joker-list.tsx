import { Pressable, ScrollView, Text, View } from 'react-native';

import type { JokerKind, JokerResponse } from '@/api/client';
import type { GameStateResponse } from '@/api/types';
import Button from '@/components/button';
import { getLanguage, translate, useT } from '@/i18n';
import { picker, styles } from '@/styles';
import { useJokerUse } from '@/use-joker-use';

const TITLE_KEYS = { showRunners: 'joker.showRunners.title', queryZone: 'joker.queryZone.title', zoneOnly: 'joker.zoneOnly.title' } as const;
const TEXT_KEYS = { showRunners: 'joker.showRunners.text', queryZone: 'joker.queryZone.text', zoneOnly: 'joker.zoneOnly.text' } as const;
const COLUMNS = ['A', 'B', 'C', 'D', 'E', 'F'];
const ROWS = ['1', '2', '3', '4', '5', '6'];

/** The jokers of the player's team (a hunter team, or all runners), each with a two-step button to use it. */
export default function JokerList({ state, error, onUse }: {
  state: GameStateResponse;
  error: string;
  onUse: (kind: JokerKind, cell?: string) => Promise<void>;
}) {
  const t = useT();
  const { confirming, cell, setCell, busy, use, cancel, start } = useJokerUse(onUse);
  const confirmKey = state.team === 'runner' ? 'joker.confirmRunner' : 'joker.confirm';

  return (
    <ScrollView contentContainerStyle={[styles.content, { gap: 12 }]}>
      <Text style={styles.heading}>{t('joker.title')}</Text>
      {state.jokers.length === 0 && <Text style={styles.hint}>{t('joker.none')}</Text>}
      {state.jokers.map((joker) => (
        <View key={joker.kind} style={styles.card}>
          <Text style={styles.heading}>{t(TITLE_KEYS[joker.kind])}</Text>
          <Text style={styles.text}>{t(TEXT_KEYS[joker.kind])}</Text>
          {joker.usedAt
            ? <Used joker={joker} state={state} />
            : joker.available
              ? confirming === joker.kind
                ? <>
                  {joker.kind === 'queryZone' && <CellPicker selected={cell} onSelect={setCell} disabled={busy} />}
                  <Text style={[styles.text, styles.error]}>
                    {joker.kind === 'queryZone' && cell ? t('joker.confirmCell', { cell }) : t(confirmKey)}
                  </Text>
                  <View style={styles.row}>
                    <Button label={t('joker.confirmYes')} disabled={busy || (joker.kind === 'queryZone' && !cell)} onPress={() => use(joker.kind)} />
                    <Button label={t('joker.cancel')} variant="secondary" disabled={busy} onPress={cancel} />
                  </View>
                </>
                : <Button label={t('joker.use')} onPress={() => start(joker.kind)} />
              : <Text style={styles.hint}>{t('joker.waitForStart')}</Text>}
        </View>
      ))}
      {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
    </ScrollView>
  );
}

/** The 6 by 6 grid as buttons, laid out like the map: A–F west to east, 1–6 north to south. */
function CellPicker({ selected, onSelect, disabled }: { selected: string | null; onSelect: (cell: string) => void; disabled: boolean }) {
  const t = useT();
  return (
    <View style={picker.list}>
      <Text style={styles.hint}>{t('joker.chooseCell')}</Text>
      {ROWS.map((row) => (
        <View key={row} style={picker.row}>
          {COLUMNS.map((column) => {
            const cell = column + row;
            const active = cell === selected;
            return (
              <Pressable
                key={cell}
                accessibilityRole="button"
                accessibilityLabel={t('joker.cell', { cell })}
                accessibilityState={{ selected: active, disabled }}
                disabled={disabled}
                onPress={() => onSelect(cell)}
                style={[picker.cell, active && picker.selected]}>
                <Text style={[picker.label, active && picker.selectedLabel]}>{cell}</Text>
              </Pressable>
            );
          })}
        </View>
      ))}
    </View>
  );
}

const time = (iso: string) => new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: '2-digit' });

function Used({ joker, state }: { joker: JokerResponse; state: GameStateResponse }) {
  const t = useT();
  const positions = joker.positions ?? [];
  const usedAt = time(joker.usedAt ?? '');
  if (joker.kind === 'zoneOnly') {
    const name = state.players.find((player) => player.playerId === joker.usedBy)?.name ?? '';
    return (
      <View style={styles.gapTiny}>
        <Text style={[styles.text, styles.accentText]}>{t('joker.usedBy', { name, time: usedAt })}</Text>
        <Text style={styles.hint}>{t('joker.zoneOnly.used', { name })}</Text>
      </View>
    );
  }
  return (
    <View style={styles.gapTiny}>
      <Text style={[styles.text, styles.accentText]}>{translate('joker.usedAt', { time: usedAt })}</Text>
      {joker.kind === 'queryZone'
        ? !!joker.cell && <Text style={styles.text}>{t(joker.found ? 'joker.zoneFound' : 'joker.zoneEmpty', { cell: joker.cell })}</Text>
        : <>
          {positions.length === 0 && <Text style={styles.hint}>{t('joker.noPositions')}</Text>}
          {positions.map((p) => (
            <Text key={p.playerId} style={styles.hint}>
              {state.players.find((player) => player.playerId === p.playerId)?.name} · {t('marker.accuracy', { meters: Math.round(p.accuracyMeters), time: time(p.recordedAt) })}
            </Text>
          ))}
        </>}
    </View>
  );
}

