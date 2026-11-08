// API keys are supplied locally or by the native build environment, never committed.
module.exports = ({ config }) => ({
  ...config,
  plugins: [
    ...(config.plugins ?? []),
    ['react-native-maps', {
      androidGoogleMapsApiKey: process.env.GOOGLE_MAPS_ANDROID_API_KEY,
    }],
  ],
});
