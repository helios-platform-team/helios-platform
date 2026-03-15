import { defineConfig } from 'prisma/config';

/**
 * Constructs a PostgreSQL connection URL from individual environment variables.
 * These env vars are injected by the Helios Operator:
 *   - DB_HOST, DB_USER, DB_PASS (from K8s Secret)
 *   - DB_NAME, DB_PORT (from K8s ConfigMap / CUE engine)
 */
function buildDatabaseUrl(): string {
  const host = process.env.DB_HOST || 'localhost';
  const user = process.env.DB_USER || 'postgres';
  const pass = encodeURIComponent(process.env.DB_PASS || 'postgres');
  const name = process.env.DB_NAME || 'postgres';
  const port = process.env.DB_PORT || '5432';

  return `postgresql://${user}:${pass}@${host}:${port}/${name}?schema=public`;
}

export default defineConfig({
  schema: 'prisma/schema.prisma',
  migrations: {
    path: 'prisma/migrations',
  },
  datasource: {
    url: process.env.DATABASE_URL || buildDatabaseUrl(),
  },
});
