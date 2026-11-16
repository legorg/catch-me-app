import { Text, View } from 'react-native';

import Button from '@/components/button';
import { useT } from '@/i18n';
import { fireAndForget } from '@/press';
import { styles } from '@/styles';
import type { LocationSharing } from '@/use-location-sharing';

type Props = { location: LocationSharing };

function BackgroundDetails({ location }: Props) {
  const t = useT();
  const { background } = location;
  if (background.active) return <Text style={styles.text}>{t('lobby.sharingBackgroundText')}</Text>;
  if (!background.available) return <Text style={styles.hint}>{t('lobby.backgroundUnavailable')}</Text>;
  if (background.permission?.canAskAgain === false) return <Text style={styles.hint}>{t('lobby.backgroundBlocked')}</Text>;
  return (
    <>
      <Text style={styles.text}>{t('lobby.allowAlways')}</Text>
      <Button label={t('lobby.shareInBackground')} variant="secondary" onPress={fireAndForget(background.request)} />
    </>
  );
}

function BackgroundSharing({ location }: Props) {
  const t = useT();
  return (
    <View style={styles.gapSmall}>
      <Text style={styles.heading}>{t(location.background.active ? 'lobby.sharingBackground' : 'lobby.sharingForeground')}</Text>
      <BackgroundDetails location={location} />
    </View>
  );
}

function AskForLocation({ location }: Props) {
  const t = useT();
  return (
    <View style={styles.gapSmall}>
      <Text style={styles.heading}>{t('lobby.shareHeading')}</Text>
      <Text style={styles.text}>{t(location.background.available ? 'lobby.shareRunning' : 'lobby.shareRunningForeground')}</Text>
      {location.permission?.canAskAgain && <Button label={t('lobby.allowLocation')} onPress={fireAndForget(location.requestPermission)} />}
      {!location.permission?.canAskAgain && <Text style={[styles.text, styles.error]}>{t('lobby.locationBlocked')}</Text>}
    </View>
  );
}

/** Info panel section: asks for location access, or says how the position is being shared. */
export default function LocationSharingInfo({ location }: Props) {
  if (!location.permission) return null;
  if (!location.permission.granted) return <AskForLocation location={location} />;
  return <BackgroundSharing location={location} />;
}
