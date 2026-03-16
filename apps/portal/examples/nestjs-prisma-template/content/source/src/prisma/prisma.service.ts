import { Injectable, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';
import { PrismaPg } from '@prisma/adapter-pg';
import * as pg from 'pg';

/**
 * PrismaService manages the database connection lifecycle using Prisma v7.
 *
 * Prisma v7 requires an explicit driver adapter — we use @prisma/adapter-pg
 * with the `pg` Node.js driver for PostgreSQL.
 *
 * Database credentials are injected by the Helios Operator as env vars:
 *   - DB_HOST: database hostname (from K8s Secret)
 *   - DB_USER: database username (from K8s Secret)
 *   - DB_PASS: database password (from K8s Secret)
 *   - DB_NAME: database name (from K8s ConfigMap, or defaults to component-db)
 *   - DB_PORT: database port (from K8s ConfigMap, or defaults to 5432)
 */
@Injectable()
export class PrismaService extends PrismaClient implements OnModuleInit, OnModuleDestroy {
  private pool: pg.Pool;

  constructor() {
    const pool = new pg.Pool({
      host: process.env.DB_HOST || 'localhost',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASS || 'postgres',
      database: process.env.DB_NAME || 'postgres',
      port: parseInt(process.env.DB_PORT || '5432', 10),
    });

    const adapter = new PrismaPg(pool as any);

    super({ adapter });

    this.pool = pool;
  }

  /**
   * Constructs a PostgreSQL connection URL from individual environment variables.
   * Used for Prisma CLI operations (migrate, studio) that require DATABASE_URL.
   */
  static buildDatabaseUrl(): string {
    const host = process.env.DB_HOST || 'localhost';
    const user = process.env.DB_USER || 'postgres';
    const pass = encodeURIComponent(process.env.DB_PASS || 'postgres');
    const name = process.env.DB_NAME || 'postgres';
    const port = process.env.DB_PORT || '5432';

    return `postgresql://${user}:${pass}@${host}:${port}/${name}?schema=public`;
  }

  async onModuleInit(): Promise<void> {
    await this.$connect();
  }

  async onModuleDestroy(): Promise<void> {
    await this.$disconnect();
    await this.pool.end();
  }
}
