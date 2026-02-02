/**
 * Standalone Simplify Module
 *
 * AI-powered standalone code simplification without active task.
 *
 * @module standalone-simplify
 */

(function() {
    'use strict';

    const form = document.getElementById('simplify-form');
    const modeRadios = document.querySelectorAll('input[name="mode"]');
    const branchField = document.getElementById('branch-field');
    const rangeField = document.getElementById('range-field');
    const filesField = document.getElementById('files-field');
    const statusEl = document.getElementById('simplify-status');
    const statusText = document.getElementById('simplify-status-text');
    const resultsEl = document.getElementById('simplify-results');
    const clearBtn = document.getElementById('simplify-clear');
    const cancelBtn = document.getElementById('simplify-cancel');

    let abortController = null;

    // Mode-specific field visibility
    modeRadios.forEach(radio => {
        radio.addEventListener('change', function() {
            branchField.classList.add('hidden');
            rangeField.classList.add('hidden');
            filesField.classList.add('hidden');

            switch(this.value) {
                case 'branch':
                    branchField.classList.remove('hidden');
                    break;
                case 'range':
                    rangeField.classList.remove('hidden');
                    break;
                case 'files':
                    filesField.classList.remove('hidden');
                    break;
            }
        });
    });

    // Form submission
    form?.addEventListener('submit', async function(e) {
        e.preventDefault();

        const formData = new FormData(form);
        const filesValue = formData.get('files');
        const data = {
            mode: formData.get('mode'),
            base_branch: formData.get('base_branch') || '',
            range: formData.get('range') || '',
            files: filesValue ? filesValue.split(',').map(f => f.trim()).filter(f => f) : [],
            context: parseInt(formData.get('context')) || 3,
            agent: formData.get('agent') || '',
            create_checkpoint: formData.get('create_checkpoint') === 'on'
        };

        // Show status
        statusEl.classList.remove('hidden');
        statusText.textContent = 'Running simplification...';

        abortController = new AbortController();

        try {
            const fetchFn = window.csrfFetch || fetch;
            const response = await fetchFn('/api/v1/workflow/simplify/standalone', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data),
                signal: abortController.signal
            });

            const result = await response.json();
            statusEl.classList.add('hidden');
            abortController = null;

            if (result.success) {
                renderResults(result);
            } else {
                renderError(result.error || 'Simplification failed');
            }
        } catch (err) {
            statusEl.classList.add('hidden');
            abortController = null;
            if (err.name !== 'AbortError') {
                renderError(err.message);
            }
        }
    });

    // Clear button
    clearBtn?.addEventListener('click', function() {
        form.reset();
        branchField.classList.add('hidden');
        rangeField.classList.add('hidden');
        filesField.classList.add('hidden');
        showEmptyState();
    });

    // Cancel button
    cancelBtn?.addEventListener('click', function() {
        if (abortController) {
            abortController.abort();
            abortController = null;
        }
        statusEl.classList.add('hidden');
    });

    /**
     * Render the simplification results using safe DOM methods.
     */
    function renderResults(result) {
        const card = document.createElement('div');
        card.className = 'card';

        // Header
        const header = document.createElement('div');
        header.className = 'px-6 py-4 border-b border-surface-200 dark:border-surface-700 flex items-center justify-between';

        const title = document.createElement('h2');
        title.className = 'text-lg font-bold text-surface-900 dark:text-surface-100';
        title.textContent = 'Simplification Results';

        const badge = document.createElement('span');
        badge.className = 'px-3 py-1 rounded-full text-sm font-medium bg-success-100 text-success-700 dark:bg-success-900/30 dark:text-success-400';
        badge.textContent = 'Complete';

        header.appendChild(title);
        header.appendChild(badge);

        // Body
        const body = document.createElement('div');
        body.className = 'p-6';

        // Summary
        const summary = document.createElement('div');
        summary.className = 'prose prose-surface dark:prose-invert max-w-none';
        summary.textContent = result.summary || 'Simplification completed successfully.';
        body.appendChild(summary);

        // Changes
        if (result.changes && result.changes.length > 0) {
            const changesSection = document.createElement('div');
            changesSection.className = 'mt-6';

            const changesTitle = document.createElement('h3');
            changesTitle.className = 'text-md font-semibold text-surface-900 dark:text-surface-100 mb-3';
            changesTitle.textContent = 'Files Modified (' + result.changes.length + ')';
            changesSection.appendChild(changesTitle);

            const changesList = document.createElement('div');
            changesList.className = 'space-y-2';

            result.changes.forEach(function(change) {
                const changeCard = document.createElement('div');
                changeCard.className = 'p-3 rounded-lg bg-surface-50 dark:bg-surface-800 flex items-center gap-3';

                const icon = createOperationIcon(change.operation);
                changeCard.appendChild(icon);

                const pathSpan = document.createElement('span');
                pathSpan.className = 'text-surface-700 dark:text-surface-300 font-mono text-sm';
                pathSpan.textContent = change.path;
                changeCard.appendChild(pathSpan);

                const opBadge = document.createElement('span');
                const opColor = getOperationColor(change.operation);
                opBadge.className = 'ml-auto px-2 py-0.5 rounded text-xs font-medium bg-' + opColor + '-100 text-' + opColor + '-700 dark:bg-' + opColor + '-900/30 dark:text-' + opColor + '-400';
                opBadge.textContent = change.operation || 'modify';
                changeCard.appendChild(opBadge);

                changesList.appendChild(changeCard);
            });

            changesSection.appendChild(changesList);
            body.appendChild(changesSection);
        }

        // Usage
        if (result.usage) {
            const usageDiv = document.createElement('div');
            usageDiv.className = 'mt-4 pt-4 border-t border-surface-200 dark:border-surface-700 text-sm text-surface-500';
            let usageText = 'Tokens: ' + result.usage.input_tokens + ' input, ' + result.usage.output_tokens + ' output';
            if (result.usage.cost_usd) {
                usageText += ' ($' + result.usage.cost_usd.toFixed(4) + ')';
            }
            usageDiv.textContent = usageText;
            body.appendChild(usageDiv);
        }

        card.appendChild(header);
        card.appendChild(body);

        resultsEl.replaceChildren(card);
    }

    /**
     * Create an SVG icon for the operation type.
     */
    function createOperationIcon(operation) {
        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('class', 'w-5 h-5 text-surface-400');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('viewBox', '0 0 24 24');

        const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path.setAttribute('stroke-linecap', 'round');
        path.setAttribute('stroke-linejoin', 'round');
        path.setAttribute('stroke-width', '2');

        switch(operation) {
            case 'create':
                path.setAttribute('d', 'M12 4v16m8-8H4');
                break;
            case 'delete':
                path.setAttribute('d', 'M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16');
                break;
            case 'rename':
                path.setAttribute('d', 'M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z');
                break;
            default: // modify
                path.setAttribute('d', 'M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z');
        }

        svg.appendChild(path);
        return svg;
    }

    /**
     * Get color class for operation type.
     */
    function getOperationColor(operation) {
        switch(operation) {
            case 'create':
                return 'success';
            case 'delete':
                return 'error';
            case 'rename':
                return 'warning';
            default:
                return 'brand';
        }
    }

    /**
     * Render an error message using safe DOM methods.
     */
    function renderError(message) {
        const card = document.createElement('div');
        card.className = 'card border-error-500';

        const body = document.createElement('div');
        body.className = 'p-6';

        const flex = document.createElement('div');
        flex.className = 'flex items-center gap-3 text-error-600 dark:text-error-400';

        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('class', 'w-6 h-6');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('viewBox', '0 0 24 24');
        const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path.setAttribute('stroke-linecap', 'round');
        path.setAttribute('stroke-linejoin', 'round');
        path.setAttribute('stroke-width', '2');
        path.setAttribute('d', 'M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z');
        svg.appendChild(path);

        const titleEl = document.createElement('span');
        titleEl.className = 'font-medium';
        titleEl.textContent = 'Simplification Error';

        flex.appendChild(svg);
        flex.appendChild(titleEl);

        const desc = document.createElement('p');
        desc.className = 'mt-2 text-surface-600 dark:text-surface-400';
        desc.textContent = message;

        body.appendChild(flex);
        body.appendChild(desc);
        card.appendChild(body);

        resultsEl.replaceChildren(card);
    }

    /**
     * Show the empty state using safe DOM methods.
     */
    function showEmptyState() {
        const card = document.createElement('div');
        card.className = 'card';

        const center = document.createElement('div');
        center.className = 'text-center py-12';

        const iconWrapper = document.createElement('div');
        iconWrapper.className = 'w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-surface-100 to-surface-50 dark:from-surface-800 dark:to-surface-900 flex items-center justify-center';

        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        svg.setAttribute('class', 'w-8 h-8 text-surface-400');
        svg.setAttribute('fill', 'none');
        svg.setAttribute('stroke', 'currentColor');
        svg.setAttribute('viewBox', '0 0 24 24');
        const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        path.setAttribute('stroke-linecap', 'round');
        path.setAttribute('stroke-linejoin', 'round');
        path.setAttribute('stroke-width', '2');
        path.setAttribute('d', 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15');
        svg.appendChild(path);
        iconWrapper.appendChild(svg);

        const mainText = document.createElement('p');
        mainText.className = 'text-surface-600 dark:text-surface-400';
        mainText.textContent = 'Configure the simplify options and click "Run Simplify" to start.';

        const subText = document.createElement('p');
        subText.className = 'text-sm text-surface-500 dark:text-surface-500 mt-1';
        subText.textContent = 'The AI will analyze and simplify your code changes.';

        center.appendChild(iconWrapper);
        center.appendChild(mainText);
        center.appendChild(subText);
        card.appendChild(center);

        resultsEl.replaceChildren(card);
    }

})();
