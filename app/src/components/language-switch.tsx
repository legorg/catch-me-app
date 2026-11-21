import { Alert, Pressable, Text } from 'react-native';

import { LANGUAGES, getLanguage, isLanguage, setLanguage, translate, useT, type Language } from '../i18n';
import { languageSwitch } from '../styles';

// Each language's short code, and its name in that language.
const LABELS: Record<Language, { code: string; name: string }> = {
  de: { code: 'DE', name: 'Deutsch' },
  en: { code: 'EN', name: 'English' },
};

/** Lists the languages in a system alert, the current one checked. Android shows at most three buttons. */
function chooseLanguage() {
  const current = getLanguage();
  Alert.alert(translate('language.switch'), undefined, [
    ...Object.keys(LANGUAGES).filter(isLanguage).map((language) => ({
      text: language === current ? `✓ ${LABELS[language].name}` : LABELS[language].name,
      onPress: () => setLanguage(language),
    })),
    { text: translate('common.cancel'), style: 'cancel' as const },
  ]);
}

/** Shows the current language's code; switches the app language until the app restarts. */
export default function LanguageSwitch() {
  const t = useT();
  const current = getLanguage();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${t('language.switch')}: ${LABELS[current].name}`}
      onPress={chooseLanguage}
      hitSlop={8}
      style={languageSwitch.button}
    >
      <Text style={languageSwitch.code}>{LABELS[current].code}</Text>
    </Pressable>
  );
}
