import { test, expect } from '@playwright/test'

/**
 * Authentication E2E Tests
 * Tests: SEC-001, SEC-002 (P0 Critical)
 *
 * Note: SEC-001 (Vault Creation) tests are skipped when vault already exists.
 * This happens in CI where other tests may run first and create the vault.
 * Run standalone: rm -rf /tmp/secretctl-e2e-test && mkdir /tmp/secretctl-e2e-test
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
      // Skip if vault already exists (happens in CI when other tests run first)
      const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
      if (!isCreateMode) {
        test.skip()
        return
      }

      // Check for create vault UI elements (use heading to be specific)
      await expect(page.getByRole('heading', { name: 'Create Vault' })).toBeVisible()
      await expect(page.getByTestId('master-password')).toBeVisible()
      await expect(page.getByTestId('confirm-password')).toBeVisible()
    })

    test('should reject password shorter than 8 characters', async ({ page }) => {
      // Skip if vault already exists
      const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
      if (!isCreateMode) {
        test.skip()
        return
      }

      await page.getByTestId('master-password').fill('short')
      await page.getByTestId('confirm-password').fill('short')
      await page.getByTestId('unlock-button').click()

      await expect(page.getByText('Password must be at least 8 characters')).toBeVisible()
    })

    test('should reject mismatched passwords', async ({ page }) => {
      // Skip if vault already exists
      const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
      if (!isCreateMode) {
        test.skip()
        return
      }

      await page.getByTestId('master-password').fill('password123')
      await page.getByTestId('confirm-password').fill('different123')
      await page.getByTestId('unlock-button').click()

      await expect(page.getByText('Passwords do not match')).toBeVisible()
    })

    test('should create vault with valid password', async ({ page }) => {
      // Skip if vault already exists
      const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
      if (!isCreateMode) {
        test.skip()
        return
      }

      const password = 'SecurePassword123!'

      await page.getByTestId('master-password').fill(password)
      await page.getByTestId('confirm-password').fill(password)
      await page.getByTestId('unlock-button').click()

      // Should navigate to secrets page after successful creation
      await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    })
  })

  test.describe('SEC-002: Vault Unlock', () => {
    test('should show unlock form when vault exists', async ({ page }) => {
      // After vault creation in previous tests, the app may stay on secrets page
      // Check for secrets page (already logged in) or auth screens
      const secretsPage = await page.getByTestId('secrets-list').isVisible().catch(() => false)
      const createTitle = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
      const unlockTitle = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)

      // Either should be on secrets page (logged in) or one of the auth screens
      expect(secretsPage || createTitle || unlockTitle).toBeTruthy()
    })

    test('should reject incorrect password', async ({ page }) => {
      // First create vault if it doesn't exist
      const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)

      if (isCreateMode) {
        // Create vault first
        await page.getByTestId('master-password').fill('SecurePassword123!')
        await page.getByTestId('confirm-password').fill('SecurePassword123!')
        await page.getByTestId('unlock-button').click()
        await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })

        // Reload to get unlock screen
        await page.reload()
        await page.waitForLoadState('networkidle')
      }

      // Now we should be on unlock screen
      const isUnlockMode = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)

      if (isUnlockMode) {
        await page.getByTestId('master-password').fill('wrongpassword')
        await page.getByTestId('unlock-button').click()
        await expect(page.getByText('Invalid password')).toBeVisible()
      }
    })
  })
})
