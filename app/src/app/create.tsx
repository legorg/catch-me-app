import { ActivityIndicator, Text, View } from 'react-native';

import Button from '@/components/button';
import FullscreenMap from '@/components/fullscreen-map';
import JoinAsCard from '@/components/join-as-card';
import JoinCodes from '@/components/join-codes';
import JokerSettingsCard from '@/components/joker-settings-card';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import RouteChoiceCard from '@/components/route-choice-card';
import RotationCard from '@/components/rotation-card';
import SettingCard from '@/components/setting-card';
import { formatNumber, useT } from '@/i18n';
import { colors, styles } from '@/styles';
import { useCreateGame } from '@/use-create-game';

export default function CreateScreen() {
  const t = useT();
  const game = useCreateGame();
  const { created, playground, displayedSize, displayedGridSize, busy } = game;

  return (
    <KeyboardScrollView>
      <Text style={styles.text}>
        {displayedSize !== null && displayedGridSize !== null
          ? t('create.gridSized', {
            cells: displayedGridSize,
            cell: formatNumber(displayedSize / 1000),
            total: formatNumber(displayedSize * displayedGridSize / 1000, { min: 1, max: 1 }),
          })
          : t('create.grid')}
      </Text>
      {!created && <>
        <SettingCard title={t('create.gridSize')} label={t('create.gridSizeLabel')} hint={t('create.gridSizeHint')}
          value={game.gridSizeText} onChange={game.setGridSizeText} editable={!busy} error={game.gridSizeError} maxLength={2} />
        <SettingCard title={t('create.cellSize')} label={t('create.cellSizeLabel')} hint={t('create.cellSizeHint')}
          value={game.cellSizeText} onChange={game.setCellSizeText} editable={!busy} error={game.sizeError} keyboardType="decimal-pad" maxLength={8} />
        <SettingCard title={t('create.revealInterval')} label={t('create.revealIntervalLabel')} hint={t('create.revealIntervalHint')}
          value={game.revealIntervalText} onChange={game.setRevealIntervalText} editable={!busy} error={game.revealIntervalError} maxLength={2} />
        <JokerSettingsCard settings={game.jokers} onChange={game.setJokerAllowance} onZoneAlertChange={game.setZoneAlert} disabled={busy} />
      </>}
      <View style={styles.mapContainer}>
        {/* Remounts once when the position arrives, so the map opens there. */}
        <FullscreenMap key={`${created?.game.sessionId ?? 'draft'}-${game.hasPosition ? 'here' : 'default'}`} playground={playground} editable={!created && !busy} onSelect={game.selectCenter} />
      </View>
      {!created && <RotationCard degrees={game.rotation} onChange={game.setRotation} disabled={busy} />}
      {playground && <Text style={styles.text}>
        {t('create.center', {
          latitude: playground.center.latitude.toFixed(6), longitude: playground.center.longitude.toFixed(6),
          lastColumn: String.fromCharCode(64 + playground.columns), rows: playground.rows,
        })}
      </Text>}
      {!created && <RouteChoiceCard mode={game.routeMode} onModeChange={game.setRouteMode} playground={playground}
        points={game.routePoints} tapMissed={game.routeTapMissed} onTap={game.tapRoutePoint} disabled={busy} />}
      <Text accessibilityLiveRegion="polite" style={[styles.text, game.error && styles.error]}>
        {'key' in game.message ? t(game.message.key) : game.message.text}
      </Text>
      {busy && <ActivityIndicator color={colors.accent} />}
      {!created && <Button label={t('create.create')} onPress={game.create} disabled={busy || game.cannotCreate} />}
      {created && <>
        <JoinCodes codes={created.joinCodes} />
        <JoinAsCard codes={created.joinCodes} name={game.name} onNameChange={game.setName} busy={busy} onJoin={game.joinAs} />
      </>}
    </KeyboardScrollView>
  );
}
