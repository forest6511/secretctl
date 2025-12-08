/**
 * E2E Test Configuration
 *
 * Security Note: Test password should be provided via environment variable
 * in CI environments. The fallback is only for local development.
 */

// Use environment variable if available, fallback to default for local dev only
// In CI, set E2E_TEST_PASSWORD secret in GitHub Actions
export const TEST_PASSWORD = process.env.E2E_TEST_PASSWORD || 'TestPassword123!'

// Warn if using default password in CI
if (process.env.CI && !process.env.E2E_TEST_PASSWORD) {
  console.warn('WARNING: E2E_TEST_PASSWORD not set in CI environment')
}
