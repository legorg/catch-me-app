import { router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useState } from 'react';
import { ActivityIndicator, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { MAX_NAME_LENGTH, createGame, isValidPlayerName, joinGame, type Coordinates, type CreateGameResponse } from '@/api/client';
import Button from '@/components/button';
import JoinCodes from '@/components/join-codes';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import LanguageSwitch from '@/components/language-switch';
import FullscreenMap from '@/components/fullscreen-map';
import { formatNumber, useT, type MessageKey } from '@/i18n';
import { playerSession } from '@/player-session';
import { DEFAULT_CELL_SIZE_METERS, DEFAULT_CENTER, parseCellSizeKm, previewPlayground } from '@/playground';
import { colors, styles } from '@/styles';

// Own messages are kept as keys, so they follow a language switch; error texts are shown as received.
type Message = { key: MessageKey } | { text: string };

export default function CreateScreen() {
  const t = useT();
  const [selected, setSelected] = useState<Coordinates | null>(null);
  const [cellSizeText, setCellSizeText] = useState(String(DEFAULT_CELL_SIZE_METERS / 1000));
  const [created, setCreated] = useState<CreateGameResponse | null>(null);
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<Message>({ key: 'create.pickCenter' });
  const [error, setError] = useState(false);
  const cellSize = parseCellSizeKm(cellSizeText);
  let playground = created?.game.playground ?? null;
  let sizeError = !created && cellSize === null ? t('create.cellSizeInvalid') : '';
  // The grid starts at a default spot; the user drags it, and can only create once they have.
  if (!created && cellSize !== null) {
    try {
      playground = previewPlayground(selected ?? DEFAULT_CENTER, cellSize);
    } catch (err) {
      sizeError = err instanceof Error ? err.message : t('create.sizeInvalid');
    }
  }
  const displayedSize = created?.game.playground.cellSizeMeters ?? cellSize;
  const cannotCreate = !selected || !playground || !!sizeError;

  function fail(err: unknown, fallback: MessageKey) {
    setError(true);
    setMessage(err instanceof Error ? { text: err.message } : { key: fallback });
  }

  function selectCenter(center: Coordinates) {
    try {
      if (cellSize === null) throw new Error(t('create.sizeFirst'));
      previewPlayground(center, cellSize);
      setSelected(center);
      setError(false);
      setMessage({ key: 'create.checkGrid' });
    } catch (err) {
      fail(err, 'create.centerInvalid');
    }
  }

  async function run(action: () => Promise<void>, progress: MessageKey) {
    if (busy) return;
    setBusy(true);
    setError(false);
    setMessage({ key: progress });
    try {
      await action();
    } catch (err) {
      fail(err, 'common.backendUnreachable');
    } finally {
      setBusy(false);
    }
  }

  const create = () => run(async () => {
    if (!selected || cellSize === null || cannotCreate) return;
    setCreated(await createGame(selected, cellSize));
    setMessage({ key: 'create.created' });
  }, 'create.creating');

  const joinAs = (code: string) => run(async () => {
    const joined = await joinGame(code, name);
    playerSession.set(joined.playerToken);
    router.replace('/lobby');
  }, 'create.joining');

  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <KeyboardScrollView>
        <LanguageSwitch />
        <Text style={styles.heading}>{t(created ? 'create.createdTitle' : 'create.chooseTitle')}</Text>
        <Text style={styles.text}>
          {displayedSize !== null
            ? t('create.gridSized', { cell: formatNumber(displayedSize / 1000), total: formatNumber(displayedSize * 6 / 1000, { min: 1, max: 1 }) })
            : t('create.grid')}
        </Text>
        {!created && <View style={styles.card}>
          <Text style={styles.heading}>{t('create.cellSize')}</Text>
          <TextInput
            accessibilityLabel={t('create.cellSizeLabel')}
            value={cellSizeText}
            onChangeText={setCellSizeText}
            editable={!busy}
            keyboardType="decimal-pad"
            autoCorrect={false}
            maxLength={8}
            style={styles.input}
          />
          <Text style={styles.hint}>{t('create.cellSizeHint')}</Text>
          {!!sizeError && <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{sizeError}</Text>}
        </View>}
        <View style={styles.mapContainer}>
          <FullscreenMap key={created?.game.sessionId ?? 'draft'} playground={playground} editable={!created && !busy} onSelect={selectCenter} />
        </View>
        {playground && <Text style={styles.text}>
          {t('create.center', { latitude: playground.center.latitude.toFixed(6), longitude: playground.center.longitude.toFixed(6) })}
        </Text>}
        <Text accessibilityLiveRegion="polite" style={[styles.text, error && styles.error]}>
          {'key' in message ? t(message.key) : message.text}
        </Text>
        {busy && <ActivityIndicator color={colors.accent} />}
        {!created && <Button label={t('create.create')} onPress={() => void create()} disabled={busy || cannotCreate} />}
        {created && <>
          <JoinCodes codes={created.joinCodes} />
          <View style={styles.card}>
            <Text style={styles.heading}>{t('create.joinYourself')}</Text>
            <TextInput
              accessibilityLabel={t('common.yourName')}
              placeholder={t('common.yourName')}
              placeholderTextColor={colors.placeholder}
              value={name}
              onChangeText={setName}
              editable={!busy}
              autoCorrect={false}
              maxLength={MAX_NAME_LENGTH}
              style={styles.input}
            />
            <View style={styles.row}>
              <Button label={t('create.joinAsRunner')} onPress={() => void joinAs(created.joinCodes.runner)} disabled={busy || !isValidPlayerName(name)} />
              <Button label={t('create.joinAsHunter')} onPress={() => void joinAs(created.joinCodes.hunter)} disabled={busy || !isValidPlayerName(name)} />
            </View>
          </View>
        </>}
      </KeyboardScrollView>
    </SafeAreaView>
  );
}
