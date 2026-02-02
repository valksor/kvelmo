/**
 * CSRF Token Management Module
 *
 * Fetches and caches the CSRF token for authenticated sessions.
 * Provides utilities for injecting the token into requests.
 *
 * @module csrf
 */

let cachedToken = '';

/**
 * Fetch the CSRF token from the server.
 * Caches the result to avoid repeated network requests.
 *
 * @returns {Promise<string>} The CSRF token, or empty string if unavailable.
 */
export async function fetchCSRFToken() {
    if (cachedToken) {
        return cachedToken;
    }

    try {
        const resp = await fetch('/api/v1/auth/csrf', { credentials: 'same-origin' });
        if (resp.ok) {
            const data = await resp.json();
            cachedToken = data.csrf_token || '';
        }
    } catch {
        // No auth configured or not logged in — CSRF not required
    }

    return cachedToken;
}

/**
 * Get the cached CSRF token synchronously.
 * Returns empty string if not yet fetched.
 *
 * @returns {string} The cached CSRF token.
 */
export function getCSRFToken() {
    return cachedToken;
}

/**
 * Clear the cached CSRF token (e.g., on logout).
 */
export function clearCSRFToken() {
    cachedToken = '';
}

/**
 * A fetch() wrapper that automatically injects the CSRF token header
 * on state-changing requests (POST, PUT, DELETE, PATCH).
 *
 * @param {string} url - The URL to fetch.
 * @param {RequestInit} [options={}] - Fetch options.
 * @returns {Promise<Response>} The fetch response.
 */
export async function csrfFetch(url, options = {}) {
    const method = (options.method || 'GET').toUpperCase();
    const needsCSRF = ['POST', 'PUT', 'DELETE', 'PATCH'].includes(method);

    if (needsCSRF && cachedToken) {
        options.headers = {
            ...options.headers,
            'X-Csrf-Token': cachedToken,
        };
    }

    return fetch(url, options);
}
