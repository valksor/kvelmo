/**
 * Forms Module
 *
 * Form utilities and validation helpers.
 *
 * @module forms
 */

/**
 * Serialize form data to a plain object.
 *
 * @param {HTMLFormElement} form - The form element
 * @returns {Object} Form data as key-value pairs
 */
export function serializeForm(form) {
    const data = {};
    const formData = new FormData(form);

    for (const [key, value] of formData.entries()) {
        // Handle multiple values for same key (checkboxes, multi-selects)
        if (data[key] !== undefined) {
            if (!Array.isArray(data[key])) {
                data[key] = [data[key]];
            }
            data[key].push(value);
        } else {
            data[key] = value;
        }
    }

    return data;
}

/**
 * Set form values from an object.
 *
 * @param {HTMLFormElement} form - The form element
 * @param {Object} data - Object with field values
 */
export function populateForm(form, data) {
    for (const [key, value] of Object.entries(data)) {
        const field = form.elements[key];
        if (!field) continue;

        if (field.type === 'checkbox') {
            field.checked = Boolean(value);
        } else if (field.type === 'radio') {
            // Handle radio button groups
            const radios = form.querySelectorAll(`[name="${key}"]`);
            radios.forEach(radio => {
                radio.checked = radio.value === value;
            });
        } else if (field.tagName === 'SELECT' && field.multiple) {
            // Handle multi-select
            const values = Array.isArray(value) ? value : [value];
            Array.from(field.options).forEach(option => {
                option.selected = values.includes(option.value);
            });
        } else {
            field.value = value || '';
        }
    }
}

/**
 * Add visual validation feedback to a field.
 *
 * @param {HTMLElement} field - The form field
 * @param {boolean} isValid - Whether the field is valid
 * @param {string} [message] - Optional error message
 */
export function setFieldValidation(field, isValid, message) {
    const container = field.closest('.form-group') || field.parentElement;
    const errorEl = container?.querySelector('.field-error');

    if (isValid) {
        field.classList.remove('border-error-500', 'focus:ring-error-500');
        field.classList.add('border-success-500');
        if (errorEl) errorEl.textContent = '';
    } else {
        field.classList.remove('border-success-500');
        field.classList.add('border-error-500', 'focus:ring-error-500');
        if (errorEl && message) {
            errorEl.textContent = message;
        }
    }
}

/**
 * Clear all validation feedback from a form.
 *
 * @param {HTMLFormElement} form - The form element
 */
export function clearFormValidation(form) {
    form.querySelectorAll('input, select, textarea').forEach(field => {
        field.classList.remove('border-error-500', 'border-success-500', 'focus:ring-error-500');
    });

    form.querySelectorAll('.field-error').forEach(el => {
        el.textContent = '';
    });
}

/**
 * Disable all form fields and buttons.
 *
 * @param {HTMLFormElement} form - The form element
 * @param {boolean} disabled - Whether to disable (true) or enable (false)
 */
export function setFormDisabled(form, disabled) {
    form.querySelectorAll('input, select, textarea, button').forEach(el => {
        el.disabled = disabled;
    });
}

/**
 * Show a loading state on a submit button.
 *
 * @param {HTMLButtonElement} button - The button element
 * @param {boolean} loading - Whether to show loading state
 */
export function setButtonLoading(button, loading) {
    if (loading) {
        button.disabled = true;
        button.dataset.originalText = button.textContent;
        button.textContent = 'Loading...';
        button.classList.add('opacity-75', 'cursor-wait');
    } else {
        button.disabled = false;
        button.textContent = button.dataset.originalText || 'Submit';
        button.classList.remove('opacity-75', 'cursor-wait');
    }
}
