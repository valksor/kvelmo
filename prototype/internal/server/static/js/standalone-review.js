/**
 * Standalone Review Module
 *
 * AI-powered standalone code review without active task.
 *
 * @module standalone-review
 */

(function() {
    'use strict';

    const form = document.getElementById('review-form');
    const modeRadios = document.querySelectorAll('input[name="mode"]');
    const branchField = document.getElementById('branch-field');
    const rangeField = document.getElementById('range-field');
    const filesField = document.getElementById('files-field');
    const statusEl = document.getElementById('review-status');
    const statusText = document.getElementById('review-status-text');
    const resultsEl = document.getElementById('review-results');
    const clearBtn = document.getElementById('review-clear');
    const cancelBtn = document.getElementById('review-cancel');
    const applyFixesCheckbox = document.getElementById('apply-fixes');
    const checkpointField = document.getElementById('checkpoint-field');

    let abortController = null;

    // Toggle checkpoint field visibility based on apply fixes checkbox
    applyFixesCheckbox?.addEventListener('change', function() {
        if (this.checked) {
            checkpointField.classList.remove('hidden');
        } else {
            checkpointField.classList.add('hidden');
        }
    });

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
        const applyFixes = formData.get('apply_fixes') === 'on';
        const data = {
            mode: formData.get('mode'),
            base_branch: formData.get('base_branch') || '',
            range: formData.get('range') || '',
            files: filesValue ? filesValue.split(',').map(f => f.trim()).filter(f => f) : [],
            context: parseInt(formData.get('context')) || 3,
            agent: formData.get('agent') || '',
            apply_fixes: applyFixes,
            create_checkpoint: applyFixes ? formData.get('create_checkpoint') === 'on' : false
        };

        // Show status
        statusEl.classList.remove('hidden');
        statusText.textContent = applyFixes ? 'Running review and fixing...' : 'Running review...';

        abortController = new AbortController();

        try {
            const fetchFn = window.csrfFetch || fetch;
            const response = await fetchFn('/api/v1/workflow/review/standalone', {
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
                renderError(result.error || 'Review failed');
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
        checkpointField.classList.add('hidden');
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
     * Render the review results using safe DOM methods.
     */
    function renderResults(result) {
        const card = document.createElement('div');
        card.className = 'card';

        // Header with verdict
        const header = document.createElement('div');
        header.className = 'px-6 py-4 border-b border-surface-200 dark:border-surface-700 flex items-center justify-between';

        const title = document.createElement('h2');
        title.className = 'text-lg font-bold text-surface-900 dark:text-surface-100';
        title.textContent = 'Review Results';

        const verdict = document.createElement('span');
        const verdictColor = result.verdict === 'APPROVED' ? 'success' : result.verdict === 'NEEDS_CHANGES' ? 'warning' : 'surface';
        verdict.className = 'px-3 py-1 rounded-full text-sm font-medium bg-' + verdictColor + '-100 text-' + verdictColor + '-700 dark:bg-' + verdictColor + '-900/30 dark:text-' + verdictColor + '-400';
        verdict.textContent = result.verdict || 'COMMENT';

        header.appendChild(title);
        header.appendChild(verdict);

        // Body
        const body = document.createElement('div');
        body.className = 'p-6';

        // Summary
        const summary = document.createElement('div');
        summary.className = 'prose prose-surface dark:prose-invert max-w-none';
        summary.textContent = result.summary || 'No summary available.';
        body.appendChild(summary);

        // Issues
        if (result.issues && result.issues.length > 0) {
            const issuesSection = document.createElement('div');
            issuesSection.className = 'mt-6';

            const issuesTitle = document.createElement('h3');
            issuesTitle.className = 'text-md font-semibold text-surface-900 dark:text-surface-100 mb-3';
            issuesTitle.textContent = 'Issues Found (' + result.issues.length + ')';
            issuesSection.appendChild(issuesTitle);

            const issuesList = document.createElement('div');
            issuesList.className = 'space-y-2';

            result.issues.forEach(function(issue) {
                const issueCard = document.createElement('div');
                const borderColor = (issue.severity === 'critical' || issue.severity === 'high') ? 'error' : issue.severity === 'medium' ? 'warning' : 'surface';
                issueCard.className = 'p-3 rounded-lg bg-surface-50 dark:bg-surface-800 border-l-4 border-' + borderColor + '-500';

                const issueMeta = document.createElement('div');
                issueMeta.className = 'flex items-center gap-2 text-sm';

                const severitySpan = document.createElement('span');
                severitySpan.className = 'font-medium text-' + borderColor + '-600 uppercase';
                severitySpan.textContent = issue.severity || 'info';
                issueMeta.appendChild(severitySpan);

                if (issue.file) {
                    const fileSpan = document.createElement('span');
                    fileSpan.className = 'text-surface-500';
                    fileSpan.textContent = issue.file + (issue.line ? ':' + issue.line : '');
                    issueMeta.appendChild(fileSpan);
                }

                const issueDesc = document.createElement('p');
                issueDesc.className = 'text-surface-700 dark:text-surface-300 mt-1';
                issueDesc.textContent = issue.description || '';

                issueCard.appendChild(issueMeta);
                issueCard.appendChild(issueDesc);
                issuesList.appendChild(issueCard);
            });

            issuesSection.appendChild(issuesList);
            body.appendChild(issuesSection);
        }

        // Changes Applied (if fixes were applied)
        if (result.changes && result.changes.length > 0) {
            const changesSection = document.createElement('div');
            changesSection.className = 'mt-6';

            const changesTitle = document.createElement('h3');
            changesTitle.className = 'text-md font-semibold text-surface-900 dark:text-surface-100 mb-3';
            changesTitle.textContent = 'Changes Applied (' + result.changes.length + ')';
            changesSection.appendChild(changesTitle);

            const changesList = document.createElement('div');
            changesList.className = 'space-y-2';

            result.changes.forEach(function(change) {
                const changeCard = document.createElement('div');
                changeCard.className = 'p-3 rounded-lg bg-surface-50 dark:bg-surface-800 flex items-center gap-3';

                const opBadge = document.createElement('span');
                const opColor = change.operation === 'create' ? 'success' : change.operation === 'delete' ? 'error' : 'brand';
                opBadge.className = 'px-2 py-0.5 rounded text-xs font-medium bg-' + opColor + '-100 text-' + opColor + '-700 dark:bg-' + opColor + '-900/30 dark:text-' + opColor + '-400';
                opBadge.textContent = (change.operation || 'update').toUpperCase();

                const pathSpan = document.createElement('span');
                pathSpan.className = 'text-surface-700 dark:text-surface-300 font-mono text-sm';
                pathSpan.textContent = change.path;

                changeCard.appendChild(opBadge);
                changeCard.appendChild(pathSpan);
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
        titleEl.textContent = 'Review Error';

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
        path.setAttribute('d', 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z');
        svg.appendChild(path);
        iconWrapper.appendChild(svg);

        const mainText = document.createElement('p');
        mainText.className = 'text-surface-600 dark:text-surface-400';
        mainText.textContent = 'Configure the review options and click "Run Review" to start.';

        const subText = document.createElement('p');
        subText.className = 'text-sm text-surface-500 dark:text-surface-500 mt-1';
        subText.textContent = 'The AI will analyze your code changes and provide feedback.';

        center.appendChild(iconWrapper);
        center.appendChild(mainText);
        center.appendChild(subText);
        card.appendChild(center);

        resultsEl.replaceChildren(card);
    }

})();
