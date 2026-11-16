import { Text, View } from 'react-native';

import Button from '@/components/button';
import { useT } from '@/i18n';
import { styles } from '@/styles';
import { type LocationSharing } from '@/use-location-sharing';

/** The lobby's location cards: foreground access, the sharing error, and "Always" access for background sharing. */
export default function LocationPermissionCards({
  location, team, skipStartPositionCheck, ready, sharing, needsForeground, needsBackground, onRequestLocation, onRequestBackground,
}: {
  location: LocationSharing;
  team: 'runner' | 'hunter';
  skipStartPositionCheck: boolean;
  ready: boolean;
  sharing: boolean;
  needsForeground: boolean;
  needsBackground: boolean;
  onRequestLocation: () => void;
  onRequestBackground: () => void;
}) {
  const t = useT();
  const { permission, background } = location;
  const canAskBackground = background.permission?.canAskAgain !== false;
  return (
    <>
      {sharing && permission && !permission.granted && <View style={styles.card}>
        <Text style={styles.heading}>{t('lobby.shareHeading')}</Text>
        <Text style={styles.text}>
          {t(team === 'runner' && !skipStartPositionCheck ? 'lobby.shareRunnerStart' : 'lobby.shareHunterLobby')}
        </Text>
        {needsForeground && <Text style={[styles.text, !ready && styles.error]}>{t('lobby.locationRequired')}</Text>}
        {permission.canAskAgain
          ? <Button label={t('lobby.allowLocation')} onPress={onRequestLocation} />
          : <Text style={[styles.text, styles.error]}>{t('lobby.locationBlocked')}</Text>}
      </View>}
      {!!location.error && <Text style={[styles.text, styles.error]}>{location.error}</Text>}
      {sharing && needsBackground && <View style={styles.card}>
        <Text style={styles.heading}>{t('lobby.backgroundHeading')}</Text>
        <Text style={[styles.text, !ready && styles.error]}>{t('lobby.backgroundRequired')}</Text>
        {/* "Always" can only be asked for once foreground access is granted (see the card above). */}
        {permission?.granted && canAskBackground && <Button label={t('lobby.shareInBackground')} variant="secondary" onPress={onRequestBackground} />}
        {permission?.granted && !canAskBackground && <Text style={[styles.text, styles.error]}>{t('lobby.backgroundRequiredBlocked')}</Text>}
      </View>}
    </>
  );
}
