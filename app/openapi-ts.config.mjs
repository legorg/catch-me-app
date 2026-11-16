/** @type {import('@hey-api/openapi-ts').UserConfig} */
export default {
  input: '../backend/api/openapi.json',
  output: 'src/api/generated',
  plugins: ['@hey-api/typescript', '@hey-api/client-fetch', '@hey-api/sdk'],
};
