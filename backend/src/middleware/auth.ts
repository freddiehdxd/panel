import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import { JwtPayload } from '../types';

// No fallback — startup validation in index.ts guarantees JWT_SECRET exists
const JWT_SECRET = process.env.JWT_SECRET!;
const COOKIE_NAME = 'panel_token';

export function authMiddleware(req: Request, res: Response, next: NextFunction): void {
  // 1. Check HttpOnly cookie first (preferred — XSS-safe)
  const cookieToken = req.cookies?.[COOKIE_NAME];
  // 2. Fall back to Authorization header (for API clients / backward compat)
  const headerToken = req.headers.authorization?.startsWith('Bearer ')
    ? req.headers.authorization.slice(7)
    : null;

  const token = cookieToken ?? headerToken;

  if (!token) {
    res.status(401).json({ success: false, error: 'Missing token' });
    return;
  }

  try {
    const payload = jwt.verify(token, JWT_SECRET) as JwtPayload;
    req.user = payload;
    next();
  } catch {
    res.status(401).json({ success: false, error: 'Invalid or expired token' });
  }
}

export function signToken(username: string): string {
  return jwt.sign({ sub: username }, JWT_SECRET, { expiresIn: '2h' });
}

export function verifyToken(token: string): JwtPayload | null {
  try {
    return jwt.verify(token, JWT_SECRET) as JwtPayload;
  } catch {
    return null;
  }
}
