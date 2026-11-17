import { useRef, useState, type ComponentRef } from 'react';
import { useWindowDimensions, type View } from 'react-native';

import { setLanguage, type Language } from './i18n';

/** Where the open list sits on screen: right-aligned below the button. */
type Anchor = { top: number; right: number };

/** The language dropdown: opens a list below its button, picks a language and closes again. */
export function useLanguageMenu() {
  const button = useRef<ComponentRef<typeof View>>(null);
  const { width } = useWindowDimensions();
  const [anchor, setAnchor] = useState<Anchor | null>(null);

  const open = () => {
    button.current?.measureInWindow((x, y, w, h) => setAnchor({ top: y + h + 6, right: width - x - w }));
  };
  const close = () => setAnchor(null);
  const pick = (language: Language) => {
    setLanguage(language);
    close();
  };
  return { button, anchor, open, close, pick };
}
