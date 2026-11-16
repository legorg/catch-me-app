import { ActivityIndicator, Text, View } from 'react-native';

import { isGameStateRunnerResponse } from '@/api/types';
import DebugAddPlayer from '@/components/debug-add-player';
import FullscreenMap from '@/components/fullscreen-map';
import HunterTeamCard from '@/components/hunter-team-card';
import JoinCodes from '@/components/join-codes';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import LobbyActions from '@/components/lobby-actions';
import LobbyCountdownModal from '@/components/lobby-countdown-modal';
import LobbyRouteCard from '@/components/lobby-route-card';
import LobbyTeamCard from '@/components/lobby-team-card';
import LocationPermissionCards from '@/components/location-permission-cards';
import { translateParts, useT } from '@/i18n';
import { lobbyInfo, noop, playerMarkers, playersByTeam, routeHighlights, routeMarkers } from '@/lobby';
import { distanceMeters, START_RADIUS_METERS } from '@/playground';
import { colors, styles } from '@/styles';
import { useLobby } from '@/use-lobby';

export default function LobbyScreen() {
  const t = useT();
  const lobby = useLobby();
  const { token, snapshot, state, error, countdown, busy, me, location, sharing } = lobby;

  if (!token || !state || !snapshot) {
    // A ScrollView must exist from the first render, or the large header never links to it.
    return (
      <>
        <KeyboardScrollView>
          {error ? <Text style={[styles.text, styles.error]}>{error}</Text> : <ActivityIndicator color={colors.accent} />}
        </KeyboardScrollView>
      </>
    );
  }

  const skipStartPositionCheck = state.startPositionCheckDisabled === true;
  const teams = playersByTeam(state.players);
  const isRunner = isGameStateRunnerResponse(state);
  const route = isRunner ? state.route : undefined;
  // The server only sends the route to runners.
  const metersToStart = route && me?.position ? distanceMeters(me.position, route.start.coordinates) : null;
  const atStart = metersToStart !== null && metersToStart <= START_RADIUS_METERS;
  const highlights = route ? routeHighlights(state.playground, route) : [];
  const markers = [
    ...(route ? routeMarkers(route) : []),
    ...playerMarkers(state).filter((marker) => marker.id !== state.playerId),
  ];
  const readyBlocked = lobby.needsHunterTeam || lobby.needsBackground || lobby.needsForeground;
  const readyDisabled = busy
    || (!me?.ready && readyBlocked)
    || (!skipStartPositionCheck && !me?.ready && isRunner && !atStart);

  return (
    <>
      <LobbyCountdownModal countdown={countdown} />
      <KeyboardScrollView>
        {skipStartPositionCheck && <View style={styles.card}>
          <Text style={[styles.heading, styles.error]}>{t('lobby.testMode')}</Text>
          <Text style={styles.text}>{t('lobby.testModeText')}</Text>
        </View>}
        <Text accessibilityLiveRegion="polite" style={styles.text}>
          {lobbyInfo(countdown !== null, skipStartPositionCheck)}
        </Text>
        <View style={styles.mapContainer}>
          <FullscreenMap playground={state.playground} editable={false} onSelect={noop} highlights={highlights} markers={markers} />
        </View>
        {state.team === 'hunter' && <Text style={styles.hint}>
          {t('lobby.legendHunter')}
        </Text>}
        <LocationPermissionCards
          location={location}
          team={state.team}
          skipStartPositionCheck={skipStartPositionCheck}
          ready={!!me?.ready}
          sharing={sharing}
          needsForeground={lobby.needsForeground}
          needsBackground={lobby.needsBackground}
          onRequestLocation={lobby.requestLocation}
          onRequestBackground={lobby.requestBackground}
        />
        <View style={styles.card}>
          <Text style={styles.text}>
            {translateParts('lobby.youAre', {
              team: <Text key="team" style={styles.heading}>{t(state.team === 'runner' ? 'team.runner' : 'team.hunter')}</Text>,
            })}
            {me ? ` · ${me.name}` : ''}{me?.hunterTeam ? ` · ${t('lobby.hunterTeam', { name: me.hunterTeam })}` : ''}
          </Text>
        </View>
        <JoinCodes codes={state.joinCodes} />
        {__DEV__ && <DebugAddPlayer codes={state.joinCodes} />}
        {route && <LobbyRouteCard route={route} skipStartPositionCheck={skipStartPositionCheck} atStart={atStart} metersToStart={metersToStart} />}
        {state.team === 'hunter' && me && <HunterTeamCard
          me={me}
          players={state.players}
          needsHunterTeam={lobby.needsHunterTeam}
          busy={busy}
          newTeam={lobby.newTeam}
          onNewTeamChange={lobby.setNewTeam}
          onJoin={lobby.changeHunterTeam}
          onCreate={lobby.createTeam}
        />}
        <LobbyTeamCard team="runner" players={teams.runner} playerId={state.playerId} />
        <LobbyTeamCard team="hunter" players={teams.hunter} playerId={state.playerId} />
        {!!error && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{error}</Text>}
        {me && <LobbyActions ready={me.ready} busy={busy} readyDisabled={readyDisabled} onToggleReady={lobby.toggleReady} onLeave={lobby.leave} />}
      </KeyboardScrollView>
    </>
  );
}
