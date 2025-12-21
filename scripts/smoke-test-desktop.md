# Desktop Smoke Test Checklist

Run this checklist on each OS after downloading the release binary.

## Pre-requisites

- [ ] Downloaded the correct binary for your OS
- [ ] macOS: Ran `xattr -d com.apple.quarantine <app>` if needed
- [ ] Windows: Allowed SmartScreen if prompted

## Test Steps

### 1. Application Launch
- [ ] Double-click the application
- [ ] Lock screen appears
- [ ] No crash or error dialogs

### 2. Vault Creation
- [ ] Click "Create Vault" or similar
- [ ] Enter master password: `smoke-test-password`
- [ ] Confirm password
- [ ] Vault created successfully
- [ ] Main screen (secret list) appears

### 3. Add Secret
- [ ] Click "Add Secret" or "+"
- [ ] Enter key: `test/smoke-key`
- [ ] Enter value: `smoke-value-123`
- [ ] Click Save
- [ ] Secret appears in list

### 4. View Secret
- [ ] Click on `test/smoke-key` in list
- [ ] Value is initially masked (*****)
- [ ] Click "Show" or eye icon
- [ ] Value `smoke-value-123` is visible
- [ ] Click "Hide" - value is masked again

### 5. Copy to Clipboard
- [ ] Click "Copy" button
- [ ] "Copied" notification appears
- [ ] Paste in another app - value is correct
- [ ] Wait 30 seconds
- [ ] Paste again - clipboard should be empty or different

### 6. Lock Vault
- [ ] Click "Lock" or lock icon
- [ ] Lock screen appears
- [ ] Cannot access secrets without password

### 7. Unlock Vault
- [ ] Enter master password
- [ ] Click Unlock
- [ ] Secret list reappears
- [ ] `test/smoke-key` is still there

### 8. Delete Secret
- [ ] Select `test/smoke-key`
- [ ] Click "Delete"
- [ ] Confirm deletion
- [ ] Secret removed from list

### 9. Session Timeout (Optional)
- [ ] Unlock vault
- [ ] Wait 15 minutes without interaction
- [ ] Vault should auto-lock

## Results

| Test | macOS | Windows | Linux |
|------|-------|---------|-------|
| Launch | | | |
| Vault Creation | | | |
| Add Secret | | | |
| View Secret | | | |
| Copy to Clipboard | | | |
| Lock Vault | | | |
| Unlock Vault | | | |
| Delete Secret | | | |

## Notes

```
Date:
Version:
Tester:
OS Version:

Issues found:


```
