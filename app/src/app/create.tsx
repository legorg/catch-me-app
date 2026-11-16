import { ActivityIndicator, Text, View } from 'react-native';

import Button from '@/components/button';
import CellSizeCard from '@/components/cell-size-card';
import FullscreenMap from '@/components/fullscreen-map';
import JoinAsCard from '@/components/join-as-card';
import JoinCodes from '@/components/join-codes';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import { formatNumber, useT } from '@/i18n';
import { colors, styles } from '@/styles';
import { useCreateGame } from '@/use-create-game';

export default function CreateScreen() {
  const t = useT();
  const game = useCreateGame();
  const { created, playground, displayedSize, busy } = game;

  return (
    <KeyboardScrollView>
      <Text style={styles.text}>
        {displayedSize !== null
          ? t('create.gridSized', { cell: formatNumber(displayedSize / 1000), total: formatNumber(displayedSize * 6 / 1000, { min: 1, max: 1 }) })
          : t('create.grid')}
      </Text>
      {!created && <CellSizeCard value={game.cellSizeText} onChange={game.setCellSizeText} editable={!busy} error={game.sizeError} />}
      <View style={styles.mapContainer}>
        {/* Remounts once when the position arrives, so the map opens there. */}
        <FullscreenMap key={`${created?.game.sessionId ?? 'draft'}-${game.hasPosition ? 'here' : 'default'}`} playground={playground} editable={!created && !busy} onSelect={game.selectCenter} />
      </View>
      {playground && <Text style={styles.text}>
        {t('create.center', { latitude: playground.center.latitude.toFixed(6), longitude: playground.center.longitude.toFixed(6) })}
      </Text>}
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
