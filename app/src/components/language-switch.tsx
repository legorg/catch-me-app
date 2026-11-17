import { Modal, Pressable, Text, View } from 'react-native';

import { LANGUAGES, getLanguage, isLanguage, useT, type Language } from '../i18n';
import { dynamicStyles, languageSwitch } from '../styles';
import { useLanguageMenu } from '../use-language-menu';

// Each language's flag, and its name in that language.
const FLAGS: Record<Language, { flag: string; name: string }> = {
  de: { flag: '🇩🇪', name: 'Deutsch' },
  en: { flag: '🇬🇧', name: 'English' },
};

/** Dropdown showing the current language's flag; switches the app language until the app restarts. */
export default function LanguageSwitch() {
  const t = useT();
  const current = getLanguage();
  const { button, anchor, open, close, pick } = useLanguageMenu();
  return (
    <>
      <Pressable
        ref={button}
        accessibilityRole="button"
        accessibilityLabel={`${t('language.switch')}: ${FLAGS[current].name}`}
        accessibilityState={{ expanded: anchor !== null }}
        onPress={open}
        hitSlop={8}
        style={languageSwitch.button}
      >
        <Text style={languageSwitch.flag}>{FLAGS[current].flag}</Text>
        <Text style={languageSwitch.caret}>▾</Text>
      </Pressable>
      {/* A modal, so a tap anywhere else closes the list and nothing on the screen covers it. */}
      <Modal visible={anchor !== null} transparent animationType="fade" statusBarTranslucent navigationBarTranslucent onRequestClose={close}>
        <Pressable style={languageSwitch.backdrop} onPress={close} accessibilityLabel={t('language.close')} />
        {anchor && (
          <View accessibilityRole="menu" style={[languageSwitch.menu, dynamicStyles.languageMenu(anchor)]}>
            {Object.keys(LANGUAGES).filter(isLanguage).map((language) => (
              <Pressable
                key={language}
                accessibilityRole="menuitem"
                accessibilityState={{ selected: language === current }}
                onPress={() => pick(language)}
                style={({ pressed }) => [languageSwitch.item, pressed && languageSwitch.pressed]}
              >
                <Text style={languageSwitch.flag}>{FLAGS[language].flag}</Text>
                <Text style={[languageSwitch.name, language === current && languageSwitch.current]}>{FLAGS[language].name}</Text>
              </Pressable>
            ))}
          </View>
        )}
      </Modal>
    </>
  );
}
