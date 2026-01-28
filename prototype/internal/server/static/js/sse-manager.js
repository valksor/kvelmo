/**
 * SSE Manager Module
 *
 * Manages SSE connection status display and accessibility announcements.
 * Works alongside the htmx-sse.js extension.
 *
 * @module sse-manager
 */

const MAX_RETRIES = 10;
const BASE_RETRY_DELAY = 1000;

let retryCount = 0;

/**
 * Initialize SSE event handlers.
 */
export function initSSE() {
    setupConnectionHandlers();
    setupA11yAnnouncements();
}

/**
 * Set up handlers for SSE connection status.
 */
function setupConnectionHandlers() {
    // SSE connection opened
    document.body.addEventListener('htmx:sseOpen', () => {
        updateConnectionStatus('connected');
    });

    // SSE connection error
    document.body.addEventListener('htmx:sseError', (evt) => {
        console.error('[SSE] Connection error:', evt);
        updateConnectionStatus('disconnected');

        // Attempt reconnection with exponential backoff
        if (retryCount < MAX_RETRIES) {
            const delay = BASE_RETRY_DELAY * Math.pow(2, retryCount);
            retryCount++;
            setTimeout(() => updateConnectionStatus('connecting'), delay);
        }
    });

    // SSE message received (reset retry count on successful message)
    document.body.addEventListener('htmx:sseMessage', () => {
        retryCount = 0;
        updateConnectionStatus('connected');
    });
}

/**
 * Update the connection status indicator in the navbar.
 *
 * @param {'connected' | 'connecting' | 'disconnected'} status - The connection status
 */
export function updateConnectionStatus(status) {
    const dot = document.getElementById('connection-dot');
    const text = document.getElementById('connection-text');

    if (!dot || !text) return;

    switch (status) {
        case 'connected':
            dot.className = 'w-2 h-2 rounded-full bg-success-500';
            text.textContent = 'Connected';
            retryCount = 0;
            break;
        case 'connecting':
            dot.className = 'w-2 h-2 rounded-full bg-warning-500 animate-pulse';
            text.textContent = 'Connecting...';
            break;
        case 'disconnected':
            dot.className = 'w-2 h-2 rounded-full bg-error-500';
            text.textContent = 'Disconnected';
            break;
    }
}

/**
 * Set up screen reader announcements for SSE state changes.
 */
function setupA11yAnnouncements() {
    // Announce workflow state changes to screen readers
    document.body.addEventListener('htmx:afterSwap', (evt) => {
        const stateElement = evt.detail.target?.querySelector('[data-state]');
        if (stateElement && stateElement.dataset.state) {
            announceToScreenReader('Workflow state: ' + stateElement.dataset.state);
        }
    });
}

/**
 * Announce a message to screen readers via ARIA live region.
 *
 * @param {string} message - The message to announce
 */
export function announceToScreenReader(message) {
    const announcer = document.getElementById('a11y-announcements');
    if (announcer) {
        // Clear and re-set to trigger announcement
        announcer.textContent = '';
        setTimeout(() => { announcer.textContent = message; }, 100);
    }
}

/**
 * Get the current connection status.
 *
 * @returns {'connected' | 'connecting' | 'disconnected'} The current status
 */
export function getConnectionStatus() {
    const text = document.getElementById('connection-text');
    if (!text) return 'disconnected';

    const content = text.textContent?.toLowerCase() || '';
    if (content.includes('connected') && !content.includes('disconnected')) {
        return 'connected';
    }
    if (content.includes('connecting')) {
        return 'connecting';
    }
    return 'disconnected';
}
