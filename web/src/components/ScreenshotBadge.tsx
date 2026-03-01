import { getScreenshotById, useScreenshotStore } from '../stores/screenshotStore'
import { useLayoutStore } from '../stores/layoutStore'

interface ScreenshotBadgeProps {
  screenshotId: string
}

export function ScreenshotBadge({ screenshotId }: ScreenshotBadgeProps) {
  const screenshot = getScreenshotById(screenshotId)
  const { select } = useScreenshotStore()
  const { openTab } = useLayoutStore()

  const handleClick = () => {
    // Select the screenshot and open screenshots tab
    select(screenshotId)
    openTab({ id: 'screenshots', type: 'screenshots', title: 'Screenshots' })
  }

  if (!screenshot) {
    return (
      <span className="badge badge-ghost badge-sm opacity-50" title="Screenshot not found">
        📷 {screenshotId.slice(0, 6)}...
      </span>
    )
  }

  return (
    <button
      onClick={handleClick}
      className="badge badge-primary badge-sm gap-1 cursor-pointer hover:badge-secondary transition-colors"
      title={`View screenshot: ${screenshot.filename}`}
    >
      📷 {screenshot.id}
    </button>
  )
}

// Component to parse and render text with screenshot references
interface ScreenshotTextProps {
  text: string
}

export function ScreenshotText({ text }: ScreenshotTextProps) {
  // Parse @screenshot-{id} patterns
  const regex = /@screenshot-([a-zA-Z0-9]+)/g
  const parts: (string | { type: 'screenshot'; id: string })[] = []
  let lastIndex = 0
  let match

  while ((match = regex.exec(text)) !== null) {
    // Add text before match
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index))
    }
    // Add screenshot reference
    parts.push({ type: 'screenshot', id: match[1] })
    lastIndex = regex.lastIndex
  }

  // Add remaining text
  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex))
  }

  return (
    <>
      {parts.map((part, i) => {
        if (typeof part === 'string') {
          return <span key={i}>{part}</span>
        }
        return <ScreenshotBadge key={i} screenshotId={part.id} />
      })}
    </>
  )
}
