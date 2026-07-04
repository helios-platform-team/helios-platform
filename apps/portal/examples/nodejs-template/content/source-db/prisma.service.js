const { PrismaClient } = require('./generated/prisma/client');
const { PrismaPg } = require('@prisma/adapter-pg');
const pg = require('pg');

class PrismaService extends PrismaClient {
  constructor() {
    const pool = new pg.Pool({
      host: process.env.DB_HOST || 'localhost',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASS || 'postgres',
      database: process.env.DB_NAME || 'postgres',
      port: parseInt(process.env.DB_PORT || '5432', 10),
    });

    const adapter = new PrismaPg(pool);
    super({ adapter });
    this.pool = pool;
  }

  static buildDatabaseUrl() {
    const host = process.env.DB_HOST || 'localhost';
    const user = process.env.DB_USER || 'postgres';
    const pass = encodeURIComponent(process.env.DB_PASS || 'postgres');
    const name = process.env.DB_NAME || 'postgres';
    const port = process.env.DB_PORT || '5432';

    return `postgresql://${user}:${pass}@${host}:${port}/${name}?schema=public`;
  }

  async onModuleInit() {
    const maxRetries = 5;
    let delay = 1000;
    for (let i = 0; i < maxRetries; i++) {
      try {
        await this.$connect();
        return;
      } catch (err) {
        if (i === maxRetries - 1) {
          console.error(
            `Could not establish connection to the database: ${err.message}. Continuing startup without active database connection.`
          );
          return;
        }
        console.warn(
          `Failed to connect to database (attempt ${i + 1}/${maxRetries}): ${
            err.message
          }. Retrying in ${delay}ms...`
        );
        await new Promise((resolve) => setTimeout(resolve, delay));
        delay *= 2;
      }
    }
  }

  async onModuleDestroy() {
    await this.$disconnect();
    await this.pool.end();
  }
}

module.exports = { PrismaService };
