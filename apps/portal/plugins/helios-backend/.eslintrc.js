module.exports = {
  extends: ['@backstage/eslint-config'],
  rules: {
    '@typescript-eslint/no-unused-vars': [
      'error',
      { argsIgnorePattern: '^_' },
    ],
  },
};
