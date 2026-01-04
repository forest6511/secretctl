import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FieldEditor, FieldDTO } from './FieldEditor'

// Mock Wails bindings with spies
const mockViewSensitiveField = vi.fn().mockResolvedValue(undefined)
const mockCopyFieldValue = vi.fn().mockResolvedValue(undefined)

vi.mock('../../wailsjs/go/main/App', () => ({
  ViewSensitiveField: (...args: unknown[]) => mockViewSensitiveField(...args),
  CopyFieldValue: (...args: unknown[]) => mockCopyFieldValue(...args),
}))

// Mock useToast hook
vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    toast: vi.fn(),
  }),
}))

describe('FieldEditor', () => {
  const defaultProps = {
    secretKey: 'test-secret',
    fieldName: 'testField',
    field: {
      value: '',
      sensitive: false,
    } as FieldDTO,
  }

  describe('Separation of Concerns - Display Masking vs Input Blocking', () => {
    // Issue #148: Textarea became readonly after typing 1 character
    // Root cause: Mixed concerns - display masking logic controlled input blocking
    // Fix: Separated into 3 distinct concerns:
    //   1. Display masking: Only in read mode for sensitive fields
    //   2. Input blocking: Controlled solely by readOnly prop
    //   3. Visual styling: Applied when display is masked

    it('allows typing in textarea edit mode for sensitive fields', async () => {
      // This is the critical regression test for Issue #148
      const onChange = vi.fn()
      const field: FieldDTO = {
        value: '',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor
          {...defaultProps}
          field={field}
          readOnly={false}
          onChange={onChange}
        />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea).not.toHaveAttribute('readonly')

      // Type first character - should NOT become readonly
      await userEvent.type(textarea, 'a')
      expect(onChange).toHaveBeenCalledWith('a')
      expect(textarea).not.toHaveAttribute('readonly')

      // Type more characters - should still NOT be readonly
      await userEvent.type(textarea, 'bc')
      expect(onChange).toHaveBeenCalledTimes(3)
    })

    it('allows typing in input edit mode for sensitive fields', async () => {
      const onChange = vi.fn()
      const field: FieldDTO = {
        value: '',
        sensitive: true,
        inputType: 'text',
      }

      render(
        <FieldEditor
          {...defaultProps}
          field={field}
          readOnly={false}
          onChange={onChange}
        />
      )

      const input = screen.getByTestId('field-value-testField')
      expect(input).not.toHaveAttribute('readonly')

      await userEvent.type(input, 'password123')
      expect(onChange).toHaveBeenCalledTimes(11)
    })

    it('shows actual value by default for textarea in read mode (visible default)', () => {
      // Textarea defaults to visible so users can verify pasted content
      const field: FieldDTO = {
        value: 'secret-ssh-key-content',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea).toHaveValue('secret-ssh-key-content')
      expect(textarea).toHaveAttribute('readonly')
    })

    it('shows actual value in edit mode for sensitive fields', () => {
      const field: FieldDTO = {
        value: 'my-secret-key',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea).toHaveValue('my-secret-key')
    })

    it('does not mask empty sensitive fields in read mode', () => {
      const field: FieldDTO = {
        value: '',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea).toHaveValue('')
    })
  })

  describe('Visibility Toggle', () => {
    it('shows eye icon for sensitive fields', () => {
      const field: FieldDTO = {
        value: 'secret',
        sensitive: true,
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const toggleButton = screen.getByTestId('toggle-field-testField')
      expect(toggleButton).toBeInTheDocument()
    })

    it('does not show eye icon for non-sensitive fields', () => {
      const field: FieldDTO = {
        value: 'public',
        sensitive: false,
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      expect(screen.queryByTestId('toggle-field-testField')).not.toBeInTheDocument()
    })

    it('hides value when visibility toggled in read mode (textarea starts visible)', async () => {
      // Textarea defaults to visible in read mode for content verification
      const field: FieldDTO = {
        value: 'secret-value',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      // Textarea defaults to visible
      expect(textarea).toHaveValue('secret-value')

      const toggleButton = screen.getByTestId('toggle-field-testField')
      await userEvent.click(toggleButton)

      // Now it's masked
      expect(textarea).toHaveValue('••••••••')
    })
  })

  describe('Input Type Rendering', () => {
    it('renders textarea for textarea inputType', () => {
      const field: FieldDTO = {
        value: 'multi\nline\ntext',
        sensitive: false,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea.tagName.toLowerCase()).toBe('textarea')
    })

    it('renders input for text inputType', () => {
      const field: FieldDTO = {
        value: 'single line',
        sensitive: false,
        inputType: 'text',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const input = screen.getByTestId('field-value-testField')
      expect(input.tagName.toLowerCase()).toBe('input')
    })

    it('uses password type for hidden sensitive input fields', () => {
      const field: FieldDTO = {
        value: 'password',
        sensitive: true,
        inputType: 'text',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const input = screen.getByTestId('field-value-testField')
      expect(input).toHaveAttribute('type', 'password')
    })
  })

  describe('Edit Mode Controls', () => {
    it('shows delete button in edit mode', () => {
      const onDelete = vi.fn()
      const field: FieldDTO = { value: 'test', sensitive: false }

      render(
        <FieldEditor
          {...defaultProps}
          field={field}
          readOnly={false}
          onDelete={onDelete}
        />
      )

      const deleteButton = screen.getByTestId('delete-field-testField')
      expect(deleteButton).toBeInTheDocument()
    })

    it('hides delete button in read mode', () => {
      const field: FieldDTO = { value: 'test', sensitive: false }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      expect(screen.queryByTestId('delete-field-testField')).not.toBeInTheDocument()
    })

    it('shows sensitive toggle button in edit mode', () => {
      const onSensitiveToggle = vi.fn()
      const field: FieldDTO = { value: 'test', sensitive: false }

      render(
        <FieldEditor
          {...defaultProps}
          field={field}
          readOnly={false}
          onSensitiveToggle={onSensitiveToggle}
        />
      )

      const toggleButton = screen.getByTestId('toggle-sensitive-testField')
      expect(toggleButton).toBeInTheDocument()
    })
  })

  describe('Copy Functionality', () => {
    it('shows copy button for all fields', () => {
      const field: FieldDTO = { value: 'test', sensitive: false }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const copyButton = screen.getByTestId('copy-field-testField')
      expect(copyButton).toBeInTheDocument()
    })
  })

  describe('Field Hint', () => {
    it('displays hint when provided', () => {
      const field: FieldDTO = {
        value: 'test',
        sensitive: false,
        hint: 'Enter your SSH private key',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      expect(screen.getByText('(Enter your SSH private key)')).toBeInTheDocument()
    })
  })

  describe('Audit Logging - ViewSensitiveField', () => {
    beforeEach(() => {
      mockViewSensitiveField.mockClear()
    })

    it('calls ViewSensitiveField when revealing sensitive field in read mode', async () => {
      const field: FieldDTO = {
        value: 'secret-value',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const toggleButton = screen.getByTestId('toggle-field-testField')

      // Textarea defaults to visible, so first click hides (no audit)
      await userEvent.click(toggleButton)
      expect(mockViewSensitiveField).not.toHaveBeenCalled()

      // Second click reveals - this triggers audit
      await userEvent.click(toggleButton)
      expect(mockViewSensitiveField).toHaveBeenCalledWith('test-secret', 'testField')
    })

    it('does NOT call ViewSensitiveField in edit mode (new secrets)', async () => {
      const field: FieldDTO = {
        value: 'secret-value',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const toggleButton = screen.getByTestId('toggle-field-testField')
      await userEvent.click(toggleButton)

      // Audit logging is intentionally skipped in edit mode
      // because user is actively editing, not just viewing
      expect(mockViewSensitiveField).not.toHaveBeenCalled()
    })

    it('does NOT call ViewSensitiveField for new secrets without secretKey', async () => {
      const field: FieldDTO = {
        value: 'secret-value',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor
          secretKey=""
          fieldName="testField"
          field={field}
          readOnly={true}
        />
      )

      const toggleButton = screen.getByTestId('toggle-field-testField')
      await userEvent.click(toggleButton)

      // No audit logging for new secrets (no secretKey)
      expect(mockViewSensitiveField).not.toHaveBeenCalled()
    })
  })

  describe('Visual Styling - Masked State', () => {
    it('applies masked styling when hidden in read mode for sensitive fields', async () => {
      const field: FieldDTO = {
        value: 'secret',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      // Textarea defaults to visible - no masked styling
      expect(textarea).not.toHaveClass('cursor-not-allowed')

      // Toggle to hide
      const toggleButton = screen.getByTestId('toggle-field-testField')
      await userEvent.click(toggleButton)

      // Now has masked styling
      expect(textarea).toHaveClass('cursor-not-allowed')
      expect(textarea).toHaveClass('bg-muted')
    })

    it('does NOT apply masked styling in edit mode', () => {
      const field: FieldDTO = {
        value: 'secret',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea).not.toHaveClass('cursor-not-allowed')
      expect(textarea).not.toHaveClass('bg-muted')
    })

    it('does NOT apply masked styling when field is visible (default for textarea)', () => {
      const field: FieldDTO = {
        value: 'secret',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={true} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      // Textarea defaults to visible - no masked styling
      expect(textarea).not.toHaveClass('cursor-not-allowed')
      expect(textarea).not.toHaveClass('bg-muted')
    })
  })

  describe('Textarea Visibility Toggle in Edit Mode', () => {
    it('toggles visibility for textarea in edit mode', async () => {
      const field: FieldDTO = {
        value: 'my-ssh-key',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      // In edit mode, actual value is stored (masked visually via -webkit-text-security)
      expect(textarea).toHaveValue('my-ssh-key')

      // Toggle button reveals/hides content
      const toggleButton = screen.getByTestId('toggle-field-testField')
      await userEvent.click(toggleButton)

      // Value remains the same, but visual masking is toggled
      expect(textarea).toHaveValue('my-ssh-key')
    })

    // Note: The actual -webkit-text-security property only works in Chromium (Wails runtime).
    // happy-dom doesn't support it, so we verify via data-masked attribute instead.

    it('applies masking in edit mode when hidden (after toggle)', async () => {
      const field: FieldDTO = {
        value: 'my-ssh-key',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      // Textarea defaults to visible in edit mode
      expect(textarea).not.toHaveAttribute('data-masked')

      // Toggle to hide
      const toggleButton = screen.getByTestId('toggle-field-testField')
      await userEvent.click(toggleButton)

      // Per ADR-005: "Same UX as single-line Input"
      // data-masked indicates -webkit-text-security: disc is applied
      expect(textarea).toHaveAttribute('data-masked', 'true')
    })

    it('removes masking when visibility is toggled back to visible', async () => {
      const field: FieldDTO = {
        value: 'my-ssh-key',
        sensitive: true,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      const toggleButton = screen.getByTestId('toggle-field-testField')

      // Start visible (default), toggle to hide
      expect(textarea).not.toHaveAttribute('data-masked')
      await userEvent.click(toggleButton)
      expect(textarea).toHaveAttribute('data-masked', 'true')

      // Toggle back to visible - masking removed
      await userEvent.click(toggleButton)
      expect(textarea).not.toHaveAttribute('data-masked')
    })

    it('does NOT apply masking for non-sensitive textarea', () => {
      const field: FieldDTO = {
        value: 'public content',
        sensitive: false,
        inputType: 'textarea',
      }

      render(
        <FieldEditor {...defaultProps} field={field} readOnly={false} />
      )

      const textarea = screen.getByTestId('field-value-testField')
      expect(textarea).not.toHaveAttribute('data-masked')
    })
  })

  describe('Fixed-Length Mask - Security by Design', () => {
    it('uses fixed-length mask regardless of actual value length', () => {
      // This is intentional security design - not revealing length
      const shortField: FieldDTO = {
        value: 'ab',
        sensitive: true,
      }

      const longField: FieldDTO = {
        value: 'this-is-a-very-long-secret-key-value',
        sensitive: true,
      }

      const { rerender } = render(
        <FieldEditor {...defaultProps} field={shortField} readOnly={true} />
      )

      const input1 = screen.getByTestId('field-value-testField')
      expect(input1).toHaveValue('••••••••')

      rerender(
        <FieldEditor {...defaultProps} field={longField} readOnly={true} />
      )

      const input2 = screen.getByTestId('field-value-testField')
      expect(input2).toHaveValue('••••••••')
    })
  })
})
