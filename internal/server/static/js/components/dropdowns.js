/**
 * Dropdown Components Module
 *
 * Handles dropdown menus, mobile menu, and the "More" navigation dropdown.
 *
 * @module components/dropdowns
 */

// Store cleanup functions to prevent listener accumulation
let moreMenuCleanup = null;

/**
 * Initialize all dropdown components.
 */
export function initDropdowns() {
    setupDropdownHandlers();
}

/**
 * Set up generic dropdown handlers using data attributes.
 */
function setupDropdownHandlers() {
    document.querySelectorAll('[data-dropdown]').forEach(dropdown => {
        const trigger = dropdown.querySelector('[data-dropdown-trigger]');
        const menu = dropdown.querySelector('[data-dropdown-menu]');

        if (!trigger || !menu) return;

        trigger.addEventListener('click', (e) => {
            e.stopPropagation();
            menu.classList.toggle('hidden');
        });
    });

    // Close all dropdowns on outside click
    document.addEventListener('click', () => {
        document.querySelectorAll('[data-dropdown-menu]').forEach(menu => {
            menu.classList.add('hidden');
        });
    });

    // Close on Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            document.querySelectorAll('[data-dropdown-menu]').forEach(menu => {
                menu.classList.add('hidden');
            });
        }
    });
}

/**
 * Initialize the "More" navigation dropdown.
 * Re-queries DOM elements each time to get fresh references after HTMX swaps.
 */
export function initMoreMenu() {
    // Clean up previous listeners
    if (moreMenuCleanup) {
        moreMenuCleanup();
        moreMenuCleanup = null;
    }

    const btn = document.getElementById('more-menu-btn');
    const menu = document.getElementById('more-menu');
    const chevron = document.getElementById('more-menu-chevron');

    // Only initialize if the More menu exists
    if (!btn || !menu) return;

    // Button click handler
    const handleBtnClick = (e) => {
        e.stopPropagation();
        e.preventDefault();
        menu.classList.toggle('hidden');
        if (chevron) {
            chevron.classList.toggle('rotate-180');
        }
    };

    // Outside click handler
    const handleOutsideClick = (e) => {
        const currentMenu = document.getElementById('more-menu');
        const currentBtn = document.getElementById('more-menu-btn');
        const currentChevron = document.getElementById('more-menu-chevron');

        if (currentMenu && !currentMenu.classList.contains('hidden')) {
            if (!currentBtn || !currentBtn.contains(e.target)) {
                currentMenu.classList.add('hidden');
                if (currentChevron) {
                    currentChevron.classList.remove('rotate-180');
                }
            }
        }
    };

    // Escape key handler
    const handleEscape = (e) => {
        if (e.key === 'Escape') {
            const currentMenu = document.getElementById('more-menu');
            const currentChevron = document.getElementById('more-menu-chevron');

            if (currentMenu && !currentMenu.classList.contains('hidden')) {
                currentMenu.classList.add('hidden');
                if (currentChevron) {
                    currentChevron.classList.remove('rotate-180');
                }
            }
        }
    };

    // Add listeners
    btn.addEventListener('click', handleBtnClick);
    document.addEventListener('click', handleOutsideClick);
    document.addEventListener('keydown', handleEscape);

    // Store cleanup function
    moreMenuCleanup = () => {
        btn.removeEventListener('click', handleBtnClick);
        document.removeEventListener('click', handleOutsideClick);
        document.removeEventListener('keydown', handleEscape);
    };
}

/**
 * Initialize the mobile menu toggle.
 */
export function initMobileMenu() {
    const button = document.getElementById('mobile-menu-button');
    const menu = document.getElementById('mobile-menu');

    if (!button || !menu) return;

    button.addEventListener('click', () => {
        menu.classList.toggle('hidden');
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

    // Close More menu
    const moreMenu = document.getElementById('more-menu');
    const moreChevron = document.getElementById('more-menu-chevron');
    if (moreMenu) {
        moreMenu.classList.add('hidden');
    }
    if (moreChevron) {
        moreChevron.classList.remove('rotate-180');
    }

    // Close mobile menu
    const mobileMenu = document.getElementById('mobile-menu');
    if (mobileMenu) {
        mobileMenu.classList.add('hidden');
    }
}
