const { defineConfig } = require('eslint/config');
const expoConfig = require('eslint-config-expo/flat');

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
    files: ['**/*.mts'],
    settings: {
      'import/resolver': { node: { extensions: ['.js', '.ts', '.mts', '.mjs'] } },
    },
  },
]);
