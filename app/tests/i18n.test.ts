import assert from 'node:assert/strict';
import { test } from 'node:test';

import { LANGUAGES, formatNumber, getLanguage, isLanguage, setLanguage, translate, translateParts } from '../src/i18n';

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
  assert.equal(translate('map.savedGrid', { rows: 6, columns: 6 }), 'Saved grid: 6 × 6');
  assert.equal(formatNumber(12, { min: 1, max: 1 }), '12.0');
  setLanguage('de');
  assert.equal(translate('home.join'), 'Beitreten');
  assert.equal(translate('playground.unknownCell', { cell: 'G7' }), 'Unbekanntes Feld G7.');
  assert.equal(formatNumber(0.5), '0,5');
  setLanguage('en');
  assert.equal(translate('home.join'), 'Join game');
});

test('translateParts puts any value in place of placeholders', () => {
  const start = { cell: 'B1' };
  assert.deepEqual(translateParts('lobby.goToStart', { cell: 'C3', start }), ['You are in ', 'C3', '. Go to ', start, ' to get ready.']);
  assert.deepEqual(translateParts('lobby.youAre', {}), ['You are a ', '{team}']);
});
