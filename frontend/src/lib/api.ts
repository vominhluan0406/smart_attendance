import { ApiResponse } from "./types";

/**
 * Determine the base URL for API requests.
 * - Server-side (SSR): uses API_URL env (e.g. http://gateway:8080 in Docker)
 * - Client-side: uses NEXT_PUBLIC_API_URL env (e.g. http://localhost:8080)
 */
export function getBaseUrl(): string {
  if (typeof window === "undefined") {
    // server-side
    return process.env.API_URL || process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  }
  return process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
}

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

/**
 * Core fetch wrapper. Handles JSON parsing, cookie forwarding for SSR,
 * and 401 redirect to /login.
 */
export async function apiFetch<T = unknown>(
  path: string,
  options: RequestInit = {},
  cookieHeader?: string
): Promise<ApiResponse<T>> {
  const url = `${getBaseUrl()}${path}`;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  // Forward cookies for server-side requests (SSR)
  if (typeof window === "undefined" && cookieHeader) {
    headers["Cookie"] = cookieHeader;
  }

  const res = await fetch(url, {
    ...options,
    headers,
    credentials: "include",
  });

  // Handle 401 - redirect to login
  if (res.status === 401) {
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
    throw new ApiError(401, "Phien dang nhap het han");
  }

  // Parse response body
  let body: ApiResponse<T>;
  try {
    body = await res.json();
  } catch {
    throw new ApiError(res.status, "Loi phan tich du lieu tu server");
  }

  if (!res.ok || !body.success) {
    throw new ApiError(
      res.status,
      body.error?.message || `Request failed with status ${res.status}`,
      body.error?.code
    );
  }

  return body;
}

/** GET request */
export function apiGet<T = unknown>(path: string, cookieHeader?: string) {
  return apiFetch<T>(path, { method: "GET" }, cookieHeader);
}

/** POST request */
export function apiPost<T = unknown>(
  path: string,
  body?: unknown,
  cookieHeader?: string
) {
  return apiFetch<T>(
    path,
    {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    },
    cookieHeader
  );
}

/** PUT request */
export function apiPut<T = unknown>(
  path: string,
  body?: unknown,
  cookieHeader?: string
) {
  return apiFetch<T>(
    path,
    {
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    },
    cookieHeader
  );
}

/** DELETE request */
export function apiDelete<T = unknown>(path: string, cookieHeader?: string) {
  return apiFetch<T>(path, { method: "DELETE" }, cookieHeader);
}
