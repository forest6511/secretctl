import { test, expect, Page } from '@playwright/test'
import { TEST_PASSWORD } from './test-config'

/**
 * Secret Editing E2E Tests (Issue #113)
 * Tests for multi-field secret editing functionality
 */

/**
 * Helper function to ensure user is authenticated
 */
async function ensureAuthenticated(page: Page) {
  await page.goto('/')
  await page.waitForLoadState('networkidle')

  // Check if we need to create vault
  const isCreateMode = await page.getByRole('heading', { name: 'Create Vault' }).isVisible().catch(() => false)
  if (isCreateMode) {
    await page.getByTestId('master-password').fill(TEST_PASSWORD)
    await page.getByTestId('confirm-password').fill(TEST_PASSWORD)
    await page.getByTestId('unlock-button').click()
    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    return
  }

  // Check if we need to unlock
  const isUnlockMode = await page.getByRole('heading', { name: 'Unlock Vault' }).isVisible().catch(() => false)
  if (isUnlockMode) {
    await page.getByTestId('master-password').fill(TEST_PASSWORD)
    await page.getByTestId('unlock-button').click()
    await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
    return
  }

  // Already on secrets page
  await expect(page.getByTestId('secrets-list')).toBeVisible({ timeout: 10000 })
}

/**
 * Helper function to create a test secret
 */
async function createTestSecret(page: Page, key: string) {
  // Click add secret button
  await page.getByTestId('add-secret-button').click()

  // Fill in the key
  await page.getByTestId('secret-key-input').fill(key)

  // Add a field
  await page.getByTestId('add-field-button').click()
  await expect(page.getByTestId('add-field-dialog')).toBeVisible()

  await page.getByTestId('field-name-input').fill('username')
  await page.getByTestId('field-value-input').fill('testuser')
  await page.getByTestId('add-field-confirm').click()

  // Save the secret
  await page.getByTestId('save-secret-button').click()

  // Wait for the secret to appear in the list
  await expect(page.getByTestId(`secret-item-${key}`)).toBeVisible({ timeout: 5000 })
}

test.describe.skip('Secret Multi-field Editing', () => {
  test.beforeEach(async ({ page }) => {
    await ensureAuthenticated(page)
  })

  test.describe('Field Management', () => {
    test('should add a new field to a secret', async ({ page }) => {
      const secretKey = `test-add-field-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Click to edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Add a new field
      await page.getByTestId('add-field-button').click()
      await expect(page.getByTestId('add-field-dialog')).toBeVisible()

      await page.getByTestId('field-name-input').fill('password')
      await page.getByTestId('field-value-input').fill('secretpassword')
      await page.getByTestId('field-sensitive-checkbox').check()
      await page.getByTestId('add-field-confirm').click()

      // Save changes
      await page.getByTestId('save-secret-button').click()

      // Verify the new field exists
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await expect(page.getByTestId('field-password')).toBeVisible()
    })

    test('should validate field name format', async ({ page }) => {
      const secretKey = `test-field-validation-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Try to add a field with invalid name
      await page.getByTestId('add-field-button').click()
      await expect(page.getByTestId('add-field-dialog')).toBeVisible()

      // Test uppercase (should fail - must be snake_case)
      await page.getByTestId('field-name-input').fill('InvalidName')
      await page.getByTestId('add-field-confirm').click()
      await expect(page.getByTestId('field-name-error')).toBeVisible()

      // Test with spaces (should fail)
      await page.getByTestId('field-name-input').fill('invalid name')
      await page.getByTestId('add-field-confirm').click()
      await expect(page.getByTestId('field-name-error')).toBeVisible()

      // Cancel the dialog
      await page.getByTestId('add-field-cancel').click()
    })

    test('should reject duplicate field names', async ({ page }) => {
      const secretKey = `test-dup-field-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Try to add a field with the same name as existing
      await page.getByTestId('add-field-button').click()
      await expect(page.getByTestId('add-field-dialog')).toBeVisible()

      await page.getByTestId('field-name-input').fill('username')
      await page.getByTestId('add-field-confirm').click()
      await expect(page.getByTestId('field-name-error')).toContainText('already exists')

      await page.getByTestId('add-field-cancel').click()
    })

    test('should edit field value', async ({ page }) => {
      const secretKey = `test-edit-value-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Edit the field value
      const fieldInput = page.getByTestId('field-value-username')
      await fieldInput.clear()
      await fieldInput.fill('newusername')

      // Save changes
      await page.getByTestId('save-secret-button').click()

      // Verify the change
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('toggle-field-username').click() // Show value
      await expect(page.getByTestId('field-username')).toContainText('newusername')
    })

    test('should toggle field sensitivity', async ({ page }) => {
      const secretKey = `test-toggle-sensitive-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Toggle sensitivity
      await page.getByTestId('toggle-sensitive-username').click()

      // Save changes
      await page.getByTestId('save-secret-button').click()

      // Verify field is now sensitive (should show masked by default)
      await page.getByTestId(`secret-item-${secretKey}`).click()
      const fieldValue = page.getByTestId('field-username')
      await expect(fieldValue).toContainText('****')
    })

    test('should delete a field', async ({ page }) => {
      const secretKey = `test-delete-field-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Add another field first
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('add-field-button').click()
      await page.getByTestId('field-name-input').fill('extra_field')
      await page.getByTestId('field-value-input').fill('extravalue')
      await page.getByTestId('add-field-confirm').click()
      await page.getByTestId('save-secret-button').click()

      // Now delete the extra field
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('delete-field-extra_field').click()

      // Confirm deletion in dialog
      await expect(page.getByText('Delete Field')).toBeVisible()
      await page.getByRole('button', { name: 'Delete' }).click()

      // Save changes
      await page.getByTestId('save-secret-button').click()

      // Verify field is deleted
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await expect(page.getByTestId('field-extra_field')).not.toBeVisible()
    })
  })

  test.describe('Environment Bindings', () => {
    test('should add an environment binding', async ({ page }) => {
      const secretKey = `test-add-binding-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Add a binding
      await page.getByTestId('add-binding-button').click()
      await expect(page.getByTestId('add-binding-dialog')).toBeVisible()

      await page.getByTestId('binding-envvar-input').fill('MY_USERNAME')
      // Field select should default to first field
      await page.getByTestId('add-binding-confirm').click()

      // Save changes
      await page.getByTestId('save-secret-button').click()

      // Verify binding exists
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await expect(page.getByTestId('binding-MY_USERNAME')).toBeVisible()
    })

    test('should validate environment variable format', async ({ page }) => {
      const secretKey = `test-binding-validation-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Try to add binding with invalid env var name
      await page.getByTestId('add-binding-button').click()
      await expect(page.getByTestId('add-binding-dialog')).toBeVisible()

      // Test lowercase (should fail - must be SCREAMING_SNAKE_CASE)
      await page.getByTestId('binding-envvar-input').fill('lowercase_var')
      await page.getByTestId('add-binding-confirm').click()
      await expect(page.getByTestId('binding-envvar-error')).toBeVisible()

      // Test with spaces
      await page.getByTestId('binding-envvar-input').fill('INVALID VAR')
      await page.getByTestId('add-binding-confirm').click()
      await expect(page.getByTestId('binding-envvar-error')).toBeVisible()

      await page.getByTestId('add-binding-cancel').click()
    })

    test('should auto-uppercase environment variable name', async ({ page }) => {
      const secretKey = `test-binding-uppercase-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Edit the secret
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Add binding with lowercase input
      await page.getByTestId('add-binding-button').click()
      await page.getByTestId('binding-envvar-input').fill('my_var')

      // Should auto-convert to uppercase
      const inputValue = await page.getByTestId('binding-envvar-input').inputValue()
      expect(inputValue).toBe('MY_VAR')

      await page.getByTestId('add-binding-cancel').click()
    })

    test('should delete an environment binding', async ({ page }) => {
      const secretKey = `test-delete-binding-${Date.now()}`
      await createTestSecret(page, secretKey)

      // First add a binding
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('add-binding-button').click()
      await page.getByTestId('binding-envvar-input').fill('DELETE_ME')
      await page.getByTestId('add-binding-confirm').click()
      await page.getByTestId('save-secret-button').click()

      // Now delete the binding
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('delete-binding-DELETE_ME').click()

      // Save changes
      await page.getByTestId('save-secret-button').click()

      // Verify binding is deleted
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await expect(page.getByTestId('binding-DELETE_ME')).not.toBeVisible()
    })

    test('should reject duplicate environment variable names', async ({ page }) => {
      const secretKey = `test-dup-binding-${Date.now()}`
      await createTestSecret(page, secretKey)

      // Add first binding
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('add-binding-button').click()
      await page.getByTestId('binding-envvar-input').fill('DUPLICATE_VAR')
      await page.getByTestId('add-binding-confirm').click()

      // Try to add same binding again
      await page.getByTestId('add-binding-button').click()
      await page.getByTestId('binding-envvar-input').fill('DUPLICATE_VAR')
      await page.getByTestId('add-binding-confirm').click()
      await expect(page.getByTestId('binding-envvar-error')).toContainText('already bound')

      await page.getByTestId('add-binding-cancel').click()
    })
  })

  test.describe('Secret Creation with Multi-fields', () => {
    test('should create a secret with multiple fields and bindings', async ({ page }) => {
      const secretKey = `test-multifield-${Date.now()}`

      // Click add secret button
      await page.getByTestId('add-secret-button').click()
      await page.getByTestId('secret-key-input').fill(secretKey)

      // Add username field
      await page.getByTestId('add-field-button').click()
      await page.getByTestId('field-name-input').fill('username')
      await page.getByTestId('field-value-input').fill('admin')
      await page.getByTestId('add-field-confirm').click()

      // Add password field (sensitive)
      await page.getByTestId('add-field-button').click()
      await page.getByTestId('field-name-input').fill('password')
      await page.getByTestId('field-value-input').fill('secret123')
      await page.getByTestId('field-sensitive-checkbox').check()
      await page.getByTestId('add-field-confirm').click()

      // Add api_key field
      await page.getByTestId('add-field-button').click()
      await page.getByTestId('field-name-input').fill('api_key')
      await page.getByTestId('field-value-input').fill('key-12345')
      await page.getByTestId('field-sensitive-checkbox').check()
      await page.getByTestId('add-field-confirm').click()

      // Add bindings
      await page.getByTestId('add-binding-button').click()
      await page.getByTestId('binding-envvar-input').fill('DB_USER')
      await page.getByTestId('binding-field-select').selectOption('username')
      await page.getByTestId('add-binding-confirm').click()

      await page.getByTestId('add-binding-button').click()
      await page.getByTestId('binding-envvar-input').fill('DB_PASSWORD')
      await page.getByTestId('binding-field-select').selectOption('password')
      await page.getByTestId('add-binding-confirm').click()

      // Save the secret
      await page.getByTestId('save-secret-button').click()

      // Verify the secret was created
      await expect(page.getByTestId(`secret-item-${secretKey}`)).toBeVisible({ timeout: 5000 })

      // Click to view details
      await page.getByTestId(`secret-item-${secretKey}`).click()

      // Verify fields exist
      await expect(page.getByTestId('field-username')).toBeVisible()
      await expect(page.getByTestId('field-password')).toBeVisible()
      await expect(page.getByTestId('field-api_key')).toBeVisible()

      // Verify bindings exist
      await expect(page.getByTestId('binding-DB_USER')).toBeVisible()
      await expect(page.getByTestId('binding-DB_PASSWORD')).toBeVisible()
    })
  })

  test.describe('Cancel Operations', () => {
    test('should cancel adding a field', async ({ page }) => {
      const secretKey = `test-cancel-field-${Date.now()}`
      await createTestSecret(page, secretKey)

      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('add-field-button').click()
      await page.getByTestId('field-name-input').fill('cancelled_field')
      await page.getByTestId('add-field-cancel').click()

      // Dialog should be closed
      await expect(page.getByTestId('add-field-dialog')).not.toBeVisible()

      // Field should not exist
      await expect(page.getByTestId('field-cancelled_field')).not.toBeVisible()
    })

    test('should cancel editing and discard changes', async ({ page }) => {
      const secretKey = `test-cancel-edit-${Date.now()}`
      await createTestSecret(page, secretKey)

      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      // Make a change
      await page.getByTestId('add-field-button').click()
      await page.getByTestId('field-name-input').fill('temp_field')
      await page.getByTestId('field-value-input').fill('tempvalue')
      await page.getByTestId('add-field-confirm').click()

      // Cancel editing
      await page.getByTestId('cancel-button').click()

      // Verify the change was discarded
      await page.getByTestId(`secret-item-${secretKey}`).click()
      await expect(page.getByTestId('field-temp_field')).not.toBeVisible()
    })

    test('should close dialog with escape key', async ({ page }) => {
      const secretKey = `test-escape-${Date.now()}`
      await createTestSecret(page, secretKey)

      await page.getByTestId(`secret-item-${secretKey}`).click()
      await page.getByTestId('edit-secret-button').click()

      await page.getByTestId('add-field-button').click()
      await expect(page.getByTestId('add-field-dialog')).toBeVisible()

      await page.keyboard.press('Escape')
      await expect(page.getByTestId('add-field-dialog')).not.toBeVisible()
    })
  })
})
