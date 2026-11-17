import { ScrollView, Text } from 'react-native';

import { isGameStateRunnerResponse, type GameStateResponse, type PlayerResponse } from '@/api/types';
import Button from '@/components/button';
import LocationSharingInfo from '@/components/location-sharing-info';
import { translateParts, useT } from '@/i18n';
import { routeTitles } from '@/lobby';
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

/** Collapsible card over the map: who you are, the route, the legend, location sharing and the exit button. */
export default function InfoPanel({ state, me, revealScheduled, location }: InfoPanelProps) {
  const t = useT();
  const texts = TEAM_TEXTS[state.team];
  const confirmExit = useConfirmExit();
  return (
    <ScrollView style={overlay.info} contentContainerStyle={styles.gapMedium}>
      <Text style={styles.text}>
        {translateParts('lobby.youAre', {
          team: <Text key="team" style={styles.heading}>{t(texts.team)}</Text>,
        })}
        {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
      </Text>
      {revealScheduled && <Text style={styles.hint}>{t(texts.reveal, { minutes: state.revealIntervalMinutes })}</Text>}
      <RouteLine state={state} />
      <Text style={styles.hint}>{t(texts.legend)}</Text>
      <LocationSharingInfo location={location} />
      {/* Rarely needed and final, so it sits here behind a confirmation. */}
      <Button label={t('game.exit')} variant="secondary" onPress={confirmExit} />
    </ScrollView>
  );
}
