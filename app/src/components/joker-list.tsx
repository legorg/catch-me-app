import { useState } from 'react';
import { ScrollView, Text, View } from 'react-native';

import type { GameStateResponse, JokerKind, JokerResponse } from '@/api/client';
import Button from '@/components/button';
import { getLanguage, translate, useT } from '@/i18n';
import { colors, styles } from '@/styles';

const TITLE_KEYS = { showRunners: 'joker.showRunners.title' } as const;
const TEXT_KEYS = { showRunners: 'joker.showRunners.text' } as const;

/** The jokers of the player's team, each with a two-step button to use it. */
export default function JokerList({ state, error, onUse }: {
  state: GameStateResponse;
  error: string;
  onUse: (kind: JokerKind) => Promise<void>;
}) {
  const t = useT();
  const [confirming, setConfirming] = useState<JokerKind | null>(null);
  const [busy, setBusy] = useState(false);
  // queryZone needs a cell picker, which comes with the app step; until then only the known jokers.
  const jokers = state.jokers.filter((j): j is JokerResponse & { kind: JokerKind } => j.kind === 'showRunners');

  const use = async (kind: JokerKind) => {
    if (busy) return;
    setBusy(true);
    try {
      await onUse(kind);
    } finally {
      setBusy(false);
      setConfirming(null);
    }
  };

  return (
    <ScrollView contentContainerStyle={[styles.content, { gap: 12 }]}>
      <Text style={styles.heading}>{t('joker.title')}</Text>
      {jokers.length === 0 && <Text style={styles.hint}>{t('joker.none')}</Text>}
      {jokers.map((joker) => (
        <View key={joker.kind} style={styles.card}>
          <Text style={styles.heading}>{t(TITLE_KEYS[joker.kind])}</Text>
          <Text style={styles.text}>{t(TEXT_KEYS[joker.kind])}</Text>
          {joker.usedAt
            ? <Used joker={joker} state={state} />
            : joker.available
              ? confirming === joker.kind
                ? <>
                  <Text style={[styles.text, styles.error]}>{t('joker.confirm')}</Text>
                  <View style={styles.row}>
                    <Button label={t('joker.confirmYes')} disabled={busy} onPress={() => void use(joker.kind)} />
                    <Button label={t('joker.cancel')} variant="secondary" disabled={busy} onPress={() => setConfirming(null)} />
                  </View>
                </>
                : <Button label={t('joker.use')} onPress={() => setConfirming(joker.kind)} />
              : <Text style={styles.hint}>{t('joker.waitForStart')}</Text>}
        </View>
      ))}
      {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
    </ScrollView>
  );
}

function Used({ joker, state }: { joker: JokerResponse; state: GameStateResponse }) {
  const t = useT();
  const time = (iso: string) => new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  const positions = joker.positions ?? [];
  return (
    <View style={{ gap: 4 }}>
      <Text style={[styles.text, { color: colors.accent }]}>{translate('joker.usedAt', { time: time(joker.usedAt ?? '') })}</Text>
      {positions.length === 0 && <Text style={styles.hint}>{t('joker.noPositions')}</Text>}
      {positions.map((p) => (
        <Text key={p.playerId} style={styles.hint}>
          {state.players.find((player) => player.playerId === p.playerId)?.name} · {t('marker.accuracy', { meters: Math.round(p.accuracyMeters), time: time(p.recordedAt) })}
        </Text>
      ))}
    </View>
  );
}
