import { Pressable, Text, View } from 'react-native';

import { LANGUAGES, getLanguage, isLanguage, setLanguage, useT } from '../i18n';
import { styles } from '../styles';

/** Switches the app language until the app restarts. */
export default function LanguageSwitch() {
  const t = useT();
  const current = getLanguage();
  return (
    <View accessibilityRole="radiogroup" accessibilityLabel={t('language.switch')} style={[styles.row, styles.alignEnd]}>
      {Object.keys(LANGUAGES).filter(isLanguage).map((language) => (
        <Pressable
          key={language}
          accessibilityRole="radio"
          accessibilityState={{ checked: language === current }}
          onPress={() => setLanguage(language)}
          hitSlop={8}
        >
          <Text style={[styles.hint, styles.boldText, language === current && styles.accentText]}>
            {language.toUpperCase()}
          </Text>
        </Pressable>
      ))}
    </View>
  );
}
