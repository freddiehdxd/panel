import winston from 'winston';
import fs from 'fs';

const LOG_DIR = '/var/log/panel';

// Ensure log directory exists before Winston tries to write to it
try {
  fs.mkdirSync(LOG_DIR, { recursive: true });
} catch {
  // Non-fatal — console logging will still work
}

export const logger = winston.createLogger({
  level: process.env.NODE_ENV === 'production' ? 'info' : 'debug',
  format: winston.format.combine(
    winston.format.timestamp(),
    winston.format.errors({ stack: true }),
    winston.format.json()
  ),
  transports: [
    new winston.transports.Console({
      format: winston.format.combine(
        winston.format.colorize(),
        winston.format.simple()
      ),
    }),
    new winston.transports.File({
      filename: `${LOG_DIR}/error.log`,
      level: 'error',
    }),
    new winston.transports.File({
      filename: `${LOG_DIR}/combined.log`,
    }),
  ],
});
