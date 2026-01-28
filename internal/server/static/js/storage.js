/**
 * Local Storage Module
 *
 * Utilities for persistent storage of user preferences.
 *
 * @module storage
 */

const PROJECT_KEY = 'mehr-selected-project';
const THEME_KEY = 'color-theme';
const NOTIFICATION_PREF_KEY = 'mehr_notification_dismissed';

/**
 * Get the currently selected project ID from localStorage.
 *
 * @returns {string | null} The project ID or null if not set
 */
export function getSelectedProject() {
    return localStorage.getItem(PROJECT_KEY);
}

/**
 * Set the selected project ID in localStorage.
 *
 * @param {string} projectId - The project ID to store
 */
export function setSelectedProject(projectId) {
    localStorage.setItem(PROJECT_KEY, projectId);
}

/**
 * Clear the selected project ID from localStorage.
 */
export function clearSelectedProject() {
    localStorage.removeItem(PROJECT_KEY);
}

/**
 * Get the stored theme preference.
 *
 * @returns {'light' | 'dark' | null} The theme preference or null if not set
 */
export function getStoredTheme() {
    return localStorage.getItem(THEME_KEY);
}

/**
 * Set the theme preference in localStorage.
 *
 * @param {'light' | 'dark'} theme - The theme to store
 */
export function setStoredTheme(theme) {
    localStorage.setItem(THEME_KEY, theme);
}

/**
 * Check if the user has dismissed the notification permission prompt.
 *
 * @returns {boolean} True if dismissed
 */
export function isNotificationPromptDismissed() {
    return localStorage.getItem(NOTIFICATION_PREF_KEY) === 'true';
}

/**
 * Mark the notification permission prompt as dismissed.
 */
export function dismissNotificationPrompt() {
    localStorage.setItem(NOTIFICATION_PREF_KEY, 'true');
}

/**
 * Get a value from localStorage with JSON parsing.
 *
 * @param {string} key - The storage key
 * @param {any} [defaultValue=null] - Default value if not found
 * @returns {any} The stored value or default
 */
export function getJSON(key, defaultValue = null) {
    const value = localStorage.getItem(key);
    if (value === null) return defaultValue;

    try {
        return JSON.parse(value);
    } catch {
        return defaultValue;
    }
}

/**
 * Set a value in localStorage with JSON stringification.
 *
 * @param {string} key - The storage key
 * @param {any} value - The value to store
 */
export function setJSON(key, value) {
    localStorage.setItem(key, JSON.stringify(value));
}

/**
 * Remove a value from localStorage.
 *
 * @param {string} key - The storage key
 */
export function remove(key) {
    localStorage.removeItem(key);
}

/**
 * Clear all Mehrhof-related storage.
 * Use with caution.
 */
export function clearAll() {
    localStorage.removeItem(PROJECT_KEY);
    localStorage.removeItem(THEME_KEY);
    localStorage.removeItem(NOTIFICATION_PREF_KEY);
}

// Expose project functions globally for backward compatibility
if (typeof window !== 'undefined') {
    window.getSelectedProject = getSelectedProject;
    window.setSelectedProject = setSelectedProject;
    window.clearSelectedProject = clearSelectedProject;
}
