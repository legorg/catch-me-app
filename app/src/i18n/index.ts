import { useSyncExternalStore } from 'react';

import de from './de.json';
import en from './en.json';

export type MessageKey = keyof typeof en;

// To add a language, create <code>.json with every key of en.json and list it
// here; the type check reports missing keys.
export const LANGUAGES = { en, de } satisfies Record<string, Record<MessageKey, string>>;
export type Language = keyof typeof LANGUAGES;

let current: Language = 'en';
const listeners = new Set<() => void>();

export function isLanguage(code: string | null | undefined): code is Language {
  return !!code && Object.hasOwn(LANGUAGES, code);
}

/** Changes the language until the app restarts and re-renders every screen using useT. */
export function setLanguage(language: Language) {
  current = language;
  listeners.forEach((listener) => listener());
}

export function getLanguage(): Language {
  return current;
}

/** The text for key in the current language; {name} placeholders are filled from params. */
export function translate(key: MessageKey, params: Record<string, string | number> = {}): string {
  return LANGUAGES[current][key].replace(/\{(\w+)\}/g, (match, name: string) => (name in params ? String(params[name]) : match));
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}

/** translate for components: re-renders them when the language changes. */
export function useT(): typeof translate {
  useSyncExternalStore(subscribe, getLanguage, getLanguage);
  return translate;
}
