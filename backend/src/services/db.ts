import { Pool } from 'pg';
import { logger } from './logger';

export const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  max: 20,
  idleTimeoutMillis: 30_000,
  connectionTimeoutMillis: 5_000,
  statement_timeout: 30_000,
});

pool.on('error', (err) => {
  logger.error('Unexpected error on idle DB client', err);
});

export async function query<T = Record<string, unknown>>(
  text: string,
  params?: unknown[]
): Promise<T[]> {
  const client = await pool.connect();
  try {
    const result = await client.query(text, params);
    return result.rows as T[];
  } finally {
    client.release();
  }
}

export async function initDb(): Promise<void> {
  await query(`
    CREATE TABLE IF NOT EXISTS apps (
      id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name        TEXT UNIQUE NOT NULL,
      repo_url    TEXT NOT NULL,
      branch      TEXT NOT NULL DEFAULT 'main',
      port        INTEGER UNIQUE NOT NULL,
      domain      TEXT,
      ssl_enabled BOOLEAN NOT NULL DEFAULT false,
      env_vars    JSONB NOT NULL DEFAULT '{}',
      created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `);

  await query(`
    CREATE TABLE IF NOT EXISTS managed_databases (
      id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name       TEXT UNIQUE NOT NULL,
      db_user    TEXT UNIQUE NOT NULL,
      password   TEXT NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `);

  // Audit log table — tracks all state-changing operations
  await query(`
    CREATE TABLE IF NOT EXISTS audit_log (
      id          BIGSERIAL PRIMARY KEY,
      username    TEXT NOT NULL,
      ip          TEXT NOT NULL,
      method      TEXT NOT NULL,
      path        TEXT NOT NULL,
      status_code INTEGER NOT NULL DEFAULT 0,
      duration_ms INTEGER NOT NULL DEFAULT 0,
      body        JSONB DEFAULT '{}',
      created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )
  `);

  // Auto-cleanup: delete audit entries older than 90 days (run once at startup)
  await query(`DELETE FROM audit_log WHERE created_at < NOW() - INTERVAL '90 days'`);

  logger.info('Database schema initialised');
}
