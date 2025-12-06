import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright configuration for secretctl Desktop App E2E tests
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false, // Vault tests need sequential execution
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1, // Single worker for vault state consistency
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
  ],

  use: {
    // Base URL for Wails dev server
    baseURL: 'http://localhost:34115',

    // Capture trace on failure for debugging
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'off', // Disable for security (may capture secrets)

    // Test timeout
    actionTimeout: 10000,
  },

  // Global timeout
  timeout: 30000,

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Don't start web server - Wails dev server must be running
  // Run: cd desktop && wails dev
})
