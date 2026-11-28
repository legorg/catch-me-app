// API keys are supplied locally or by the native build environment, never committed.
const { AndroidConfig, withAndroidManifest, withAppBuildGradle } = require('expo/config-plugins');

const palette = require('./src/palette.json');

// react-native-maps pins WorkManager 2.9.1, which brings Room 2.5. Before 2.7, Room's keep rule
// keeps its generated databases but not their constructors, so R8 removes them and minified
// release builds crash at start creating WorkManager's database. WorkManager 2.11 brings Room 2.7,
// whose rule also keeps the constructor.
const WORK_MANAGER = 'androidx.work:work-runtime:2.11.2';

function withWorkManager(config) {
  return withAppBuildGradle(config, (gradle) => {
    if (!gradle.modResults.contents.includes(WORK_MANAGER)) {
      gradle.modResults.contents = gradle.modResults.contents.replace(
        /^dependencies\s*\{/m,
        (match) => `${match}\n    implementation("${WORK_MANAGER}")`,
      );
    }
    return gradle;
  });
}

// End-to-end test builds (E2E_BUILD=true) talk to a local plain-HTTP backend on the emulator's
// host (10.0.2.2). Android release builds block cleartext HTTP, so only these builds allow it.
// Never set E2E_BUILD for builds that are distributed.
function withE2ECleartextTraffic(config) {
  if (process.env.E2E_BUILD !== 'true') {
    return config;
  }
  return withAndroidManifest(config, (manifest) => {
    AndroidConfig.Manifest.getMainApplicationOrThrow(manifest.modResults).$['android:usesCleartextTraffic'] = 'true';
    return manifest;
  });
}

module.exports = ({ config }) => withE2ECleartextTraffic(withWorkManager({
  ...config,
  // Android's window background, behind the navigation bar and while the app starts.
  backgroundColor: palette.background,
  plugins: [
    ...(config.plugins ?? []),
    ['react-native-maps', {
      androidGoogleMapsApiKey: process.env.GOOGLE_MAPS_KEY,
    }],
  ],
}));
