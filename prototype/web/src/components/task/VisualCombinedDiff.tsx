import { useMemo } from 'react'
import { parseUnifiedDiff, type DiffRow } from '@/utils/unifiedDiff'

interface VisualCombinedDiffProps {
  diff: string
}

interface LineProps {
  prefix: ' ' | '-' | '+'
  lineNumber?: number
  text?: string
  tone: 'context' | 'added' | 'removed'
}

function DiffLine({ prefix, lineNumber, text, tone }: LineProps) {
  const toneClass =
    tone === 'added'
      ? 'bg-success/10 text-success-content'
      : tone === 'removed'
        ? 'bg-error/10 text-error-content'
        : 'bg-base-100 text-base-content'

  const prefixClass =
    tone === 'added'
      ? 'text-success'
      : tone === 'removed'
        ? 'text-error'
        : 'text-base-content/50'

  return (
    <div className={`grid grid-cols-[1.5rem_3rem_1fr] text-xs font-mono ${toneClass}`}>
      <div className={`px-1 py-1 text-center select-none font-bold ${prefixClass}`}>{prefix}</div>
      <div className="px-2 py-1 text-right text-base-content/50 border-r border-base-300/60 select-none">
        {lineNumber ?? ''}
      </div>
      <pre className="px-2 py-1 whitespace-pre-wrap break-words min-h-[1.75rem]">{text ?? ''}</pre>
    </div>
  )
}

function rowToLines(row: DiffRow): LineProps[] {
  switch (row.type) {
    case 'context':
      return [{ prefix: ' ', lineNumber: row.leftLine, text: row.leftText, tone: 'context' }]
    case 'removed':
      return [{ prefix: '-', lineNumber: row.leftLine, text: row.leftText, tone: 'removed' }]
    case 'added':
      return [{ prefix: '+', lineNumber: row.rightLine, text: row.rightText, tone: 'added' }]
    case 'changed':
      return [
        { prefix: '-', lineNumber: row.leftLine, text: row.leftText, tone: 'removed' },
        { prefix: '+', lineNumber: row.rightLine, text: row.rightText, tone: 'added' },
      ]
  }
}

export function VisualCombinedDiff({ diff }: VisualCombinedDiffProps) {
  const rows = useMemo(() => parseUnifiedDiff(diff), [diff])
  const lines = useMemo(() => rows.flatMap(rowToLines), [rows])

  if (lines.length === 0) {
    return (
      <div className="text-sm text-base-content/60">
        Visual diff is unavailable for this patch format. Switch to Raw view.
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-base-300 overflow-hidden">
      <div className="max-h-[60vh] overflow-auto">
        {lines.map((line, index) => (
          <DiffLine
            key={`${line.prefix}-${line.lineNumber ?? 'n'}-${index}`}
            prefix={line.prefix}
            lineNumber={line.lineNumber}
            text={line.text}
            tone={line.tone}
          />
        ))}
      </div>
    </div>
  )
}
