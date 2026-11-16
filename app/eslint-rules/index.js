'use strict';

module.exports = {
  meta: { name: 'eslint-plugin-catchme' },
  rules: {
    'no-functions-in-components': require('./no-functions-in-components'),
  },
};
