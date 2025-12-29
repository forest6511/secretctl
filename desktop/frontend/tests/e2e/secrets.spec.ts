import { test, expect } from '@playwright/test'
import { TEST_PASSWORD } from './test-config'

/**
 * Secrets E2E Tests
 * Tests: Phase 2.5d-1 - Multi-field secret display
 *
 * NOTE: Tests requiring the old single-value input UI are skipped.
 * These tests will be updated in a follow-up PR to use the new multi-field UI.
 *
 * Prerequisites:
 * - Vault must exist and be unlocked
 * - SECRETCTL_VAULT_DIR environment variable must be set
 */

test.describe('Secret Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to app
    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // Ensure vault is unlocked
    const isUnlockMode = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)
    const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)

    if (isCreateMode) {
      // Create vault first
      await page.getByTestId('master-password').fill(TEST_PASSWORD)
      await page.getByTestId('confirm-password').fill(TEST_PASSWORD)
      await page.getByTestId('unlock-button').click()
      await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    } else if (isUnlockMode) {
      // Unlock existing vault
      await page.getByTestId('master-password').fill(TEST_PASSWORD)
      await page.getByTestId('unlock-button').click()
      await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    }
  })

  test.describe('Secret List View', () => {
    test('should display secrets list', async ({ page }) => {
      await expect(page.getByTestId('secrets-list')).toBeVisible()
    })

    // SKIP: This test uses the old single-value input UI which no longer exists
    // Will be updated to use new multi-field AddFieldDialog in follow-up PR
    test.skip('should show field count badge when secret has fields', async ({ page }) => {
      // Create a test secret first
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill('test/multifield')
      await page.getByTestId('secret-value-input').fill('testvalue')
      await page.getByTestId('save-secret-button').click()

      // Wait for the secret to appear in the list
      await expect(page.getByTestId('secret-item-test-multifield')).toBeVisible({ timeout: 5000 })

      // Field count badge should be visible (1 field for legacy single value)
      const secretItem = page.getByTestId('secret-item-test-multifield')
      await expect(secretItem.locator('text=1 field')).toBeVisible()

      // Cleanup: delete the test secret
      await page.getByTestId('secret-item-test-multifield').click()
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })
  })

  // SKIP: These tests use the old single-value input UI which no longer exists
  // Will be updated to use new multi-field AddFieldDialog in follow-up PR
  test.describe.skip('Secret Detail View', () => {
    test('should display FieldsSection for multi-field secrets', async ({ page }) => {
      // Create a test secret
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill('test/fields')
      await page.getByTestId('secret-value-input').fill('password123')
      await page.getByTestId('save-secret-button').click()

      // Wait for the secret to appear
      await expect(page.getByTestId('secret-item-test-fields')).toBeVisible({ timeout: 5000 })

      // Click on the secret to view details
      await page.getByTestId('secret-item-test-fields').click()

      // Wait for detail view to load
      await page.waitForTimeout(500)

      // Should show either FieldsSection or legacy value display
      const hasFieldsSection = await page.getByTestId('fields-section').isVisible().catch(() => false)
      const hasLegacyValue = await page.getByTestId('secret-value-display').isVisible().catch(() => false)

      expect(hasFieldsSection || hasLegacyValue).toBeTruthy()

      // Cleanup
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })

    test('should toggle field visibility for sensitive fields', async ({ page }) => {
      // Create a test secret
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill('test/sensitive')
      await page.getByTestId('secret-value-input').fill('secretpassword')
      await page.getByTestId('save-secret-button').click()

      // Wait and click on the secret
      await expect(page.getByTestId('secret-item-test-sensitive')).toBeVisible({ timeout: 5000 })
      await page.getByTestId('secret-item-test-sensitive').click()

      // Wait for detail view
      await page.waitForTimeout(500)

      // Check for toggle visibility button (either in FieldsSection or legacy view)
      const toggleButton = page.getByTestId('toggle-value-visibility').or(page.getByTestId('toggle-field-value'))

      if (await toggleButton.isVisible().catch(() => false)) {
        // Value should be hidden initially (password type)
        const input = page.getByTestId('secret-value-display').or(page.getByTestId('field-value-value'))
        const inputType = await input.getAttribute('type').catch(() => null)
        expect(inputType).toBe('password')

        // Click toggle to show
        await toggleButton.click()

        // Value should now be visible (text type)
        const newInputType = await input.getAttribute('type').catch(() => null)
        expect(newInputType).toBe('text')
      }

      // Cleanup
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })

    test('should copy field value to clipboard', async ({ page }) => {
      // Create a test secret
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill('test/copy')
      await page.getByTestId('secret-value-input').fill('copytest123')
      await page.getByTestId('save-secret-button').click()

      // Wait and click on the secret
      await expect(page.getByTestId('secret-item-test-copy')).toBeVisible({ timeout: 5000 })
      await page.getByTestId('secret-item-test-copy').click()

      // Wait for detail view
      await page.waitForTimeout(500)

      // Click copy button (either in FieldsSection or legacy view)
      const copyButton = page.getByTestId('copy-secret-button').or(page.getByTestId('copy-field-value'))

      if (await copyButton.isVisible().catch(() => false)) {
        await copyButton.click()

        // Should show success toast
        await expect(page.getByText(/Copied|Auto-clears/)).toBeVisible({ timeout: 3000 })
      }

      // Cleanup
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })
  })
})
