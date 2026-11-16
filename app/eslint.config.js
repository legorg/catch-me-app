const { defineConfig } = require('eslint/config');
const expoConfig = require('eslint-config-expo/flat');
const catchme = require('./eslint-rules');

// Expo's TypeScript rules also apply to the ESM-only development scripts.
const configWithEsmTypescript = expoConfig.map((config) =>
  config.files?.includes('**/*.ts')
    ? { ...config, files: [...config.files, '**/*.mts'] }
    : config,
);

module.exports = defineConfig([
  configWithEsmTypescript,
  {
    ignores: ['dist/**', '.expo/**', 'src/api/generated/**'],
  },
  {
    // Project-local rules (see eslint-rules/). Components only: tests and generated code are excluded.
    files: ['src/**/*.tsx'],
    plugins: { catchme },
    rules: { 'catchme/no-functions-in-components': 'warn' },
  },
  {
    files: ['**/*.mts'],
    settings: {
      'import/resolver': { node: { extensions: ['.js', '.ts', '.mts', '.mjs'] } },
    },
  },
]);
