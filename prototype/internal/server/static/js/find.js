/**
 * Find Search Module
 *
 * AI-powered code search with streaming support.
 * Handles both standard and streaming search modes.
 *
 * @module find
 */

(function() {
    'use strict';

    const form = document.getElementById('find-form');
    const queryInput = document.getElementById('find-query');
    const pathInput = document.getElementById('find-path');
    const patternInput = document.getElementById('find-pattern');
    const contextSelect = document.getElementById('find-context');
    const streamCheckbox = document.getElementById('find-stream');
    const submitBtn = document.getElementById('find-submit');
    const clearBtn = document.getElementById('find-clear');
    const cancelBtn = document.getElementById('find-cancel');
    const resultsContainer = document.getElementById('find-results');
    const statusContainer = document.getElementById('find-status');

    let eventSource = null;
    let isSearching = false;

    // Clear button handler
    clearBtn?.addEventListener('click', () => {
        queryInput.value = '';
        pathInput.value = '';
        patternInput.value = '';
        contextSelect.value = '3';
        streamCheckbox.checked = false;
        showEmptyState();
    });

    // Cancel button handler
    cancelBtn?.addEventListener('click', () => {
        if (eventSource) {
            eventSource.close();
            eventSource = null;
        }
        isSearching = false;
        hideStatus();
    });

    // Form submit handler
    form?.addEventListener('submit', async (e) => {
        e.preventDefault();

        const query = queryInput.value.trim();
        if (!query) {
            showError('Please enter a search query');
            return;
        }

        if (isSearching) {
            return;
        }

        const useStream = streamCheckbox.checked;
        if (useStream) {
            await executeStreamingSearch(query);
        } else {
            await executeStandardSearch(query);
        }
    });

    /**
     * Execute a standard (non-streaming) search.
     */
    async function executeStandardSearch(query) {
        const params = new URLSearchParams({ q: query });

        if (pathInput.value) {
            params.set('path', pathInput.value);
        }
        if (patternInput.value) {
            params.set('pattern', patternInput.value);
        }
        params.set('context', contextSelect.value);

        showStatus();
        isSearching = true;

        try {
            const response = await fetch(`/api/v1/find?${params.toString()}`);
            const data = await response.json();

            if (!response.ok) {
                showError(data.error || 'Search failed');
                return;
            }

            displayResults(data.matches, data.query, data.count);
        } catch (error) {
            showError('Network error: ' + error.message);
        } finally {
            isSearching = false;
            hideStatus();
        }
    }

    /**
     * Execute a streaming search using Server-Sent Events.
     */
    async function executeStreamingSearch(query) {
        const params = new URLSearchParams({
            q: query,
            stream: 'true',
            context: contextSelect.value
        });

        if (pathInput.value) {
            params.set('path', pathInput.value);
        }
        if (patternInput.value) {
            params.set('pattern', patternInput.value);
        }

        showStatus();
        isSearching = true;

        // Clear results container
        resultsContainer.innerHTML = `
            <div class="card">
                <div class="p-4">
                    <div class="flex items-center gap-2 text-surface-600 dark:text-surface-400">
                        <div class="animate-spin text-brand-600">&#9696;</div>
                        <span>Searching for: <strong>${escapeHtml(query)}</strong></span>
                    </div>
                </div>
                <div id="stream-results" class="border-t border-surface-200 dark:border-surface-700"></div>
            </div>
        `;

        try {
            eventSource = new EventSource(`/api/v1/find?${params.toString()}`);

            const results = [];
            let resultCount = 0;

            eventSource.addEventListener('started', (e) => {
                const data = JSON.parse(e.data);
                resultsContainer.innerHTML = `
                    <div class="card">
                        <div class="p-4">
                            <div class="flex items-center gap-2 text-surface-600 dark:text-surface-400">
                                <div class="animate-spin text-brand-600">&#9696;</div>
                                <span>Searching for: <strong>${escapeHtml(data.query)}</strong></span>
                            </div>
                        </div>
                        <div id="stream-results" class="border-t border-surface-200 dark:border-surface-700"></div>
                    </div>
                `;
            });

            eventSource.addEventListener('result', (e) => {
                const result = JSON.parse(e.data);
                resultCount++;
                results.push(result);

                const streamContainer = document.getElementById('stream-results');
                if (streamContainer) {
                    streamContainer.insertAdjacentHTML('beforeend', renderResultCard(result, resultCount));
                }
            });

            eventSource.addEventListener('complete', (e) => {
                const data = JSON.parse(e.data);
                finishStreamingResults(data.count);
            });

            eventSource.addEventListener('error', (e) => {
                showError('Search error: ' + e.data);
                eventSource.close();
                isSearching = false;
                hideStatus();
            });

        } catch (error) {
            showError('Failed to start search: ' + error.message);
            isSearching = false;
            hideStatus();
        }
    }

    /**
     * Display non-streaming search results.
     */
    function displayResults(matches, query, count) {
        if (count === 0) {
            resultsContainer.innerHTML = `
                <div class="card">
                    <div class="text-center py-12">
                        <div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-surface-100 to-surface-50 dark:from-surface-800 dark:to-surface-900 flex items-center justify-center">
                            <svg class="w-8 h-8 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                            </svg>
                        </div>
                        <p class="text-surface-600 dark:text-surface-400">No matches found for <strong>${escapeHtml(query)}</strong></p>
                        <p class="text-sm text-surface-500 dark:text-surface-500 mt-1">Try different keywords or check the spelling.</p>
                    </div>
                </div>
            `;
            return;
        }

        let html = `
            <div class="card">
                <div class="px-6 py-4 border-b border-surface-200 dark:border-surface-700">
                    <h3 class="font-bold text-surface-900 dark:text-surface-100">
                        Found ${count} match${count !== 1 ? 'es' : ''} for <em>${escapeHtml(query)}</em>
                    </h3>
                </div>
                <div class="divide-y divide-surface-200 dark:divide-surface-700">
        `;

        matches.forEach((match, index) => {
            html += renderResultCard(match, index + 1);
        });

        html += `
                </div>
            </div>
        `;

        resultsContainer.innerHTML = html;
    }

    /**
     * Render a single result card.
     */
    function renderResultCard(match, index) {
        const contextHtml = match.context && match.context.length > 0
            ? match.context.map(line => `<div class="text-xs text-surface-500 dark:text-surface-500 font-mono whitespace-pre-wrap">${escapeHtml(line)}</div>`).join('')
            : '';

        const reasonHtml = match.reason
            ? `<div class="mt-2 text-sm text-brand-600 dark:text-brand-400">${escapeHtml(match.reason)}</div>`
            : '';

        return `
            <div class="p-4 hover:bg-surface-50 dark:hover:bg-surface-800 transition-smooth">
                <div class="flex items-start gap-3">
                    <span class="flex-shrink-0 w-6 h-6 flex items-center justify-center rounded-full bg-brand-100 dark:bg-brand-900 text-brand-600 dark:text-brand-400 text-xs font-bold">
                        ${index}
                    </span>
                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2 flex-wrap">
                            <span class="font-mono text-sm text-surface-900 dark:text-surface-100">${escapeHtml(match.file)}</span>
                            <span class="text-surface-400">:</span>
                            <span class="text-surface-600 dark:text-surface-400">${match.line}</span>
                        </div>
                        ${match.snippet ? `<div class="mt-2 font-mono text-sm text-surface-700 dark:text-surface-300 bg-surface-100 dark:bg-surface-800 rounded px-2 py-1 whitespace-pre-wrap">${escapeHtml(match.snippet)}</div>` : ''}
                        ${contextHtml ? `<div class="mt-2 space-y-0.5">${contextHtml}</div>` : ''}
                        ${reasonHtml}
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Finalize streaming results display.
     */
    function finishStreamingResults(count) {
        eventSource.close();
        eventSource = null;
        isSearching = false;
        hideStatus();

        // Update the status line
        const statusLine = resultsContainer.querySelector('.card > div:first-child .flex');
        if (statusLine && count > 0) {
            statusLine.innerHTML = `
                <svg class="w-5 h-5 text-success-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
                <span>Found <strong>${count}</strong> match${count !== 1 ? 'es' : ''}</span>
            `;
        }
    }

    /**
     * Show the empty state.
     */
    function showEmptyState() {
        resultsContainer.innerHTML = `
            <div class="card">
                <div class="text-center py-12">
                    <div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-surface-100 to-surface-50 dark:from-surface-800 dark:to-surface-900 flex items-center justify-center">
                        <svg class="w-8 h-8 text-surface-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
                        </svg>
                    </div>
                    <p class="text-surface-600 dark:text-surface-400">Enter a search query to find code in your repository.</p>
                    <p class="text-sm text-surface-500 dark:text-surface-500 mt-1">The AI will search using Grep/Glob tools and return focused results.</p>
                </div>
            </div>
        `;
    }

    /**
     * Show an error message.
     */
    function showError(message) {
        resultsContainer.innerHTML = `
            <div class="card border border-error-200 dark:border-error-800">
                <div class="p-4 text-error-600 dark:text-error-400 flex items-center gap-3">
                    <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                    </svg>
                    <span>${escapeHtml(message)}</span>
                </div>
            </div>
        `;
    }

    /**
     * Show the search status indicator.
     */
    function showStatus() {
        statusContainer.classList.remove('hidden');
    }

    /**
     * Hide the search status indicator.
     */
    function hideStatus() {
        statusContainer.classList.add('hidden');
    }

    /**
     * Escape HTML to prevent XSS.
     */
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

})();
