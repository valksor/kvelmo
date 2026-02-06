export interface DiffRow {
  type: 'context' | 'removed' | 'added' | 'changed'
  leftLine?: number
  rightLine?: number
  leftText?: string
  rightText?: string
}

const HUNK_HEADER_RE = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/

export function parseUnifiedDiff(diff: string): DiffRow[] {
  const lines = diff.split('\n')
  const rows: DiffRow[] = []

  let leftLine = 0
  let rightLine = 0
  let inHunk = false
  let removed: Array<{ line: number; text: string }> = []
  let added: Array<{ line: number; text: string }> = []

  const flushChanges = () => {
    if (removed.length === 0 && added.length === 0) {
      return
    }

    const pairCount = Math.max(removed.length, added.length)
    for (let i = 0; i < pairCount; i += 1) {
      const left = removed[i]
      const right = added[i]
      if (left && right) {
        rows.push({
          type: 'changed',
          leftLine: left.line,
          rightLine: right.line,
          leftText: left.text,
          rightText: right.text,
        })
        continue
      }
      if (left) {
        rows.push({
          type: 'removed',
          leftLine: left.line,
          leftText: left.text,
        })
        continue
      }
      if (right) {
        rows.push({
          type: 'added',
          rightLine: right.line,
          rightText: right.text,
        })
      }
    }

    removed = []
    added = []
  }

  for (const line of lines) {
    const hunkMatch = line.match(HUNK_HEADER_RE)
    if (hunkMatch) {
      flushChanges()
      leftLine = Number.parseInt(hunkMatch[1], 10)
      rightLine = Number.parseInt(hunkMatch[2], 10)
      inHunk = true
      continue
    }

    if (!inHunk || line.length === 0) {
      continue
    }

    const prefix = line[0]
    const text = line.slice(1)

    if (prefix === ' ') {
      flushChanges()
      rows.push({
        type: 'context',
        leftLine,
        rightLine,
        leftText: text,
        rightText: text,
      })
      leftLine += 1
      rightLine += 1
      continue
    }

    if (prefix === '-') {
      removed.push({ line: leftLine, text })
      leftLine += 1
      continue
    }

    if (prefix === '+') {
      added.push({ line: rightLine, text })
      rightLine += 1
      continue
    }

    if (prefix === '\\') {
      continue
    }
  }

  flushChanges()

  return rows
}
