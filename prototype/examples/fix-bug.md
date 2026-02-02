---
title: Fix Email Validation Accepting Invalid Addresses
type: fix
priority: high
labels:
  - bug
  - validation
---

# Fix Email Validation Accepting Invalid Addresses

The email validation in the user registration form currently accepts addresses without a TLD (e.g., `user@localhost`). This causes delivery failures when sending confirmation emails.

## Steps to Reproduce

1. Submit the registration form with email `test@invalid`
2. Form accepts it without error
3. Confirmation email fails to send

## Expected Behavior

Reject email addresses that don't have a valid domain with a TLD (e.g., `user@example.com` is valid, `user@localhost` is not).

## Constraints

- Don't break existing valid addresses (international domains, plus-addressing like `user+tag@example.com`)
- Show a clear, user-friendly error message
- Add test cases covering edge cases
