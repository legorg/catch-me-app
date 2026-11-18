import { useLocalSearchParams } from 'expo-router';
import { ActivityIndicator, Text, View } from 'react-native';

import { ApiError } from '@/api/client';
import Button from '@/components/button';
import GameResultSummary from '@/components/game-result-summary';
import KeyboardScrollView from '@/components/keyboard-scroll-view';
import { isValidGameId, normalizeGameId } from '@/game-result';
import { useT } from '@/i18n';
import { colors, styles } from '@/styles';
import { returnFromResult, useGameResult } from '@/use-game-result';

function ResultContent({ sessionId }: { sessionId: string }) {
  const t = useT();
  const { result, error, busy, retry } = useGameResult(sessionId);
  const valid = isValidGameId(sessionId);
  const message = !valid ? t('result.invalidId')
    : error instanceof ApiError && error.status === 404 ? t('result.unavailable')
      : error instanceof Error ? error.message : t('common.backendUnreachable');
  return (
    <KeyboardScrollView>
      <View style={styles.card}>
        <Text style={styles.heading}>{t('result.gameId')}</Text>
        <Text selectable style={styles.text}>{sessionId}</Text>
        <Text style={styles.hint}>{t('result.keepIdPrivate')}</Text>
      </View>
      {busy && <>
        <ActivityIndicator color={colors.accent} />
        <Text accessibilityLiveRegion="polite" style={styles.text}>{t('result.loading')}</Text>
      </>}
      {!!error && <View style={styles.card}>
        <Text accessibilityLiveRegion="polite" style={[styles.text, styles.error]}>{message}</Text>
        {valid && <Button label={t('result.retry')} onPress={retry} />}
      </View>}
      {result && <GameResultSummary result={result} />}
      <Button label={t('result.home')} variant="secondary" onPress={returnFromResult} />
    </KeyboardScrollView>
  );
}

/** Direct entry and automatic game-end navigation use exactly this screen. */
export default function GameResultScreen() {
  const { sessionId } = useLocalSearchParams<{ sessionId?: string | string[] }>();
  const id = normalizeGameId(typeof sessionId === 'string' ? sessionId : '');
  return <ResultContent key={id} sessionId={id} />;
}
