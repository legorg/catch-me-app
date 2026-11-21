import { Alert } from 'react-native';

import { translate } from '@/i18n';

/** Asks before something final: the yes button is destructive and runs onYes, cancel does nothing. */
export function confirm(title: string, text: string, yesLabel: string, onYes: () => void) {
  Alert.alert(title, text, [
    { text: translate('common.cancel'), style: 'cancel' },
    { text: yesLabel, style: 'destructive', onPress: onYes },
  ]);
}

/** Tells the player something with the system alert and a single OK button. */
export function inform(title: string, text: string) {
  Alert.alert(title, text, [{ text: translate('common.ok') }]);
}
