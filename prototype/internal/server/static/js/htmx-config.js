/**
 * HTMX Configuration Module
 *
 * Sets up HTMX event handlers, loading states, and configuration.
 *
 * @module htmx-config
 */

import { showToast, showBrowserNotification } from './notifications.js';

/**
 * Initialize HTMX event handlers and configuration.
 */
export function initHTMX() {
    setupRequestHandlers();
    setupResponseHandlers();
    setupNavigationHandlers();
    setupConfigRequest();
}

/**
 * Set up handlers for HTMX request lifecycle.
 */
function setupRequestHandlers() {
    // Add loading state to buttons during requests
    document.body.addEventListener('htmx:beforeRequest', (evt) => {
        const el = evt.detail.elt;
        if (el.tagName === 'BUTTON') {
            el.disabled = true;
            el.classList.add('htmx-request');
        }
    });

    // Remove loading state after request completes
    document.body.addEventListener('htmx:afterRequest', (evt) => {
        const el = evt.detail.elt;
        if (el.tagName === 'BUTTON') {
            el.disabled = false;
            el.classList.remove('htmx-request');
        }
    });
}

/**
 * Set up handlers for HTMX responses.
 */
function setupResponseHandlers() {
    // Handle JSON responses with success/error messages
    document.body.addEventListener('htmx:afterRequest', (evt) => {
        try {
            const xhr = evt.detail.xhr;
            if (!xhr || !xhr.responseText) return;

            // Only parse JSON responses
            const contentType = xhr.getResponseHeader('Content-Type');
            if (!contentType || !contentType.includes('application/json')) return;

            const response = JSON.parse(xhr.responseText);

            if (response.error) {
                showToast(response.error, 'error');
                showBrowserNotification('Error', response.error, 'error');
            } else if (response.success && response.message) {
                showToast(response.message, 'success');
                showBrowserNotification('Success', response.message, 'success');
            }
        } catch (e) {
            // Response might not be JSON, ignore
        }
    });

    // Handle HTMX errors
    document.body.addEventListener('htmx:responseError', (evt) => {
        const status = evt.detail.xhr?.status;
        let message = 'An error occurred';

        switch (status) {
            case 400:
                message = 'Bad request';
                break;
            case 401:
                message = 'Please log in to continue';
                // Redirect to login
                window.location.href = '/login';
                return;
            case 403:
                message = 'Access denied';
                break;
            case 404:
                message = 'Not found';
                break;
            case 500:
                message = 'Server error. Please try again.';
                break;
        }

        showToast(message, 'error');
    });
}

/**
 * Set up handlers for navigation events.
 */
function setupNavigationHandlers() {
    // Scroll to top on page navigation
    document.body.addEventListener('htmx:beforeSwap', (evt) => {
        const requestPath = evt.detail.pathInfo?.requestPath;
        if (requestPath && requestPath !== window.location.pathname) {
            window.scrollTo({ top: 0, behavior: 'smooth' });
        }
    });
}

/**
 * Configure HTMX request defaults.
 */
function setupConfigRequest() {
    document.body.addEventListener('htmx:configRequest', (evt) => {
        // Add custom header to identify HTMX requests
        evt.detail.headers['X-Requested-With'] = 'XMLHttpRequest';
    });
}

/**
 * Trigger an HTMX refresh on a specific element.
 *
 * @param {string} selector - CSS selector for the element to refresh
 */
export function triggerRefresh(selector) {
    const el = document.querySelector(selector);
    if (el && window.htmx) {
        window.htmx.trigger(el, 'refresh');
    }
}

/**
 * Process a response and swap content using HTMX.
 *
 * @param {string} html - HTML content to swap
 * @param {string} target - CSS selector for target element
 * @param {string} [swap='innerHTML'] - HTMX swap strategy
 */
export function swapContent(html, target, swap = 'innerHTML') {
    const el = document.querySelector(target);
    if (el && window.htmx) {
        window.htmx.swap(el, html, { swapStyle: swap });
    }
}
