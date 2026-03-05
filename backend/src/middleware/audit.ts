import { Request, Response, NextFunction } from 'express';
import { query } from '../services/db';
import { logger } from '../services/logger';

/**
 * Audit logging middleware.
 * Logs all state-changing requests (POST, PUT, DELETE) to the audit_log table.
 * GET requests are not logged to avoid noise.
 */
export function auditMiddleware(req: Request, res: Response, next: NextFunction): void {
  // Only log state-changing operations
  if (!['POST', 'PUT', 'DELETE', 'PATCH'].includes(req.method)) {
    next();
    return;
  }

  // Capture original end to log after response is sent
  const originalEnd = res.end;
  const startTime = Date.now();

  // Override res.end to capture status code
  res.end = function (...args: Parameters<typeof originalEnd>) {
    const duration = Date.now() - startTime;
    const username = req.user?.sub ?? 'anonymous';
    const ip =
      (req.headers['x-real-ip'] as string) ??
      (req.headers['x-forwarded-for'] as string)?.split(',')[0]?.trim() ??
      req.ip ??
      'unknown';

    // Sanitize body — never log passwords or tokens
    const sanitizedBody = { ...req.body };
    for (const key of ['password', 'token', 'secret', 'jwt', 'current_password', 'new_password']) {
      if (key in sanitizedBody) sanitizedBody[key] = '***REDACTED***';
    }

    // Fire-and-forget — don't block the response
    query(
      `INSERT INTO audit_log (username, ip, method, path, status_code, duration_ms, body)
       VALUES ($1, $2, $3, $4, $5, $6, $7)`,
      [
        username,
        ip,
        req.method,
        req.originalUrl,
        res.statusCode,
        duration,
        JSON.stringify(sanitizedBody),
      ]
    ).catch((err) => {
      logger.error('Failed to write audit log', { error: err.message });
    });

    return originalEnd.apply(res, args);
  } as typeof originalEnd;

  next();
}
