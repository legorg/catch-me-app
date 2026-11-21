// API keys are supplied locally or by the native build environment, never committed.
const palette = require('./src/palette.json');

module.exports = ({ config }) => ({
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
