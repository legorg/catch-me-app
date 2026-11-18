import { ScrollView, Text } from 'react-native';

import { isGameStateRunnerResponse, type GameStateResponse, type PlayerResponse } from '@/api/types';
import Button from '@/components/button';
import LocationSharingInfo from '@/components/location-sharing-info';
import { translateParts, useT } from '@/i18n';
import { routeTitles } from '@/lobby';
import { isActivePlayer, isSpectator } from '@/game-state';
import { overlay, styles } from '@/styles';
import { useConfirmExit } from '@/use-confirm-exit';
import type { LocationSharing } from '@/use-location-sharing';

// Team-specific texts, so no branching on the team is needed in the markup.
const TEAM_TEXTS = {
  runner: { team: 'team.runner', reveal: 'lobby.revealRunner', legend: 'lobby.legendRunner' },
  hunter: { team: 'team.hunter', reveal: 'lobby.revealHunter', legend: 'lobby.legendHunter' },
} as const;

function RouteLine({ state }: { state: GameStateResponse }) {
  if (!isGameStateRunnerResponse(state)) return null;
  const titles = routeTitles(state.route);
  return (
    <Text style={styles.text}>
      {translateParts('lobby.route', {
        start: <Text key="start" style={[styles.heading, styles.startText]}>{titles.start}</Text>,
        stop: <Text key="stop" style={[styles.heading, styles.targetText]}>{titles.stop}</Text>,
        distance: (state.route.distanceMeters / 1000).toFixed(1),
      })}
    </Text>
  );
}

type InfoPanelProps = {
  state: GameStateResponse;
  me: PlayerResponse | undefined;
  /** Whether a reveal is scheduled, which the panel explains. */
  revealScheduled: boolean;
  location: LocationSharing;
};

/** Outcomes are visible even when there is no retained position for a marker. */
function RunnerOutcomes({ state }: { state: GameStateResponse }) {
  const t = useT();
  const caught = state.players.filter((player) => player.runner?.status === 'caught').map((player) => player.name).join(', ');
  const finished = state.players.filter((player) => player.runner?.status === 'finished').map((player) => player.name).join(', ');
  return <>
    {!!caught && <Text style={styles.text}>{t('catch.players', { names: caught })}</Text>}
    {!!finished && <Text style={styles.text}>{t('finish.players', { names: finished })}</Text>}
  </>;
}

/** Collapsible card over the map: who you are, the route, the legend, location sharing and the exit button. */
export default function InfoPanel({ state, me, revealScheduled, location }: InfoPanelProps) {
  const t = useT();
  const texts = TEAM_TEXTS[state.team];
  const spectator = isSpectator(state);
  const confirmExit = useConfirmExit();
  return (
    <ScrollView style={overlay.info} contentContainerStyle={styles.gapMedium}>
      <Text style={styles.text}>
        {translateParts('lobby.youAre', {
          team: <Text key="team" style={styles.heading}>{t(spectator ? 'spectator.title' : texts.team)}</Text>,
        })}
        {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
      </Text>
      {!spectator && revealScheduled && isActivePlayer(me) && <Text style={styles.hint}>{t(texts.reveal, { minutes: state.revealIntervalMinutes })}</Text>}
      {!spectator && <RouteLine state={state} />}
      <Text style={styles.hint}>{t(spectator ? 'spectator.info' : texts.legend)}</Text>
      {spectator && <Text style={styles.hint}>{t('spectator.fairPlay')}</Text>}
      <RunnerOutcomes state={state} />
      {spectator ? <Text style={styles.text}>{t('spectator.noTracking')}</Text>
        : me?.runner?.status === 'caught' || me?.runner?.status === 'finished'
        ? <Text style={styles.text}>{t(me.runner.status === 'caught' ? 'catch.sharingStopped' : 'finish.sharingStopped')}</Text>
        : isActivePlayer(me) && <LocationSharingInfo location={location} />}
      {/* Rarely needed and final, so it sits here behind a confirmation. */}
      <Button label={t('game.exit')} variant="secondary" onPress={confirmExit} />
    </ScrollView>
  );
}
