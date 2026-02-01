import { test as base, expect } from '@playwright/test';
import { addCoverageReport } from 'monocart-reporter';

const isCoverage = process.env.COVERAGE === 'true';

/**
 * Extended test fixture that collects V8 coverage when COVERAGE=true
 */
export const test = base.extend({
  // Automatically collect coverage for each test
  page: async ({ page }, use) => {
    if (isCoverage) {
      // Start collecting JS coverage
      await page.coverage.startJSCoverage({
        resetOnNavigation: false,
      });
    }

    // Run the test
    await use(page);

    if (isCoverage) {
      // Stop and collect coverage
      const jsCoverage = await page.coverage.stopJSCoverage();

      // Add to monocart reporter
      await addCoverageReport(jsCoverage, test.info());
    }
  },
});

export { expect };
