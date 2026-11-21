// API keys are supplied locally or by the native build environment, never committed.
const { withAppBuildGradle } = require('expo/config-plugins');

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

module.exports = ({ config }) => withWorkManager({
  ...config,
  // Android's window background, behind the navigation bar and while the app starts.
  backgroundColor: palette.background,
  plugins: [
    ...(config.plugins ?? []),
    ['react-native-maps', {
      androidGoogleMapsApiKey: process.env.GOOGLE_MAPS_KEY,
    }],
  ],
});
