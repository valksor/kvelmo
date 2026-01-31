/**
 * Dropdown Components Module
 *
 * Handles dropdown menus, mobile menu, and the "More" navigation dropdown.
 * Uses event delegation for robust handling across HTMX swaps.
 *
 * @module components/dropdowns
 */

let dropdownInitialized = false;
let mobileMenuInitialized = false;

/**
 * Initialize all dropdown components.
 * Uses event delegation so we don't need to reinitialize after HTMX swaps.
 */
export function initDropdowns() {
    if (dropdownInitialized) {
        console.log('[Dropdowns] Already initialized');
        return;
    }
    dropdownInitialized = true;
    console.log('[Dropdowns] Initializing with event delegation');

    // Delegated click handler for dropdown triggers
    document.addEventListener('click', (e) => {
        const trigger = e.target.closest('[data-dropdown-trigger]');

        if (trigger) {
            e.stopPropagation();
            e.preventDefault();

            const dropdown = trigger.closest('[data-dropdown]');
            if (!dropdown) return;

            const menu = dropdown.querySelector('[data-dropdown-menu]');
            if (!menu) return;

            // Close other dropdowns first
            document.querySelectorAll('[data-dropdown-menu]').forEach(m => {
                if (m !== menu) {
                    m.classList.add('hidden');
                    // Reset chevron on other dropdowns
                    const otherDropdown = m.closest('[data-dropdown]');
                    const otherChevron = otherDropdown?.querySelector('[data-dropdown-chevron]');
                    if (otherChevron) {
                        otherChevron.classList.remove('rotate-180');
                    }
                }
            });

            // Toggle this dropdown
            const isOpening = menu.classList.toggle('hidden');

            // Rotate chevron if present
            const chevron = dropdown.querySelector('[data-dropdown-chevron]');
            if (chevron) {
                if (isOpening) {
                    chevron.classList.remove('rotate-180');
                } else {
                    chevron.classList.add('rotate-180');
                }
            }
            return;
        }

        // Click outside - close all dropdowns
        document.querySelectorAll('[data-dropdown-menu]').forEach(menu => {
            menu.classList.add('hidden');
        });
        document.querySelectorAll('[data-dropdown-chevron]').forEach(chevron => {
            chevron.classList.remove('rotate-180');
        });
    });

    // Close on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            document.querySelectorAll('[data-dropdown-menu]').forEach(menu => {
                menu.classList.add('hidden');
            });
            document.querySelectorAll('[data-dropdown-chevron]').forEach(chevron => {
                chevron.classList.remove('rotate-180');
            });
        }
    });
}

/**
 * Initialize the "More" navigation dropdown.
 * This is a no-op since we now use data-dropdown attributes and event delegation.
 * Kept for backwards compatibility.
 */
export function initMoreMenu() {
    // No-op: "More" menu now uses data-dropdown pattern and is handled by initDropdowns()
}

/**
 * Initialize the mobile menu toggle.
 * Uses event delegation for robustness.
 */
export function initMobileMenu() {
    if (mobileMenuInitialized) {
        return;
    }
    mobileMenuInitialized = true;

    document.addEventListener('click', (e) => {
        const button = e.target.closest('#mobile-menu-button');
        if (button) {
            const menu = document.getElementById('mobile-menu');
            if (menu) {
                menu.classList.toggle('hidden');
            }
        }
    });
}

/**
 * Close all open dropdown menus.
 */
export function closeAllDropdowns() {
    // Close data-dropdown menus
    document.querySelectorAll('[data-dropdown-menu]').forEach(menu => {
        menu.classList.add('hidden');
    });
    document.querySelectorAll('[data-dropdown-chevron]').forEach(chevron => {
        chevron.classList.remove('rotate-180');
    });

    // Close mobile menu
    const mobileMenu = document.getElementById('mobile-menu');
    if (mobileMenu) {
        mobileMenu.classList.add('hidden');
    }
}
