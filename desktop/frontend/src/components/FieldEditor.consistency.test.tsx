/**
 * FieldEditor Behavior Consistency Tests
 *
 * Purpose: Automatically verify that Input and Textarea fields behave consistently
 * for sensitive field handling, per ADR-005 "Same UX as single-line Input".
 *
 * This test file prevents UX inconsistencies between field types by testing
 * both with the same scenarios and expectations.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FieldEditor, FieldDTO, InputType } from './FieldEditor'

// Mock Wails bindings
const mockViewSensitiveField = vi.fn().mockResolvedValue(undefined)
const mockCopyFieldValue = vi.fn().mockResolvedValue(undefined)

vi.mock('../../wailsjs/go/main/App', () => ({
  ViewSensitiveField: (...args: unknown[]) => mockViewSensitiveField(...args),
  CopyFieldValue: (...args: unknown[]) => mockCopyFieldValue(...args),
}))

vi.mock('@/hooks/useToast', () => ({
  useToast: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    toast: vi.fn(),
  }),
}))

/**
 * ADR-005 Behavior Matrix
 *
 * This matrix defines expected behavior for both Input and Textarea.
 * Any deviation indicates a UX inconsistency that must be fixed.
 *
 * | Scenario                    | Input              | Textarea           |
 * |-----------------------------|--------------------|--------------------|
 * | Edit + Sensitive + Hidden   | type=password      | data-masked=true   |
 * | Edit + Sensitive + Visible  | type=text          | data-masked=null   |
 * | Read + Sensitive + Hidden   | value=••••••••     | value=••••••••     |
 * | Read + Sensitive + Visible  | actual value       | actual value       |
 * | Eye button presence         | YES                | YES                |
 * | Eye button functional       | YES                | YES                |
 */

describe('FieldEditor ADR-005 Consistency Matrix', () => {
  const fieldTypes: InputType[] = ['text', 'textarea']

  const defaultProps = {
    secretKey: 'test-secret',
    fieldName: 'testField',
  }

  beforeEach(() => {
    mockViewSensitiveField.mockClear()
    mockCopyFieldValue.mockClear()
  })

  describe.each(fieldTypes)('InputType: %s', (inputType) => {
    const isTextarea = inputType === 'textarea'

    describe('Sensitive Field - Edit Mode', () => {
      it('shows eye button for visibility toggle', () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={false} />
        )

        const toggleButton = screen.getByTestId('toggle-field-testField')
        expect(toggleButton).toBeInTheDocument()
      })

      it('applies masking when hidden (default state)', () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={false} />
        )

        const element = screen.getByTestId('field-value-testField')

        if (isTextarea) {
          // Textarea uses data-masked attribute
          expect(element).toHaveAttribute('data-masked', 'true')
        } else {
          // Input uses type=password
          expect(element).toHaveAttribute('type', 'password')
        }
      })

      it('removes masking when visibility toggled', async () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={false} />
        )

        const element = screen.getByTestId('field-value-testField')
        const toggleButton = screen.getByTestId('toggle-field-testField')

        await userEvent.click(toggleButton)

        if (isTextarea) {
          expect(element).not.toHaveAttribute('data-masked')
        } else {
          expect(element).toHaveAttribute('type', 'text')
        }
      })

      it('allows typing (not readonly)', async () => {
        const onChange = vi.fn()
        const field: FieldDTO = {
          value: '',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor
            {...defaultProps}
            field={field}
            readOnly={false}
            onChange={onChange}
          />
        )

        const element = screen.getByTestId('field-value-testField')
        expect(element).not.toHaveAttribute('readonly')

        await userEvent.type(element, 'abc')
        expect(onChange).toHaveBeenCalledTimes(3)
      })
    })

    describe('Sensitive Field - Read Mode', () => {
      it('shows eye button for visibility toggle', () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={true} />
        )

        const toggleButton = screen.getByTestId('toggle-field-testField')
        expect(toggleButton).toBeInTheDocument()
      })

      it('shows fixed-length mask when hidden', () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={true} />
        )

        const element = screen.getByTestId('field-value-testField')
        expect(element).toHaveValue('••••••••')
      })

      it('shows actual value when visibility toggled', async () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={true} />
        )

        const element = screen.getByTestId('field-value-testField')
        const toggleButton = screen.getByTestId('toggle-field-testField')

        await userEvent.click(toggleButton)

        expect(element).toHaveValue('secret-value')
      })

      it('calls ViewSensitiveField for audit logging when revealed', async () => {
        const field: FieldDTO = {
          value: 'secret-value',
          sensitive: true,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={true} />
        )

        const toggleButton = screen.getByTestId('toggle-field-testField')
        await userEvent.click(toggleButton)

        expect(mockViewSensitiveField).toHaveBeenCalledWith(
          'test-secret',
          'testField'
        )
      })
    })

    describe('Non-Sensitive Field', () => {
      it('does NOT show eye button', () => {
        const field: FieldDTO = {
          value: 'public-value',
          sensitive: false,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={false} />
        )

        expect(
          screen.queryByTestId('toggle-field-testField')
        ).not.toBeInTheDocument()
      })

      it('shows actual value in edit mode', () => {
        const field: FieldDTO = {
          value: 'public-value',
          sensitive: false,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={false} />
        )

        const element = screen.getByTestId('field-value-testField')
        expect(element).toHaveValue('public-value')
      })

      it('shows actual value in read mode', () => {
        const field: FieldDTO = {
          value: 'public-value',
          sensitive: false,
          inputType,
        }

        render(
          <FieldEditor {...defaultProps} field={field} readOnly={true} />
        )

        const element = screen.getByTestId('field-value-testField')
        expect(element).toHaveValue('public-value')
      })
    })
  })
})
