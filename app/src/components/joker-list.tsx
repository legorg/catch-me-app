import type { ReactNode } from 'react';
import { Pressable, ScrollView, Text, View } from 'react-native';

import type { JokerRequest, JokerResponse } from '@/api/client';
import type { Coordinates, GameStateResponse, Playground } from '@/api/types';
import Button from '@/components/button';
import { inform } from '@/confirm';
import PlaygroundMap from '@/components/playground-map';
import { getLanguage, useT } from '@/i18n';
import { isSpectator, ownPlayer } from '@/game-state';
import { JOKER_TITLE_KEYS as TITLE_KEYS } from '@/joker-settings';
import { REVEAL_JOKER_KINDS, locatedEntries, pendingRevealJokerAt, remainingUses, trapStatus, zoneAlertEntries, type JokerUse } from '@/lobby';
import { cellAt } from '@/playground';
import { picker, styles } from '@/styles';
import { useJokerUse } from '@/use-joker-use';

const TEXT_KEYS = {
  showRunners: 'joker.showRunners.text', queryZone: 'joker.queryZone.text', zoneOnly: 'joker.zoneOnly.text', fakePosition: 'joker.fakePosition.text',
  showHunters: 'joker.showHunters.text', compass: 'joker.compass.text', trap: 'joker.trap.text',
} as const;
const TRAP_STATUS_KEYS = { waiting: 'joker.trap.waiting', active: 'joker.trap.active', triggered: 'joker.trap.triggered' } as const;

/** Personal and team jokers, each with a two-step button to use it. */
export default function JokerList({ state, error, onUse }: {
  state: GameStateResponse;
  error: string;
  onUse: (joker: JokerRequest) => Promise<void>;
}) {
  const t = useT();
  const { confirming, cell, setCell, point, setPoint, busy, ready, missing, use, cancel, start } = useJokerUse(state.playground, onUse);
  const confirmKey = confirming === 'fakePosition' ? 'joker.confirmPersonal' : state.team === 'runner' ? 'joker.confirmRunner' : 'joker.confirm';
  const pointCell = point && cellAt(state.playground, point);
  // The compass starts at the hunter's own position; the server refuses it without one.
  const hasPosition = !!state.players.find((p) => p.playerId === state.playerId)?.position;
  const runnerStatus = ownPlayer(state)?.runner?.status;
  const pendingRevealAt = pendingRevealJokerAt(state);

  if (isSpectator(state) || runnerStatus === 'caught' || runnerStatus === 'finished') {
    return (
      <View style={styles.content}>
        <Text style={styles.heading}>{t('joker.title')}</Text>
        <Text style={styles.text}>{t(isSpectator(state) ? 'spectator.noJokers' : runnerStatus === 'caught' ? 'catch.noJokers' : 'finish.noJokers')}</Text>
      </View>
    );
  }

  return (
    <ScrollView contentContainerStyle={[styles.content, { gap: 12 }]}>
      <Text style={styles.heading}>{t('joker.title')}</Text>
      {state.jokers.length === 0 && <Text style={styles.hint}>{t('joker.none')}</Text>}
      {state.jokers.map((joker) => {
        const allowance = state.jokerSettings[joker.kind].allowance;
        const remaining = remainingUses(state, joker);
        return (
          <View key={joker.kind} style={styles.card}>
            <Text style={styles.heading}>{t(TITLE_KEYS[joker.kind])}</Text>
            <Text style={styles.text}>{t(TEXT_KEYS[joker.kind])}</Text>
            <Text style={styles.hint}>{allowance === 0 ? t('joker.disabled') : t('joker.remaining', { remaining, allowance })}</Text>
            <Uses joker={joker} state={state} />
            {joker.available
              ? confirming === joker.kind
                ? <>
                  {joker.kind === 'queryZone' && <CellPicker playground={state.playground} selected={cell} onSelect={setCell} disabled={busy} />}
                  {(joker.kind === 'fakePosition' || joker.kind === 'trap') && <PointPicker playground={state.playground} point={point} onPick={setPoint} />}
                  <Text style={[styles.text, styles.error]}>
                    {joker.kind === 'queryZone' && cell
                      ? t('joker.confirmCell', { cell })
                      : joker.kind === 'fakePosition' && pointCell
                        ? t('joker.confirmPoint', { cell: pointCell })
                        : joker.kind === 'trap' && pointCell ? t('joker.trap.confirmPoint', { cell: pointCell }) : t(confirmKey)}
                  </Text>
                  <View style={styles.row}>
                    <Button label={t('joker.confirmYes')} disabled={busy || !ready(joker.kind)} disabledReason={missing(joker.kind)} onPress={() => use(joker.kind)} />
                    <Button label={t('joker.cancel')} variant="secondary" disabled={busy} onPress={cancel} />
                  </View>
                </>
                : joker.kind === 'compass' && !hasPosition
                  ? <Text style={styles.hint}>{t('joker.compass.needsPosition')}</Text>
                  : <Button label={t('joker.use')} onPress={() => pendingRevealAt && REVEAL_JOKER_KINDS.includes(joker.kind)
                    ? inform(t('joker.pending.title'), t('joker.pending.text', { time: time(pendingRevealAt) }))
                    : start(joker.kind)} />
              : remaining > 0 && <Text style={styles.hint}>{t('joker.waitForStart')}</Text>}
          </View>
        );
      })}
      {state.team === 'runner' && <><ZoneAlertCard state={state} /><LocatedCard state={state} /></>}
      {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
    </ScrollView>
  );
}

/** The passive zone alert: whether it is on in this game, and the runner's own alerts. */
function ZoneAlertCard({ state }: { state: GameStateResponse }) {
  const t = useT();
  const alerts = zoneAlertEntries(state);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('joker.zoneAlert.title')}</Text>
      <Text style={styles.text}>{t('joker.zoneAlert.text')}</Text>
      {!state.jokerSettings.zoneAlert.enabled
        ? <Text style={styles.hint}>{t('joker.disabled')}</Text>
        : alerts.length === 0
          ? <Text style={styles.hint}>{t('joker.zoneAlert.none')}</Text>
          : alerts.map((alert) => <Text key={alert.id} style={styles.text}>{alert.text}</Text>)}
    </View>
  );
}

/** Every hunter team's showRunners use that included the runner: which team, and when. */
function LocatedCard({ state }: { state: GameStateResponse }) {
  const t = useT();
  const located = locatedEntries(state);
  return (
    <View style={styles.card}>
      <Text style={styles.heading}>{t('joker.located.title')}</Text>
      <Text style={styles.text}>{t('joker.located.text')}</Text>
      {located.length === 0
        ? <Text style={styles.hint}>{t('joker.located.none')}</Text>
        : located.map((entry) => <Text key={entry.id} style={styles.text}>{entry.text}</Text>)}
    </View>
  );
}

/** The game's grid as buttons, laid out like the map: rows A, B, … north to south, columns 1, 2, … west to east. */
function CellPicker({ playground, selected, onSelect, disabled }: {
  playground: Playground; selected: string | null; onSelect: (cell: string) => void; disabled: boolean;
}) {
  const t = useT();
  const rows = Array.from({ length: playground.rows }, (_, i) => String.fromCharCode(65 + i));
  const columns = Array.from({ length: playground.columns }, (_, i) => String(i + 1));
  // Many narrow buttons need a smaller label to fit names like "L12".
  const small = playground.columns > 8;
  return (
    <View style={picker.list}>
      <Text style={styles.hint}>{t('joker.chooseCell')}</Text>
      {rows.map((row) => (
        <View key={row} style={[picker.row, small && picker.rowTight]}>
          {columns.map((column) => {
            const cell = row + column;
            const active = cell === selected;
            return (
              <Pressable
                key={cell}
                accessibilityRole="button"
                accessibilityLabel={t('joker.cell', { cell })}
                accessibilityState={{ selected: active, disabled }}
                disabled={disabled}
                onPress={() => onSelect(cell)}
                style={[picker.cell, small && picker.cellTight, active && picker.selected]}>
                <Text style={[picker.label, small && picker.labelSmall, active && picker.selectedLabel]}>{cell}</Text>
              </Pressable>
            );
          })}
        </View>
      ))}
    </View>
  );
}

/** A map of the playground whose center is the picked point: the runner moves the map until the crosshair is on it. */
function PointPicker({ playground, point, onPick }: { playground: Playground; point: Coordinates | null; onPick: (point: Coordinates) => void }) {
  const t = useT();
  const inside = point && cellAt(playground, point);
  return (
    <View style={styles.gapTiny}>
      <Text style={styles.hint}>{t('joker.choosePoint')}</Text>
      <PlaygroundMap playground={playground} editable={false} onSelect={noop} onPick={onPick} />
      {point && <Text style={inside ? styles.hint : [styles.text, styles.error]}>{inside ? t('joker.pointIn', { cell: inside }) : t('joker.pointOutside')}</Text>}
    </View>
  );
}

const noop = () => {};

const time = (iso: string) => new Date(iso).toLocaleTimeString(getLanguage(), { hour: '2-digit', minute: '2-digit', second: '2-digit' });

const nameOf = (state: GameStateResponse, playerId?: string) => state.players.find((player) => player.playerId === playerId)?.name ?? '';

/** Every use of the joker the player may see, oldest first, each with what it showed. */
function Uses({ joker, state }: { joker: JokerResponse; state: GameStateResponse }) {
  const t = useT();
  switch (joker.kind) {
    case 'zoneOnly':
      return joker.uses.map((use, index) => (
        <Used key={index} heading={t('joker.usedBy', { name: nameOf(state, use.usedBy), time: time(use.usedAt) })}>
          <Text style={styles.hint}>{t('joker.zoneOnly.used', { name: nameOf(state, use.usedBy) })}</Text>
        </Used>
      ));
    case 'fakePosition':
      return joker.uses.map((use, index) => (
        <Used key={index} heading={t('joker.usedBy', { name: nameOf(state, use.usedBy), time: time(use.usedAt) })}>
          <Text style={styles.hint}>{t('joker.fakePosition.used', { name: nameOf(state, use.usedBy), cell: cellAt(state.playground, use.point) ?? '' })}</Text>
        </Used>
      ));
    case 'queryZone':
      return joker.uses.map((use, index) => (
        <Used key={index} heading={t('joker.usedAt', { time: time(use.usedAt) })}>
          <Text style={styles.text}>{t(use.found ? 'joker.zoneFound' : 'joker.zoneEmpty', { cell: use.cell })}</Text>
        </Used>
      ));
    case 'compass':
      return joker.uses.map((use, index) => (
        <Used key={index} heading={t('joker.usedAt', { time: time(use.usedAt) })}>
          <Text style={styles.hint}>{use.bearings.length > 0 ? t('joker.compass.used') : t('joker.noPositions')}</Text>
        </Used>
      ));
    case 'trap':
      return joker.uses.map((use, index) => {
        // As of the last game state; the map's circle follows the clock between polls.
        const status = trapStatus(use, Date.parse(state.serverTime));
        return (
          <Used key={index} heading={t('joker.usedAt', { time: time(use.usedAt) })}>
            <Text style={status === 'triggered' ? [styles.text, styles.error] : styles.hint}>
              {t(TRAP_STATUS_KEYS[status], { time: time(status === 'triggered' && use.triggeredAt ? use.triggeredAt : use.activeAt) })}
            </Text>
          </Used>
        );
      });
    case 'showRunners':
      return joker.uses.map((use, index) => (
        <Used key={index} heading={t('joker.usedAt', { time: time(use.usedAt) })}>
          <Positions positions={use.positions} state={state} empty={t('joker.noPositions')} />
        </Used>
      ));
    case 'showHunters':
      return joker.uses.map((use, index) => (
        <Used key={index} heading={t('joker.usedBy', { name: nameOf(state, use.usedBy), time: time(use.usedAt) })}>
          <Positions positions={use.positions} state={state} empty={t('joker.showHunters.noPositions')} />
        </Used>
      ));
    default:
      return null;
  }
}

function Used({ heading, children }: { heading: string; children: ReactNode }) {
  return (
    <View style={styles.gapTiny}>
      <Text style={[styles.text, styles.accentText]}>{heading}</Text>
      {children}
    </View>
  );
}

function Positions({ positions, state, empty }: { positions: JokerUse<'showRunners'>['positions']; state: GameStateResponse; empty: string }) {
  const t = useT();
  if (positions.length === 0) return <Text style={styles.hint}>{empty}</Text>;
  return positions.map((p) => (
    <Text key={p.playerId} style={styles.hint}>
      {nameOf(state, p.playerId)} · {t('marker.accuracy', { meters: Math.round(p.accuracyMeters), time: time(p.recordedAt) })}
    </Text>
  ));
}
