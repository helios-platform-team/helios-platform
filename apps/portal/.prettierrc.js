module.exports = {
  ...require('@backstage/cli/config/prettier'),
  plugins: ['@trivago/prettier-plugin-sort-imports'],
  importOrder: ['^@backstage/(.*)$', '<THIRD_PARTY_MODULES>', '^[./]'],
  importOrderSeparation: true,
  importOrderSortSpecifiers: true,
};
