import { test, expect, Page } from '@playwright/test'
import { TEST_PASSWORD } from './test-config'

/**
 * Secrets E2E Tests
 * Tests: Phase 2.5d-1 - Multi-field secret display
 *
 * Prerequisites:
 * - Vault must exist and be unlocked
 * - SECRETCTL_VAULT_DIR environment variable must be set
 */

function secretKeyToTestId(key: string) {
  return key.replace(/\//g, '-')
}

async function ensureAuthenticated(page: Page) {
  await page.goto('/')
  await page.waitForLoadState('networkidle')

  const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
  if (isCreateMode) {
    await page.getByTestId('master-password').fill(TEST_PASSWORD)
    await page.getByTestId('confirm-password').fill(TEST_PASSWORD)
    await page.getByTestId('unlock-button').click()
    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    return
  }

  const isUnlockMode = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)
  if (isUnlockMode) {
    await page.getByTestId('master-password').fill(TEST_PASSWORD)
    await page.getByTestId('unlock-button').click()
    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    return
  }

  await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
}

async function createTestSecret(
  page: Page,
  key: string,
  {
    fieldName = 'username',
    fieldValue = 'testuser',
    sensitive = false,
  }: { fieldName?: string; fieldValue?: string; sensitive?: boolean } = {},
) {
  await page.getByTestId('add-secret-button').click()
  await page.getByTestId('secret-key-input').fill(key)

  await page.getByTestId('add-field-button').click()
  await expect(page.getByTestId('add-field-dialog')).toBeVisible()

  await page.getByTestId('field-name-input').fill(fieldName)
  await page.getByTestId('field-value-input').fill(fieldValue)
  if (sensitive) {
    await page.getByTestId('field-sensitive-checkbox').check()
  }
  await page.getByTestId('add-field-confirm').click()

  await page.getByTestId('save-secret-button').click()

  const secretItemTestId = `secret-item-${secretKeyToTestId(key)}`
  await expect(page.getByTestId(secretItemTestId)).toBeVisible({ timeout: 5000 })
  return secretItemTestId
}

test.describe('Secret Management', () => {
  test.beforeEach(async ({ page }) => {
    await ensureAuthenticated(page)
  })

  test.describe('Secret List View', () => {
    test('CORE-001 should display secrets list', async ({ page }) => {
      await expect(page.getByTestId('secrets-list')).toBeVisible()
    })

    test('CORE-002 should show field count badge when secret has fields', async ({ page }) => {
      const secretKey = `test/multifield-${Date.now()}`
      const secretItemTestId = await createTestSecret(page, secretKey, {
        fieldName: 'password',
        fieldValue: 'testvalue',
      })

      await page.waitForTimeout(200)
      const secretItem = page.getByTestId(secretItemTestId)
      // Note: Secret has 2 fields (default "value" + added "password")
      await expect(secretItem.locator('text=2 fields')).toBeVisible({ timeout: 5000 })

      // Cleanup: delete the test secret
      await page.getByTestId(secretItemTestId).click()
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })
  })

  test.describe('Secret Detail View', () => {
    test('CORE-003 should display FieldsSection for multi-field secrets', async ({ page }) => {
      const secretKey = `test/fields-${Date.now()}`
      const secretItemTestId = await createTestSecret(page, secretKey, {
        fieldName: 'password',
        fieldValue: 'password123',
      })

      // Click on the secret to view details
      await page.getByTestId(secretItemTestId).click()
      await expect(page.getByTestId('fields-section')).toBeVisible()

      // Cleanup
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })

    test('CORE-004 should toggle field visibility for sensitive fields', async ({ page }) => {
      const secretKey = `test/sensitive-${Date.now()}`
      const secretItemTestId = await createTestSecret(page, secretKey, {
        fieldName: 'secret',
        fieldValue: 'secretpassword',
        sensitive: true,
      })

      await page.getByTestId(secretItemTestId).click()

      const fieldValue = page.getByTestId('field-value-secret')
      await expect(fieldValue).toHaveAttribute('type', 'password')

      await page.getByTestId('toggle-field-secret').click()
      await expect(fieldValue).toHaveAttribute('type', 'text')

      // Cleanup
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })

    test('CORE-005 should copy field value to clipboard', async ({ page }) => {
      const secretKey = `test/copy-${Date.now()}`
      const secretItemTestId = await createTestSecret(page, secretKey, {
        fieldName: 'token',
        fieldValue: 'copytest123',
      })

      await page.getByTestId(secretItemTestId).click()

      await page.getByTestId('copy-field-token').click()
      await expect(page.getByText(/Copied|Auto-clears/)).toBeVisible({ timeout: 3000 })

      // Cleanup
      await page.getByTestId('delete-secret-button').click()
      await page.getByTestId('confirm-dialog-confirm').click()
    })
  })
})
