const { defineConfig } = require('prisma/config');

function buildDatabaseUrl() {
  const host = process.env.DB_HOST || 'localhost';
  const user = process.env.DB_USER || 'postgres';
  const pass = encodeURIComponent(process.env.DB_PASS || 'postgres');
  const name = process.env.DB_NAME || 'postgres';
  const port = process.env.DB_PORT || '5432';

  return `postgresql://${user}:${pass}@${host}:${port}/${name}?schema=public`;
}

module.exports = defineConfig({
  schema: 'prisma/schema.prisma',
  migrations: {
    path: 'prisma/migrations',
  },
  datasource: {
    url: process.env.DATABASE_URL || buildDatabaseUrl(),
  },
});
