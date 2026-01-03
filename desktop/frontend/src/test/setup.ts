import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Mock Wails runtime
vi.mock('../../wailsjs/go/main/App', () => ({
  ViewSensitiveField: vi.fn().mockResolvedValue(undefined),
  CopyFieldValue: vi.fn().mockResolvedValue(undefined),
}))

// Mock clipboard API - define on window.navigator
Object.defineProperty(window.navigator, 'clipboard', {
  value: {
    writeText: vi.fn().mockResolvedValue(undefined),
    readText: vi.fn().mockResolvedValue(''),
  },
  writable: true,
  configurable: true,
})
