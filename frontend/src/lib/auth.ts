import { cookies } from "next/headers";
import { Session, UserRole } from "./types";

/**
 * Decode a JWT payload without verification.
 * We only need to extract claims -- the gateway already verified the token.
 */
function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = parts[1];
    // base64url -> base64 -> decode
    const base64 = payload.replace(/-/g, "+").replace(/_/g, "/");
    const json = Buffer.from(base64, "base64").toString("utf-8");
    return JSON.parse(json);
  } catch {
    return null;
  }
}

/**
 * Read the session from the access_token cookie (server-side only).
 * Returns null if no valid token or if the token has expired.
 */
export async function getSession(): Promise<Session | null> {
  const cookieStore = await cookies();
  const token = cookieStore.get("access_token")?.value;
  if (!token) return null;

  const payload = decodeJwtPayload(token);
  if (!payload) return null;

  // Check expiration if present (exp is in seconds)
  if (payload.exp && typeof payload.exp === "number") {
    const now = Math.floor(Date.now() / 1000);
    if (now >= payload.exp) {
      return null;
    }
  }

  return {
    userId: (payload.user_id || payload.sub || "") as string,
    email: (payload.email || "") as string,
    fullName: (payload.full_name || payload.name || "") as string,
    role: (payload.role || "employee") as UserRole,
    branchId: (payload.branch_id || undefined) as string | undefined,
    branchName: (payload.branch_name || undefined) as string | undefined,
  };
}


/**
 * Get the raw cookie header string for forwarding to API in SSR.
 */
export async function getCookieHeader(): Promise<string> {
  const cookieStore = await cookies();
  return cookieStore
    .getAll()
    .map((c) => `${c.name}=${c.value}`)
    .join("; ");
}
