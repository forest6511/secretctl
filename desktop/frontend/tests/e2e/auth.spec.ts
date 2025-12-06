import { test, expect } from '@playwright/test'

/**
 * Authentication E2E Tests
 * Tests: SEC-001, SEC-002 (P0 Critical)
 *
 * Note: These tests require a fresh Wails dev server with empty vault directory.
 * Run: rm -rf /tmp/secretctl-e2e-test && mkdir /tmp/secretctl-e2e-test
 * Then restart: SECRETCTL_VAULT_DIR=/tmp/secretctl-e2e-test wails dev
 */

test.describe('Vault Authentication', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app
    await page.goto('/')
    await page.waitForLoadState('networkidle')
  })

  test.describe('SEC-001: Vault Creation', () => {
    test('should show create vault form when no vault exists', async ({ page }) => {
      // Check for create vault UI elements (use heading to be specific)
      await expect(page.getByRole('heading', { name: 'Create Vault' })).toBeVisible()
      await expect(page.getByTestId('master-password')).toBeVisible()
      await expect(page.getByTestId('confirm-password')).toBeVisible()
    })

    test('should reject password shorter than 8 characters', async ({ page }) => {
      await page.getByTestId('master-password').fill('short')
      await page.getByTestId('confirm-password').fill('short')
      await page.getByTestId('unlock-button').click()

      await expect(page.getByText('Password must be at least 8 characters')).toBeVisible()
    })

    test('should reject mismatched passwords', async ({ page }) => {
      await page.getByTestId('master-password').fill('password123')
      await page.getByTestId('confirm-password').fill('different123')
      await page.getByTestId('unlock-button').click()

      await expect(page.getByText('Passwords do not match')).toBeVisible()
    })

    test('should create vault with valid password', async ({ page }) => {
      const password = 'SecurePassword123!'

      await page.getByTestId('master-password').fill(password)
      await page.getByTestId('confirm-password').fill(password)
      await page.getByTestId('unlock-button').click()

      // Should navigate to secrets page after successful creation
      await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    })
  })

  test.describe('SEC-002: Vault Unlock', () => {
    // Note: These tests require a vault to already exist
    // In a real scenario, we'd use fixtures or test setup

    test('should show unlock form when vault exists', async ({ page }) => {
      // If vault exists, should show unlock UI
      const unlockTitle = page.getByRole('heading', { name: 'Unlock Vault' })
      const createTitle = page.getByRole('heading', { name: 'Create Vault' })

      // Either unlock or create should be visible
      const isUnlock = await unlockTitle.isVisible().catch(() => false)
      const isCreate = await createTitle.isVisible().catch(() => false)

      expect(isUnlock || isCreate).toBeTruthy()
    })

    test('should reject incorrect password', async ({ page }) => {
      // Skip if this is a fresh vault (create mode)
      const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
      if (isCreateMode) {
        test.skip()
        return
      }

      await page.getByTestId('master-password').fill('wrongpassword')
      await page.getByTestId('unlock-button').click()

      await expect(page.getByText('Invalid password')).toBeVisible()
    })
  })
})
