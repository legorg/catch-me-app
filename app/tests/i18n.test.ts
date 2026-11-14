import assert from 'node:assert/strict';
import { test } from 'node:test';

import { LANGUAGES, getLanguage, isLanguage, setLanguage, translate } from '../src/i18n';

const placeholders = (text: string) => [...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();

test('every language has every text with the same placeholders', () => {
  const { en } = LANGUAGES;
  for (const [language, messages] of Object.entries(LANGUAGES)) {
    assert.deepEqual(Object.keys(messages).sort(), Object.keys(en).sort(), `${language} keys differ from en`);
    for (const [key, text] of Object.entries(messages)) {
      assert.ok(text.trim(), `${language}: ${key} is empty`);
      assert.deepEqual(placeholders(text), placeholders(en[key as keyof typeof en]), `${language}: ${key} placeholders`);
    }
  }
});

test('the language switches and placeholders are filled', () => {
  assert.equal(getLanguage(), 'en');
  assert.ok(isLanguage('de') && !isLanguage('xx') && !isLanguage('toString') && !isLanguage(null));
  setLanguage('de');
  assert.equal(translate('home.join'), 'Beitreten');
  setLanguage('en');
  assert.equal(translate('home.join'), 'Join game');
});
