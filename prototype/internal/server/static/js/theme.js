/**
 * Theme Management Module
 *
 * Handles light/dark mode toggling with localStorage persistence
 * and system preference detection using DaisyUI themes.
 *
 * @module theme
 */

const THEME_KEY = 'color-theme';
const LIGHT_THEME = 'winter';
const DARK_THEME = 'business';

let listenersInitialized = false;

/**
 * Update theme toggle button icons based on current theme state.
 * Shows the light icon (sun) when in dark mode, and dark icon (moon) when in light mode.
 */
export function updateThemeIcons() {
    const isDark = getCurrentTheme() === 'dark';

    // Find all theme icons using data attributes
    document.querySelectorAll('[data-theme-icon="sun"]').forEach(icon => {
        icon.classList.toggle('hidden', !isDark);
    });
    document.querySelectorAll('[data-theme-icon="moon"]').forEach(icon => {
        icon.classList.toggle('hidden', isDark);
    });
}

/**
 * Initialize theme on page load.
 * Defaults to light mode; respects system preference if no stored preference.
 */
export function initTheme() {
    const storedTheme = localStorage.getItem(THEME_KEY);
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

    let theme;
    if (storedTheme === 'dark' || storedTheme === DARK_THEME) {
        theme = DARK_THEME;
    } else if (storedTheme === 'light' || storedTheme === LIGHT_THEME) {
        theme = LIGHT_THEME;
    } else {
        // No stored preference - use system preference
        theme = prefersDark ? DARK_THEME : LIGHT_THEME;
    }

    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem(THEME_KEY, theme === DARK_THEME ? 'dark' : 'light');

    updateThemeIcons();
    setupThemeListeners();
}

/**
 * Toggle between light and dark theme.
 * Updates DOM data-theme, localStorage, and icon visibility.
 */
export function toggleTheme() {
    const currentTheme = document.documentElement.getAttribute('data-theme');
    const newTheme = currentTheme === DARK_THEME ? LIGHT_THEME : DARK_THEME;

    document.documentElement.setAttribute('data-theme', newTheme);
    localStorage.setItem(THEME_KEY, newTheme === DARK_THEME ? 'dark' : 'light');

    updateThemeIcons();
}

/**
 * Set up event listeners for theme toggle buttons.
 * Uses event delegation for robustness across HTMX swaps.
 */
function setupThemeListeners() {
    // Prevent duplicate listener registration
    if (listenersInitialized) {
        return;
    }
    listenersInitialized = true;

    // Event delegation on document - works even after HTMX swaps
    document.addEventListener('click', (e) => {
        const btn = e.target.closest('#theme-toggle, #theme-toggle-mobile');
        if (btn) {
            e.preventDefault();
            e.stopPropagation();
            toggleTheme();
        }
    });

    // Listen for system preference changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
        // Only update if user hasn't explicitly set a preference
        if (!localStorage.getItem(THEME_KEY)) {
            const newTheme = e.matches ? DARK_THEME : LIGHT_THEME;
            document.documentElement.setAttribute('data-theme', newTheme);
            updateThemeIcons();
        }
    });
}

/**
 * Get the current theme.
 * @returns {'light' | 'dark'} The current theme
 */
export function getCurrentTheme() {
    const dataTheme = document.documentElement.getAttribute('data-theme');
    return dataTheme === DARK_THEME ? 'dark' : 'light';
}

/**
 * Set the theme explicitly.
 * @param {'light' | 'dark'} theme - The theme to set
 */
export function setTheme(theme) {
    const daisyTheme = theme === 'dark' ? DARK_THEME : LIGHT_THEME;
    document.documentElement.setAttribute('data-theme', daisyTheme);
    localStorage.setItem(THEME_KEY, theme);
    updateThemeIcons();
}
