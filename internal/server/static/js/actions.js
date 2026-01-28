/**
 * Mehrhof Web UI - Action Handler Module
 *
 * Uses event delegation to handle all user interactions via data attributes.
 * This eliminates inline onclick handlers and centralizes action handling.
 *
 * Usage in templates:
 *   <button data-action="show-tab" data-tab="settings">Settings</button>
 *   <button data-action="load-data" data-endpoint="/api/items">Load</button>
 *
 * @module actions
 */

import { showToast } from './notifications.js';
import { getSelectedProject, clearSelectedProject } from './storage.js';

// Action handler registry
const handlers = new Map();

// Page-specific state (populated by page scripts)
const pageState = {
    agentLogsEventSource: null,
    agentLogsExpanded: true
};

/**
 * Register an action handler.
 * @param {string} action - The action name (matches data-action attribute)
 * @param {function} handler - Handler function receives (element, event)
 */
export function registerAction(action, handler) {
    handlers.set(action, handler);
}

/**
 * Register multiple action handlers at once.
 * @param {Object} actions - Map of action names to handlers
 */
export function registerActions(actions) {
    for (const [action, handler] of Object.entries(actions)) {
        handlers.set(action, handler);
    }
}

/**
 * Initialize action handling with event delegation.
 * Call once on DOMContentLoaded.
 */
export function initActions() {
    // Single delegated click handler for all actions
    document.body.addEventListener('click', handleClick);

    // Handle change events for inputs with actions
    document.body.addEventListener('change', handleChange);

    // Register built-in actions
    registerBuiltinActions();

    console.log('[Actions] Initialized with event delegation');
}

/**
 * Delegated click handler - finds closest [data-action] and executes.
 */
function handleClick(event) {
    const actionEl = event.target.closest('[data-action]');
    if (!actionEl) return;

    const action = actionEl.dataset.action;
    const handler = handlers.get(action);

    if (handler) {
        // Prevent default for buttons/links with actions
        if (actionEl.tagName === 'BUTTON' || actionEl.tagName === 'A') {
            event.preventDefault();
        }

        try {
            handler(actionEl, event);
        } catch (err) {
            console.error(`[Actions] Error in handler for "${action}":`, err);
            showToast(`Action failed: ${err.message}`, 'error');
        }
    } else {
        console.warn(`[Actions] No handler registered for action: ${action}`);
    }
}

/**
 * Delegated change handler for inputs with actions.
 */
function handleChange(event) {
    const actionEl = event.target.closest('[data-action-change]');
    if (!actionEl) return;

    const action = actionEl.dataset.actionChange;
    const handler = handlers.get(action);

    if (handler) {
        try {
            handler(actionEl, event);
        } catch (err) {
            console.error(`[Actions] Error in change handler for "${action}":`, err);
        }
    }
}

/**
 * Register all built-in actions.
 */
function registerBuiltinActions() {
    registerActions({
        // ─────────────────────────────────────────────────────────────────
        // Tab/Section Navigation
        // ─────────────────────────────────────────────────────────────────

        'show-section': (el) => {
            const sectionId = el.dataset.section;
            showSection(sectionId);
        },

        'show-tab': (el) => {
            const tabName = el.dataset.tab;
            const prefix = el.dataset.prefix || '';
            showTab(tabName, prefix);
        },

        'show-project-tab': (el) => {
            const tabName = el.dataset.tab;
            showProjectTab(tabName);
        },

        // ─────────────────────────────────────────────────────────────────
        // Data Loading
        // ─────────────────────────────────────────────────────────────────

        'load-tasks': () => {
            if (typeof window.loadTasks === 'function') {
                window.loadTasks();
            }
        },

        'load-queues': () => {
            if (typeof window.loadQueues === 'function') {
                window.loadQueues();
            }
        },

        'load-agents': () => {
            if (typeof window.loadAgentsList === 'function') {
                window.loadAgentsList();
            }
        },

        'load-providers': () => {
            if (typeof window.loadProvidersList === 'function') {
                window.loadProvidersList();
            }
        },

        'load-licenses': () => {
            if (typeof window.loadLicenses === 'function') {
                window.loadLicenses();
            }
        },

        'apply-filters': () => {
            if (typeof window.applyFilters === 'function') {
                window.applyFilters();
            }
        },

        // ─────────────────────────────────────────────────────────────────
        // Toggle/Expand
        // ─────────────────────────────────────────────────────────────────

        'toggle-spec': (el) => {
            const specNumber = el.dataset.spec;
            toggleSpecificationDetail(specNumber);
        },

        'toggle-agent-logs': () => {
            toggleAgentLogs();
        },

        // ─────────────────────────────────────────────────────────────────
        // Clipboard
        // ─────────────────────────────────────────────────────────────────

        'copy-spec': (el) => {
            const specNumber = el.dataset.spec;
            copySpecificationContent(specNumber);
        },

        // ─────────────────────────────────────────────────────────────────
        // Agent Logs
        // ─────────────────────────────────────────────────────────────────

        'clear-agent-logs': () => {
            clearAgentLogs();
        },

        // ─────────────────────────────────────────────────────────────────
        // File Upload
        // ─────────────────────────────────────────────────────────────────

        'reset-upload': () => {
            if (typeof window.resetUpload === 'function') {
                window.resetUpload();
            }
        },

        'trigger-file-input': (el) => {
            const inputId = el.dataset.input || 'file-input';
            document.getElementById(inputId)?.click();
        },

        // ─────────────────────────────────────────────────────────────────
        // Forms
        // ─────────────────────────────────────────────────────────────────

        'show-alias-form': () => {
            if (typeof window.showAliasForm === 'function') {
                window.showAliasForm();
            }
        },

        'close-modal': (el) => {
            const modalId = el.dataset.modal;
            if (modalId) {
                document.getElementById(modalId)?.classList.add('hidden');
            } else if (typeof window.closeEditModal === 'function') {
                window.closeEditModal();
            }
        },

        // ─────────────────────────────────────────────────────────────────
        // Navigation
        // ─────────────────────────────────────────────────────────────────

        'go-to-project-settings': () => {
            const project = getSelectedProject();
            if (project) {
                window.location.href = '/settings?project=' + encodeURIComponent(project);
            }
        },

        'clear-saved-project': () => {
            clearSelectedProject();
            document.getElementById('saved-project-link')?.classList.add('hidden');
        },

        // ─────────────────────────────────────────────────────────────────
        // Project Operations
        // ─────────────────────────────────────────────────────────────────

        'view-queue': (el) => {
            const queueId = el.dataset.queue;
            if (typeof window.viewQueue === 'function') {
                window.viewQueue(queueId);
            }
        },

        'delete-queue': (el) => {
            const queueId = el.dataset.queue;
            if (typeof window.deleteQueue === 'function') {
                window.deleteQueue(queueId);
            }
        },

        'edit-task': (el) => {
            const taskId = el.dataset.task;
            if (typeof window.editTask === 'function') {
                window.editTask(taskId);
            }
        },

        'auto-reorder': () => {
            if (typeof window.autoReorder === 'function') {
                window.autoReorder();
            }
        },

        'submit-tasks': (el) => {
            const dryRun = el.dataset.dryRun === 'true';
            if (typeof window.submitTasks === 'function') {
                window.submitTasks(dryRun);
            }
        },

        'start-implementation': () => {
            if (typeof window.startImplementation === 'function') {
                window.startImplementation();
            }
        },

        'show-graph': (el) => {
            const viewType = el.dataset.view;
            if (typeof window.showGraphVisualization === 'function') {
                window.showGraphVisualization(viewType);
            }
        }
    });
}

// ─────────────────────────────────────────────────────────────────────────────
// Built-in Action Implementations
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Show a settings section (settings page navigation).
 */
function showSection(sectionId) {
    // Hide all sections
    document.querySelectorAll('.settings-section').forEach(s => s.classList.add('hidden'));

    // Deactivate all nav items
    document.querySelectorAll('.nav-section').forEach(n => {
        n.classList.remove('bg-brand-50', 'dark:bg-brand-900/20', 'text-brand-700', 'dark:text-brand-300');
        n.classList.add('text-surface-600', 'dark:text-surface-400', 'hover:bg-surface-100', 'dark:hover:bg-surface-800');
    });

    // Show selected section
    const section = document.getElementById('section-' + sectionId);
    if (section) {
        section.classList.remove('hidden');
    }

    // Activate selected nav
    const nav = document.getElementById('nav-' + sectionId);
    if (nav) {
        nav.classList.remove('text-surface-600', 'dark:text-surface-400', 'hover:bg-surface-100', 'dark:hover:bg-surface-800');
        nav.classList.add('bg-brand-50', 'dark:bg-brand-900/20', 'text-brand-700', 'dark:text-brand-300');
    }
}

/**
 * Show a tab panel (generic tab switching).
 * @param {string} tabName - Tab identifier
 * @param {string} prefix - Optional prefix for panel/tab IDs (e.g., 'project-')
 */
function showTab(tabName, prefix = '') {
    const panelSelector = prefix ? `.${prefix}panel` : '.tab-panel';
    const tabSelector = prefix ? `.${prefix}tab` : '.tab-btn';

    // Hide all panels
    document.querySelectorAll(panelSelector).forEach(p => p.classList.add('hidden'));

    // Deactivate all tabs
    document.querySelectorAll(tabSelector).forEach(t => {
        t.classList.remove('bg-white', 'dark:bg-surface-700', 'text-brand-600', 'dark:text-brand-400', 'shadow-sm');
        t.classList.add('text-surface-500', 'dark:text-surface-400');
    });

    // Show selected panel
    const panelId = prefix ? `${prefix}panel-${tabName}` : `panel-${tabName}`;
    const panel = document.getElementById(panelId);
    if (panel) {
        panel.classList.remove('hidden');
    }

    // Activate selected tab
    const tabId = prefix ? `${prefix}tab-${tabName}` : `tab-${tabName}`;
    const tab = document.getElementById(tabId);
    if (tab) {
        tab.classList.remove('text-surface-500', 'dark:text-surface-400');
        tab.classList.add('bg-white', 'dark:bg-surface-700', 'text-brand-600', 'dark:text-brand-400', 'shadow-sm');
    }
}

/**
 * Show a project tab (project page specific).
 */
function showProjectTab(tabName) {
    document.querySelectorAll('.project-panel').forEach(p => p.classList.add('hidden'));
    document.querySelectorAll('.project-tab').forEach(t => {
        t.classList.remove('bg-white', 'dark:bg-surface-700', 'text-brand-600', 'dark:text-brand-400', 'shadow-sm');
        t.classList.add('text-surface-500', 'dark:text-surface-400');
    });

    const panel = document.getElementById('project-panel-' + tabName);
    if (panel) {
        panel.classList.remove('hidden');
    }

    const tab = document.getElementById('project-tab-' + tabName);
    if (tab) {
        tab.classList.remove('text-surface-500', 'dark:text-surface-400');
        tab.classList.add('bg-white', 'dark:bg-surface-700', 'text-brand-600', 'dark:text-brand-400', 'shadow-sm');
    }

    // Trigger data loading for specific tabs
    if (tabName === 'queues' && typeof window.loadQueues === 'function') {
        window.loadQueues();
    }
    if (tabName === 'tasks' && typeof window.loadQueueSelector === 'function') {
        window.loadQueueSelector();
    }
}

/**
 * Toggle specification detail expansion.
 * Matches IDs from specification.html template:
 *   specification-detail-{number}, specification-chevron-{number}
 */
function toggleSpecificationDetail(specNumber) {
    const detail = document.getElementById(`specification-detail-${specNumber}`);
    const chevron = document.getElementById(`specification-chevron-${specNumber}`);
    const button = chevron?.closest('button');

    if (detail && chevron) {
        const isHidden = detail.classList.contains('hidden');
        detail.classList.toggle('hidden');
        chevron.classList.toggle('rotate-180');

        // Update ARIA state for accessibility
        if (button) {
            button.setAttribute('aria-expanded', isHidden ? 'true' : 'false');
        }
    }
}

/**
 * Copy specification content to clipboard.
 * Finds the <pre> element inside the specification detail section.
 */
async function copySpecificationContent(specNumber) {
    const detail = document.getElementById(`specification-detail-${specNumber}`);
    const pre = detail?.querySelector('pre');
    if (!pre) return;

    try {
        await navigator.clipboard.writeText(pre.textContent);
        showToast('Specification content copied to clipboard', 'success');
    } catch (err) {
        console.error('Failed to copy:', err);
        showToast('Failed to copy content', 'error');
    }
}

/**
 * Toggle agent logs panel expansion.
 */
function toggleAgentLogs() {
    const container = document.getElementById('agent-logs-container');
    const btn = document.getElementById('toggle-logs-btn');

    if (!container || !btn) return;

    if (pageState.agentLogsExpanded) {
        container.style.maxHeight = '60px';
        btn.textContent = '▶';
        pageState.agentLogsExpanded = false;
    } else {
        container.style.maxHeight = '400px';
        btn.textContent = '▼';
        pageState.agentLogsExpanded = true;
    }
}

/**
 * Clear agent logs content.
 */
function clearAgentLogs() {
    const content = document.getElementById('agent-logs-content');
    if (content) {
        content.textContent = '// Agent output will appear here...';
    }
}

// Export page state for complex page scripts that need it
export { pageState };
