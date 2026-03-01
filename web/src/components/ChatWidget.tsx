import { useState, useRef, useEffect, useCallback } from 'react'
import { useChatStore, type ChatMessage } from '../stores/chatStore'
import { useGlobalStore } from '../stores/globalStore'
import { useScreenshotStore, getScreenshotById, formatScreenshotRef } from '../stores/screenshotStore'
import { ScreenshotBadge } from './ScreenshotBadge'

interface FileEntry {
  name: string
  path: string
  rel_path: string
  is_dir: boolean
}

interface ChatWidgetProps {
  embedded?: boolean
}

export function ChatWidget({ embedded = false }: ChatWidgetProps) {
  const { messages, isTyping, sendMessage, clearMessages, handleAction } = useChatStore()
  const { client, connected } = useGlobalStore()
  const { attachedIds, clearAttached, detach } = useScreenshotStore()
  const [input, setInput] = useState('')
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // File autocomplete state
  const [showAutocomplete, setShowAutocomplete] = useState(false)
  const [autocompleteQuery, setAutocompleteQuery] = useState('')
  const [autocompleteResults, setAutocompleteResults] = useState<FileEntry[]>([])
  const [autocompleteIndex, setAutocompleteIndex] = useState(0)
  const [autocompleteLoading, setAutocompleteLoading] = useState(false)
  const autocompleteRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isTyping])

  // Auto-resize textarea
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.style.height = 'auto'
      inputRef.current.style.height = `${Math.min(inputRef.current.scrollHeight, 120)}px`
    }
  }, [input])

  // Search files for autocomplete
  const searchFiles = useCallback(async (query: string) => {
    if (!client || !connected || query.length < 1) {
      setAutocompleteResults([])
      return
    }

    setAutocompleteLoading(true)
    try {
      const result = await client.call<{ entries: FileEntry[] }>('files.search', {
        query,
        max_results: 10
      })
      setAutocompleteResults(result.entries || [])
      setAutocompleteIndex(0)
    } catch (err) {
      console.error('File search failed:', err)
      setAutocompleteResults([])
    } finally {
      setAutocompleteLoading(false)
    }
  }, [client, connected])

  // Debounced file search
  useEffect(() => {
    if (!showAutocomplete || !autocompleteQuery) {
      return
    }

    const timer = setTimeout(() => {
      searchFiles(autocompleteQuery)
    }, 150)

    return () => clearTimeout(timer)
  }, [autocompleteQuery, showAutocomplete, searchFiles])

  // Handle input changes - detect @ mentions
  const handleInputChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    setInput(value)

    // Check for @ trigger
    const cursorPos = e.target.selectionStart
    const textBeforeCursor = value.slice(0, cursorPos)
    const atMatch = textBeforeCursor.match(/@(\S*)$/)

    if (atMatch) {
      setShowAutocomplete(true)
      setAutocompleteQuery(atMatch[1])
    } else {
      setShowAutocomplete(false)
      setAutocompleteQuery('')
    }
  }

  // Insert file reference
  const insertFileReference = (file: FileEntry) => {
    const cursorPos = inputRef.current?.selectionStart || input.length
    const textBeforeCursor = input.slice(0, cursorPos)
    const textAfterCursor = input.slice(cursorPos)

    // Find the @ position
    const atMatch = textBeforeCursor.match(/@(\S*)$/)
    if (atMatch) {
      const atPos = cursorPos - atMatch[0].length
      const newText = input.slice(0, atPos) + `@${file.rel_path}` + textAfterCursor
      setInput(newText)
    }

    setShowAutocomplete(false)
    setAutocompleteQuery('')
    inputRef.current?.focus()
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (input.trim() && !isTyping) {
      // Append screenshot references to message if any are attached
      let message = input
      if (attachedIds.length > 0) {
        const refs = attachedIds.map(id => formatScreenshotRef(id)).join(' ')
        message = `${input}\n\n${refs}`
      }
      sendMessage(message)
      setInput('')
      clearAttached()
      setShowAutocomplete(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // Handle autocomplete navigation
    if (showAutocomplete && autocompleteResults.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setAutocompleteIndex(i => Math.min(i + 1, autocompleteResults.length - 1))
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setAutocompleteIndex(i => Math.max(i - 1, 0))
        return
      }
      if (e.key === 'Tab' || e.key === 'Enter') {
        e.preventDefault()
        insertFileReference(autocompleteResults[autocompleteIndex])
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        setShowAutocomplete(false)
        return
      }
    }

    // Normal enter = submit
    if (e.key === 'Enter' && !e.shiftKey && !showAutocomplete) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  const content = (
    <div className="flex flex-col h-full">
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-3 space-y-4">
        {messages.length === 0 ? (
          <div className="h-full flex flex-col items-center justify-center text-base-content/50">
            <svg aria-hidden="true" className="w-12 h-12 mb-3 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            <p className="text-sm">Start a conversation</p>
            <p className="text-xs mt-1 opacity-70">Type @ to mention files</p>
          </div>
        ) : (
          messages.map(message => (
            <MessageBubble
              key={message.id}
              message={message}
              onAction={(actionId) => handleAction(message.id, actionId)}
            />
          ))
        )}

        {/* Typing indicator */}
        {isTyping && (
          <div className="flex items-start gap-2">
            <div className="w-7 h-7 rounded-full bg-secondary flex items-center justify-center text-secondary-content text-xs font-medium flex-shrink-0">
              AI
            </div>
            <div className="bg-base-200 rounded-lg px-3 py-2">
              <div className="flex gap-1">
                <span className="w-2 h-2 bg-base-content/40 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                <span className="w-2 h-2 bg-base-content/40 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                <span className="w-2 h-2 bg-base-content/40 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
              </div>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      <div className="flex-shrink-0 border-t border-base-300 p-3">
        {/* Attached screenshots indicator */}
        {attachedIds.length > 0 && (
          <div className="flex flex-wrap items-center gap-2 mb-2 p-2 bg-base-200 rounded-lg">
            <span className="text-xs text-base-content/60">Attached:</span>
            {attachedIds.map(id => {
              const screenshot = getScreenshotById(id)
              return (
                <span
                  key={id}
                  className="badge badge-sm badge-primary gap-1"
                >
                  {screenshot?.id || id.slice(0, 6)}
                  <button
                    type="button"
                    onClick={() => detach(id)}
                    className="hover:text-error"
                    aria-label={`Remove screenshot ${screenshot?.id || id.slice(0, 6)}`}
                  >
                    <svg aria-hidden="true" className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </span>
              )
            })}
            <button
              type="button"
              onClick={clearAttached}
              className="btn btn-ghost btn-xs text-base-content/60"
            >
              Clear all
            </button>
          </div>
        )}
        <form onSubmit={handleSubmit} className="relative">
          <div className="flex gap-2">
            <div className="flex-1 relative">
              <textarea
                ref={inputRef}
                value={input}
                onChange={handleInputChange}
                onKeyDown={handleKeyDown}
                placeholder="Type a message... (@ to mention files)"
                className="textarea textarea-bordered w-full resize-none min-h-[40px] max-h-[120px] pr-10 text-sm"
                rows={1}
                disabled={isTyping}
                role="combobox"
                aria-label="Message"
                aria-haspopup="listbox"
                aria-expanded={showAutocomplete}
                aria-controls="chat-autocomplete"
                aria-activedescendant={showAutocomplete && autocompleteResults.length > 0 ? `autocomplete-item-${autocompleteIndex}` : undefined}
              />
              {input.length > 0 && (
                <span className="absolute right-2 bottom-2 text-xs text-base-content/40">
                  {input.length}
                </span>
              )}

              {/* Autocomplete dropdown */}
              {showAutocomplete && (
                <div
                  ref={autocompleteRef}
                  id="chat-autocomplete"
                  role="listbox"
                  aria-label="File suggestions"
                  className="absolute bottom-full left-0 right-0 mb-1 bg-base-200 border border-base-300 rounded-lg shadow-lg max-h-48 overflow-auto z-50"
                >
                  {autocompleteLoading && (
                    <div className="px-3 py-2 text-sm text-base-content/60 flex items-center gap-2">
                      <span className="loading loading-spinner loading-xs"></span>
                      Searching...
                    </div>
                  )}
                  {!autocompleteLoading && autocompleteResults.length === 0 && autocompleteQuery && (
                    <div className="px-3 py-2 text-sm text-base-content/60">
                      No files found for "{autocompleteQuery}"
                    </div>
                  )}
                  {!autocompleteLoading && autocompleteResults.map((file, index) => (
                    <button
                      key={file.path}
                      id={`autocomplete-item-${index}`}
                      type="button"
                      role="option"
                      aria-selected={index === autocompleteIndex}
                      className={`w-full px-3 py-2 text-left text-sm flex items-center gap-2 hover:bg-base-300 transition-colors ${
                        index === autocompleteIndex ? 'bg-primary/20' : ''
                      }`}
                      onClick={() => insertFileReference(file)}
                      onMouseEnter={() => setAutocompleteIndex(index)}
                    >
                      <span className="text-base-content/60">
                        {file.is_dir ? 'D' : 'F'}
                      </span>
                      <span className="flex-1 truncate">
                        <span className="font-medium">{file.name}</span>
                        <span className="text-base-content/50 ml-2 text-xs">{file.rel_path}</span>
                      </span>
                    </button>
                  ))}
                  {!autocompleteLoading && autocompleteResults.length > 0 && (
                    <div className="px-3 py-1 text-[10px] text-base-content/40 border-t border-base-300 flex gap-2">
                      <span>Up/Down navigate</span>
                      <span>Tab/Enter select</span>
                      <span>Esc close</span>
                    </div>
                  )}
                </div>
              )}
            </div>
            <button
              type="submit"
              disabled={!input.trim() || isTyping}
              className="btn btn-primary btn-square"
              aria-label="Send message"
            >
              {isTyping ? (
                <span className="loading loading-spinner loading-sm" aria-hidden="true" />
              ) : (
                <svg aria-hidden="true" className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
              )}
            </button>
          </div>
        </form>

        {/* Quick actions */}
        <div className="flex gap-2 mt-2">
          {messages.length > 0 && (
            <button
              onClick={clearMessages}
              className="btn btn-ghost btn-xs text-base-content/60"
            >
              Clear chat
            </button>
          )}
        </div>
      </div>
    </div>
  )

  if (embedded) {
    return <div className="h-full">{content}</div>
  }

  return (
    <div className="card bg-base-200 h-full">
      <div className="card-body p-0 h-full">
        <div className="flex items-center justify-between px-4 py-3 border-b border-base-300">
          <h2 className="card-title text-base flex items-center gap-2">
            <svg aria-hidden="true" className="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            Chat
          </h2>
        </div>
        {content}
      </div>
    </div>
  )
}

// Message bubble component
interface MessageBubbleProps {
  message: ChatMessage
  onAction: (actionId: string) => void
}

function MessageBubble({ message, onAction }: MessageBubbleProps) {
  const isUser = message.role === 'user'
  const isSystem = message.role === 'system'
  const isSubagent = message.role === 'subagent'
  const isPermission = message.role === 'permission'

  // System messages: centered, minimal
  if (isSystem) {
    return (
      <div className="flex justify-center">
        <div className="text-xs text-base-content/50 bg-base-200 px-3 py-1 rounded-full">
          {message.content}
        </div>
      </div>
    )
  }

  // Subagent messages: show agent activity
  if (isSubagent && message.subagent) {
    const { status, type, description, duration } = message.subagent
    const icon = status === 'started' ? '▶' : status === 'completed' ? '✓' : '✗'
    const bgColor = status === 'started' ? 'bg-info/20 border-info/30'
      : status === 'completed' ? 'bg-success/20 border-success/30'
      : 'bg-error/20 border-error/30'
    const textColor = status === 'started' ? 'text-info'
      : status === 'completed' ? 'text-success'
      : 'text-error'

    return (
      <div className="flex justify-center">
        <div className={`text-xs px-3 py-1.5 rounded-lg border ${bgColor} flex items-center gap-2`}>
          <span className={textColor}>{icon}</span>
          <span className="font-medium">{type}</span>
          <span className="text-base-content/60">"{description}"</span>
          {duration != null && duration > 0 && status !== 'started' && (
            <span className="text-base-content/40">({(duration / 1000).toFixed(1)}s)</span>
          )}
        </div>
      </div>
    )
  }

  // Permission messages: show with danger styling
  if (isPermission && message.permission) {
    const { tool, dangerLevel, dangerReason } = message.permission
    const bgColor = dangerLevel === 'dangerous' ? 'bg-error/20 border-error/40'
      : dangerLevel === 'caution' ? 'bg-warning/20 border-warning/40'
      : 'bg-base-200 border-base-300'
    const iconColor = dangerLevel === 'dangerous' ? 'text-error'
      : dangerLevel === 'caution' ? 'text-warning'
      : 'text-base-content/50'

    return (
      <div className="flex justify-center">
        <div className={`text-xs px-3 py-1.5 rounded-lg border ${bgColor} max-w-md`}>
          <div className="flex items-center gap-2">
            <span className={iconColor}>
              {dangerLevel === 'safe' ? '🔒' : '⚠️'}
            </span>
            <span className="font-medium">Permission: {tool}</span>
          </div>
          {dangerLevel !== 'safe' && dangerReason && (
            <div className={`mt-1 ${iconColor}`}>
              {dangerLevel.toUpperCase()}: {dangerReason}
            </div>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className={`flex items-start gap-2 ${isUser ? 'flex-row-reverse' : ''}`}>
      {/* Avatar */}
      <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-medium flex-shrink-0 ${
        isUser
          ? 'bg-primary text-primary-content'
          : 'bg-secondary text-secondary-content'
      }`}>
        {isUser ? 'You' : 'AI'}
      </div>

      {/* Content */}
      <div className={`flex flex-col gap-1 max-w-[80%] ${isUser ? 'items-end' : 'items-start'}`}>
        <div className={`rounded-lg px-3 py-2 ${
          isUser
            ? 'bg-primary text-primary-content'
            : 'bg-base-200 text-base-content'
        }`}>
          <MessageContent content={message.content} isUser={isUser} />

          {message.status === 'streaming' && (
            <span className="inline-block w-1.5 h-4 ml-0.5 bg-current animate-pulse" />
          )}
        </div>

        {/* Actions */}
        {message.actions && message.actions.length > 0 && (
          <div className="flex gap-2 mt-1">
            {message.actions.map(action => (
              <button
                key={action.id}
                onClick={() => onAction(action.id)}
                className={`btn btn-xs ${
                  action.type === 'approve' ? 'btn-success' :
                  action.type === 'reject' ? 'btn-error' :
                  'btn-ghost'
                }`}
              >
                {action.label}
              </button>
            ))}
          </div>
        )}

        {/* Timestamp */}
        <span className="text-[10px] text-base-content/40">
          {message.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
      </div>
    </div>
  )
}

// Simple markdown-like content renderer
function MessageContent({ content, isUser }: { content: string; isUser: boolean }) {
  // Split content by code blocks
  const parts = content.split(/(```[\s\S]*?```)/g)

  return (
    <div className="text-sm whitespace-pre-wrap break-words">
      {parts.map((part, i) => {
        if (part.startsWith('```')) {
          // Code block
          const match = part.match(/```(\w*)\n?([\s\S]*?)```/)
          if (match) {
            const [, lang, code] = match
            return (
              <div key={i} className="my-2 -mx-1">
                {lang && (
                  <div className="text-[10px] uppercase tracking-wide opacity-60 mb-1">
                    {lang}
                  </div>
                )}
                <pre className={`p-2 rounded text-xs overflow-x-auto ${
                  isUser ? 'bg-primary-content/10' : 'bg-neutral text-neutral-content'
                }`}>
                  <code>{code.trim()}</code>
                </pre>
              </div>
            )
          }
        }
        // Regular text - handle inline code, bold, file refs, and screenshot references
        return (
          <span key={i}>
            {renderTextWithMentions(part, isUser)}
          </span>
        )
      })}
    </div>
  )
}

// Render text with file/screenshot references and inline code
function renderTextWithMentions(text: string, isUser: boolean): React.ReactNode[] {
  // Match @file-paths and @screenshot-ids
  const mentionRegex = /@(screenshot-[a-zA-Z0-9]+|[\w./-]+\.\w+|[\w/-]+)/g
  const parts: React.ReactNode[] = []
  let lastIndex = 0
  let match

  while ((match = mentionRegex.exec(text)) !== null) {
    // Add text before match (with inline code parsing)
    if (match.index > lastIndex) {
      const textBefore = text.slice(lastIndex, match.index)
      parts.push(...parseInlineCode(textBefore, isUser, `text-${lastIndex}`))
    }

    const mention = match[1]
    if (mention.startsWith('screenshot-')) {
      // Screenshot reference
      parts.push(<ScreenshotBadge key={`ss-${match.index}`} screenshotId={mention.replace('screenshot-', '')} />)
    } else {
      // File reference
      parts.push(
        <span
          key={`file-${match.index}`}
          className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-mono ${
            isUser ? 'bg-primary-content/20' : 'bg-primary/20 text-primary'
          }`}
          title={mention}
        >
          {mention.split('/').pop()}
        </span>
      )
    }
    lastIndex = mentionRegex.lastIndex
  }

  // Add remaining text
  if (lastIndex < text.length) {
    const remaining = text.slice(lastIndex)
    parts.push(...parseInlineCode(remaining, isUser, `text-${lastIndex}`))
  }

  return parts
}

// Parse inline code segments
function parseInlineCode(text: string, isUser: boolean, keyPrefix: string): React.ReactNode[] {
  return text.split(/(`[^`]+`)/).map((segment, j) => {
    if (segment.startsWith('`') && segment.endsWith('`')) {
      return (
        <code key={`${keyPrefix}-${j}`} className={`px-1 rounded text-xs ${
          isUser ? 'bg-primary-content/10' : 'bg-base-300'
        }`}>
          {segment.slice(1, -1)}
        </code>
      )
    }
    return <span key={`${keyPrefix}-${j}`}>{segment}</span>
  })
}

// Export icon for use in Widget
export function ChatIcon() {
  return (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
    </svg>
  )
}
