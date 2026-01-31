/**
 * Theme Management Module
 *
 * Handles dark/light mode toggling with localStorage persistence
 * and system preference detection.
 *
 * @module theme
 */

const THEME_KEY = 'color-theme';

/**
 * Update theme toggle button icons based on current theme state.
 * Shows the light icon (sun) when in dark mode, and dark icon (moon) when in light mode.
 */
export function updateThemeIcons() {
    const isDark = document.documentElement.classList.contains('dark');

    // Desktop icons
    const lightIcon = document.getElementById('theme-toggle-light-icon');
    const darkIcon = document.getElementById('theme-toggle-dark-icon');

    // Mobile icons
    const lightIconMobile = document.getElementById('theme-toggle-light-icon-mobile');
    const darkIconMobile = document.getElementById('theme-toggle-dark-icon-mobile');

    // Desktop: show sun in dark mode (to switch to light), moon in light mode (to switch to dark)
    if (lightIcon) lightIcon.classList.toggle('hidden', !isDark);
    if (darkIcon) darkIcon.classList.toggle('hidden', isDark);

    // Mobile: same logic
    if (lightIconMobile) lightIconMobile.classList.toggle('hidden', !isDark);
    if (darkIconMobile) darkIconMobile.classList.toggle('hidden', isDark);
}

/**
 * Initialize theme on page load.
 * Defaults to light mode; only uses dark if explicitly chosen by the user.
 */
export function initTheme() {
    const storedTheme = localStorage.getItem(THEME_KEY);

    // Default to light mode, only use dark if explicitly chosen
    if (storedTheme === 'dark') {
        document.documentElement.classList.add('dark');
    } else {
        document.documentElement.classList.remove('dark');
        if (!storedTheme) {
            localStorage.setItem(THEME_KEY, 'light');
        }
    }

    updateThemeIcons();
    setupThemeListeners();
}

/**
 * Toggle between light and dark theme.
 * Updates DOM classes, localStorage, and icon visibility.
 */
export function toggleTheme() {
    const isDark = document.documentElement.classList.contains('dark');

    if (isDark) {
        document.documentElement.classList.remove('dark');
        localStorage.setItem(THEME_KEY, 'light');
    } else {
        document.documentElement.classList.add('dark');
        localStorage.setItem(THEME_KEY, 'dark');
    }

    updateThemeIcons();
}

/**
 * Set up event listeners for theme toggle buttons.
 * Uses both direct listeners and event delegation for robustness.
 */
function setupThemeListeners() {
    // Direct listeners for known buttons
    const desktopBtn = document.getElementById('theme-toggle');
    const mobileBtn = document.getElementById('theme-toggle-mobile');

    const handleClick = (e) => {
        e.preventDefault();
        e.stopPropagation();
        toggleTheme();
    };

    if (desktopBtn) {
        desktopBtn.addEventListener('click', handleClick);
    }

    if (mobileBtn) {
        mobileBtn.addEventListener('click', handleClick);
    }

    // Event delegation fallback for dynamically added buttons
    document.body.addEventListener('click', (e) => {
        const btn = e.target.closest('#theme-toggle, #theme-toggle-mobile');
        if (btn) {
            e.preventDefault();
            toggleTheme();
        }
    });
}

/**
 * Get the current theme.
 * @returns {'light' | 'dark'} The current theme
 */
export function getCurrentTheme() {
    return document.documentElement.classList.contains('dark') ? 'dark' : 'light';
}

/**
 * Set the theme explicitly.
 * @param {'light' | 'dark'} theme - The theme to set
 */
export function setTheme(theme) {
    if (theme === 'dark') {
        document.documentElement.classList.add('dark');
    } else {
        document.documentElement.classList.remove('dark');
    }
    localStorage.setItem(THEME_KEY, theme);
    updateThemeIcons();
}
