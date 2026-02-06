import { useMemo } from 'react'
import { parseUnifiedDiff } from '@/utils/unifiedDiff'

interface VisualUnifiedDiffProps {
  diff: string
}

function DiffCell({
  lineNumber,
  text,
  tone,
}: {
  lineNumber?: number
  text?: string
  tone: 'context' | 'added' | 'removed'
}) {
  const toneClass =
    tone === 'added'
      ? 'bg-success/10 text-success-content'
      : tone === 'removed'
        ? 'bg-error/10 text-error-content'
        : 'bg-base-100 text-base-content'

  return (
    <div className={`grid grid-cols-[3rem_1fr] text-xs font-mono ${toneClass}`}>
      <div className="px-2 py-1 text-right text-base-content/50 border-r border-base-300/60 select-none">
        {lineNumber ?? ''}
      </div>
      <pre className="px-2 py-1 whitespace-pre-wrap break-words min-h-[1.75rem]">{text ?? ''}</pre>
    </div>
  )
}

export function VisualUnifiedDiff({ diff }: VisualUnifiedDiffProps) {
  const rows = useMemo(() => parseUnifiedDiff(diff), [diff])

  if (rows.length === 0) {
    return (
      <div className="text-sm text-base-content/60">
        Visual diff is unavailable for this patch format. Switch to Raw view.
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-base-300 overflow-hidden">
      <div className="grid grid-cols-2 bg-base-300/60 text-xs font-semibold uppercase tracking-wide">
        <div className="px-3 py-2 border-r border-base-300">Before</div>
        <div className="px-3 py-2">After</div>
      </div>

      <div className="max-h-[60vh] overflow-auto">
        {rows.map((row, index) => (
          <div key={`${row.type}-${row.leftLine ?? 'n'}-${row.rightLine ?? 'n'}-${index}`} className="grid grid-cols-2 border-t border-base-300/50">
            <DiffCell
              lineNumber={row.leftLine}
              text={row.leftText}
              tone={row.type === 'removed' || row.type === 'changed' ? 'removed' : 'context'}
            />
            <DiffCell
              lineNumber={row.rightLine}
              text={row.rightText}
              tone={row.type === 'added' || row.type === 'changed' ? 'added' : 'context'}
            />
          </div>
        ))}
      </div>
    </div>
  )
}
