# Add greeting utility function

Create a simple greeting utility function in this TypeScript project.

## Requirements

1. Create `./src/greet.ts` containing:
   - Export a function `greet(name: string): string`
   - Return format: `Hello, {name}!`

2. Create `./src/greet.test.ts` containing:
   - Use Vitest (already configured in the project)
   - Test that `greet("World")` returns `"Hello, World!"`
   - Use `describe`, `it`, and `expect` from Vitest

## Implementation Notes

- Files go in the existing `src/` directory (same level as `src/index.ts`)
- Use standard ES module syntax (`export function`)
- No additional runtime dependencies needed (Vitest is the project's test runner)

## Acceptance Criteria

- `src/greet.ts` exports `greet` function
- `src/greet.test.ts` has passing test
- No TypeScript errors
