/**
 * Mehrhof Web UI - Main Application Module
 *
 * This is the entry point for the Mehrhof web application. It initializes
 * all UI modules and sets up global state management.
 *
 * @module app
 */

import { initTheme, toggleTheme, updateThemeIcons } from './theme.js';
import { initNotifications, showToast, showBrowserNotification } from './notifications.js';
import { initHTMX } from './htmx-config.js';
import { initSSE, updateConnectionStatus } from './sse-manager.js';
import { initDropdowns, initMoreMenu, initMobileMenu } from './components/dropdowns.js';
import { getSelectedProject, setSelectedProject, clearSelectedProject } from './storage.js';
import { initActions } from './actions.js';
import { fetchCSRFToken, csrfFetch } from './csrf.js';

// Application state
const state = {
    initialized: false
};

/**
 * Initialize the application on DOM ready.
 * Sets up all modules and event listeners.
 */
function init() {
    if (state.initialized) {
        console.log('[App] Already initialized');
        return;
    }

    console.log('[App] Initializing...');

    // Initialize core modules
    initTheme();
    initNotifications();
    initHTMX();
    initSSE();
    initActions();

    // Fetch CSRF token for authenticated sessions (non-blocking)
    fetchCSRFToken();

    // Initialize UI components
    initDropdowns();
    initMoreMenu();
    initMobileMenu();

    state.initialized = true;
    console.log('[App] Initialization complete');
}

/**
 * Reinitialize UI components after HTMX swaps.
 * Called after content is dynamically updated.
 */
function reinitAfterSwap() {
    console.log('[App] Reinitializing after HTMX swap');

    // Theme must be reinitialized to ensure dark class is applied
    initTheme();
    updateThemeIcons();

    // Reinit notifications for navbar bell
    initNotifications();

    // Reinit UI components
    initDropdowns();
    initMoreMenu();
    initMobileMenu();
}

// Initialize on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}

// Reinitialize after HTMX swaps
document.body.addEventListener('htmx:afterSwap', reinitAfterSwap);

// Minimal globals for inline scripts that cannot use ES modules.
// These are used by <script> blocks in templates that need localStorage access.
// All other functions should be imported via ES modules or use data-action delegation.
window.getSelectedProject = getSelectedProject;
window.setSelectedProject = setSelectedProject;
window.clearSelectedProject = clearSelectedProject;
window.csrfFetch = csrfFetch;
window.showToast = showToast;
