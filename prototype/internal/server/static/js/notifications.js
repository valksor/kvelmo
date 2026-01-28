/**
 * Notification System Module
 *
 * Handles toast notifications, notification center, and browser notifications.
 * Uses safe DOM manipulation (no innerHTML with user data).
 *
 * @module notifications
 */

const NOTIFICATION_PREF_KEY = 'mehr_notification_dismissed';
const MAX_NOTIFICATIONS = 50;

// Notification storage
let notifications = [];
let unreadCount = 0;

// Notification type configurations
const notificationTypes = {
    success: {
        bgColor: 'bg-success-50 dark:bg-success-900/30',
        textColor: 'text-success-800 dark:text-success-300',
        borderColor: 'border-success-200 dark:border-success-800',
        icon: '✓'
    },
    error: {
        bgColor: 'bg-error-50 dark:bg-error-900/30',
        textColor: 'text-error-800 dark:text-error-300',
        borderColor: 'border-error-200 dark:border-error-800',
        icon: '✕'
    },
    warning: {
        bgColor: 'bg-warning-50 dark:bg-warning-900/30',
        textColor: 'text-warning-800 dark:text-warning-300',
        borderColor: 'border-warning-200 dark:border-warning-800',
        icon: '⚠'
    },
    info: {
        bgColor: 'bg-info-50 dark:bg-info-900/30',
        textColor: 'text-info-800 dark:text-info-300',
        borderColor: 'border-info-200 dark:border-info-800',
        icon: 'ℹ'
    }
};

/**
 * Create a close button SVG icon element safely.
 * @returns {SVGElement} The SVG element
 */
function createCloseIcon() {
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('class', 'w-4 h-4');
    svg.setAttribute('fill', 'currentColor');
    svg.setAttribute('viewBox', '0 0 20 20');

    const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('fill-rule', 'evenodd');
    path.setAttribute('d', 'M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z');
    path.setAttribute('clip-rule', 'evenodd');

    svg.appendChild(path);
    return svg;
}

/**
 * Initialize the notification system.
 * Sets up event listeners for notification center toggle.
 */
export function initNotifications() {
    const bell = document.getElementById('notification-bell');
    const center = document.getElementById('notification-center');
    const closeBtn = document.getElementById('close-notifications');

    if (bell && center) {
        bell.addEventListener('click', () => {
            center.classList.toggle('hidden');
            if (!center.classList.contains('hidden')) {
                markAllAsRead();
            }
        });
    }

    if (closeBtn && center) {
        closeBtn.addEventListener('click', () => {
            center.classList.add('hidden');
        });
    }

    // Close when clicking outside
    document.addEventListener('click', (e) => {
        if (center && bell &&
            !center.classList.contains('hidden') &&
            !center.contains(e.target) &&
            !bell.contains(e.target)) {
            center.classList.add('hidden');
        }
    });

    // Request browser notification permission after delay
    requestBrowserNotificationPermission();
}

/**
 * Show a toast notification with animation and auto-dismiss.
 *
 * @param {string} message - The notification message to display
 * @param {'success' | 'error' | 'warning' | 'info'} [type='info'] - The notification type
 * @param {number} [duration=5000] - Auto-dismiss duration in ms (0 for no auto-dismiss)
 * @returns {HTMLElement} The created toast element
 */
export function showToast(message, type = 'info', duration = 5000) {
    const container = document.getElementById('toast-container');
    if (!container) {
        console.warn('[Notifications] Toast container not found');
        return null;
    }

    const config = notificationTypes[type] || notificationTypes.info;

    // Build toast using safe DOM methods (no innerHTML with user data)
    const toast = document.createElement('div');
    toast.className = `${config.bgColor} ${config.textColor} ${config.borderColor} border rounded-xl shadow-lg p-4 transition-all duration-300 translate-x-full opacity-0 pointer-events-auto`;

    const content = document.createElement('div');
    content.className = 'flex items-start gap-3';

    // Icon (safe - hardcoded)
    const iconSpan = document.createElement('span');
    iconSpan.className = 'text-lg flex-shrink-0';
    iconSpan.textContent = config.icon;
    content.appendChild(iconSpan);

    // Message (safe - uses textContent)
    const messageDiv = document.createElement('div');
    messageDiv.className = 'flex-1 text-sm font-medium';
    messageDiv.textContent = message;
    content.appendChild(messageDiv);

    // Close button (safe - SVG built with DOM methods)
    const closeBtn = document.createElement('button');
    closeBtn.className = 'flex-shrink-0 p-1 rounded-lg hover:bg-black/5 dark:hover:bg-white/10 transition-smooth';
    closeBtn.setAttribute('aria-label', 'Close notification');
    closeBtn.appendChild(createCloseIcon());
    closeBtn.addEventListener('click', () => toast.remove());
    content.appendChild(closeBtn);

    toast.appendChild(content);
    container.appendChild(toast);

    // Animate in
    requestAnimationFrame(() => {
        toast.classList.remove('translate-x-full', 'opacity-0');
    });

    // Auto-dismiss
    if (duration > 0) {
        setTimeout(() => {
            toast.classList.add('translate-x-full', 'opacity-0');
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }

    // Add to notification center
    addNotification(message, type);

    return toast;
}

/**
 * Add a notification to the notification center.
 *
 * @param {string} message - The notification message
 * @param {'success' | 'error' | 'warning' | 'info'} [type='info'] - The notification type
 */
function addNotification(message, type = 'info') {
    const notification = {
        id: Date.now(),
        message,
        type,
        timestamp: new Date(),
        read: false
    };

    notifications.unshift(notification);
    if (notifications.length > MAX_NOTIFICATIONS) {
        notifications.pop();
    }

    unreadCount++;
    updateNotificationBadge();
    renderNotificationCenter();
}

/**
 * Update the notification badge count.
 */
function updateNotificationBadge() {
    const badge = document.getElementById('notification-badge');
    if (!badge) return;

    if (unreadCount > 0) {
        badge.textContent = unreadCount > 9 ? '9+' : String(unreadCount);
        badge.classList.remove('hidden');
    } else {
        badge.classList.add('hidden');
    }
}

/**
 * Render the notification center list.
 * Uses safe DOM manipulation (no innerHTML with user data).
 */
function renderNotificationCenter() {
    const list = document.getElementById('notification-list');
    if (!list) return;

    // Clear existing content
    list.replaceChildren();

    if (notifications.length === 0) {
        const empty = document.createElement('p');
        empty.className = 'px-3 py-8 text-center text-sm text-surface-500 dark:text-surface-400';
        empty.textContent = 'No notifications';
        list.appendChild(empty);
        return;
    }

    // Build notification items safely
    notifications.forEach(notif => {
        const config = notificationTypes[notif.type] || notificationTypes.info;

        const item = document.createElement('div');
        item.className = `px-3 py-3 rounded-xl hover:bg-surface-100 dark:hover:bg-surface-700 cursor-pointer transition-smooth ${notif.read ? 'opacity-60' : ''}`;
        item.addEventListener('click', () => markAsRead(notif.id));

        const content = document.createElement('div');
        content.className = 'flex items-start gap-3';

        // Icon
        const icon = document.createElement('span');
        icon.className = 'text-sm flex-shrink-0 mt-0.5';
        icon.textContent = config.icon;
        content.appendChild(icon);

        // Text content
        const textWrapper = document.createElement('div');
        textWrapper.className = 'flex-1 min-w-0';

        const message = document.createElement('p');
        message.className = `text-sm font-medium text-surface-900 dark:text-surface-100 ${notif.read ? '' : 'font-bold'}`;
        message.textContent = notif.message; // Safe - uses textContent
        textWrapper.appendChild(message);

        const time = document.createElement('p');
        time.className = 'text-xs text-surface-500 dark:text-surface-400 mt-1';
        time.textContent = formatTimeAgo(notif.timestamp);
        textWrapper.appendChild(time);

        content.appendChild(textWrapper);
        item.appendChild(content);
        list.appendChild(item);
    });
}

/**
 * Mark a notification as read by ID.
 *
 * @param {number} id - The notification ID
 */
function markAsRead(id) {
    const notif = notifications.find(n => n.id === id);
    if (notif && !notif.read) {
        notif.read = true;
        unreadCount--;
        updateNotificationBadge();
        renderNotificationCenter();
    }
}

/**
 * Mark all notifications as read.
 */
function markAllAsRead() {
    notifications.forEach(n => { n.read = true; });
    unreadCount = 0;
    updateNotificationBadge();
    renderNotificationCenter();
}

/**
 * Format a date as a relative time string.
 *
 * @param {Date} date - The date to format
 * @returns {string} The formatted relative time string
 */
function formatTimeAgo(date) {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (seconds < 60) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
}

/**
 * Request browser notification permission.
 * Shows a toast asking the user to enable notifications.
 */
function requestBrowserNotificationPermission() {
    if (!('Notification' in window)) return;
    if (Notification.permission === 'granted') return;
    if (Notification.permission === 'denied') return;
    if (localStorage.getItem(NOTIFICATION_PREF_KEY) === 'true') return;

    // Show permission prompt after delay
    setTimeout(() => {
        const toast = showToast('Enable browser notifications for task updates?', 'info', 0);
        if (!toast) return;

        const messageDiv = toast.querySelector('.text-sm.font-medium');
        if (!messageDiv) return;

        const buttonContainer = document.createElement('div');
        buttonContainer.className = 'flex items-center gap-2 mt-2';

        // Enable button
        const enableBtn = document.createElement('button');
        enableBtn.className = 'px-4 py-2 bg-gradient-to-r from-brand-600 to-violet-600 text-white text-sm rounded-xl font-semibold shadow-brand hover:shadow-brand-lg transition-all duration-200 hover:scale-105';
        enableBtn.textContent = 'Enable';
        enableBtn.addEventListener('click', () => {
            Notification.requestPermission().then(permission => {
                if (permission === 'granted') {
                    showToast('Browser notifications enabled!', 'success');
                }
                if (permission === 'denied') {
                    localStorage.setItem(NOTIFICATION_PREF_KEY, 'true');
                }
                toast.remove();
            });
        });

        // Dismiss button
        const dismissBtn = document.createElement('button');
        dismissBtn.className = 'px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 font-medium transition-all duration-200';
        dismissBtn.textContent = 'Not now';
        dismissBtn.addEventListener('click', () => {
            localStorage.setItem(NOTIFICATION_PREF_KEY, 'true');
            toast.remove();
        });

        buttonContainer.appendChild(enableBtn);
        buttonContainer.appendChild(dismissBtn);
        messageDiv.appendChild(buttonContainer);
    }, 5000);
}

/**
 * Show a native browser notification.
 *
 * @param {string} title - The notification title
 * @param {string} body - The notification body text
 * @param {'success' | 'error' | 'warning' | 'info'} [type='info'] - The notification type
 */
export function showBrowserNotification(title, body, type = 'info') {
    if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(title, { body });
    }
}
