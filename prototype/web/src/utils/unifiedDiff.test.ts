import { describe, expect, it } from 'vitest'
import { parseUnifiedDiff } from './unifiedDiff'

describe('parseUnifiedDiff', () => {
  it('parses new-file additions into added rows', () => {
    const diff = [
      'diff --git a/hello.md b/hello.md',
      'new file mode 100644',
      'index 0000000..e9168db',
      '--- /dev/null',
      '+++ b/hello.md',
      '@@ -0,0 +1 @@',
      '+# Hello World',
      '\\ No newline at end of file',
    ].join('\n')

    const rows = parseUnifiedDiff(diff)
    expect(rows).toHaveLength(1)
    expect(rows[0]).toMatchObject({
      type: 'added',
      rightLine: 1,
      rightText: '# Hello World',
    })
  })

  it('parses changed lines into paired changed rows', () => {
    const diff = [
      'diff --git a/file.txt b/file.txt',
      'index 123..456 100644',
      '--- a/file.txt',
      '+++ b/file.txt',
      '@@ -1,3 +1,3 @@',
      ' line 1',
      '-line 2 old',
      '+line 2 new',
      ' line 3',
    ].join('\n')

    const rows = parseUnifiedDiff(diff)
    expect(rows.map((row) => row.type)).toEqual(['context', 'changed', 'context'])
    expect(rows[1]).toMatchObject({
      leftLine: 2,
      rightLine: 2,
      leftText: 'line 2 old',
      rightText: 'line 2 new',
    })
  })
})
