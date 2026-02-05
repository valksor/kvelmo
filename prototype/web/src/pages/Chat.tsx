import { useState, useRef, useEffect, useCallback } from 'react'
import { Send, Loader2, StopCircle, Trash2, AlertCircle, Terminal, MessageSquare } from 'lucide-react'
import { apiRequest } from '@/api/client'

interface Message {
  id: string
  role: 'user' | 'assistant' | 'system' | 'error'
  content: string
  timestamp: Date
  isStreaming?: boolean
}

interface ChatResponse {
  success: boolean
  message?: string
  messages?: Array<{
    role: string
    content: string
    timestamp: string
  }>
}

interface CommandResponse {
  success: boolean
  message?: string
  state?: string
}

// Known workflow commands that should be routed to the command handler
const WORKFLOW_COMMANDS = new Set([
  'start', 'plan', 'implement', 'review', 'continue', 'finish', 'abandon',
  'undo', 'redo', 'status', 'st', 'cost', 'budget', 'list', 'note',
  'quick', 'find', 'memory', 'simplify', 'label', 'specification', 'spec',
  'chat', 'answer', 'help', 'library', 'question',
])

export default function Chat() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const abortControllerRef = useRef<AbortController | null>(null)

  // Auto-scroll to bottom when new messages arrive
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, scrollToBottom])

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const generateId = () => Math.random().toString(36).substring(2, 11)

  const isCommand = (text: string): boolean => {
    const firstWord = text.trim().split(/\s+/)[0].toLowerCase()
    return WORKFLOW_COMMANDS.has(firstWord)
  }

  const parseCommand = (text: string): { command: string; args: string[] } => {
    const parts = text.trim().split(/\s+/)
    return {
      command: parts[0].toLowerCase(),
      args: parts.slice(1),
    }
  }

  const sendMessage = async () => {
    const trimmedInput = input.trim()
    if (!trimmedInput || isLoading) return

    setError(null)
    const userMessage: Message = {
      id: generateId(),
      role: 'user',
      content: trimmedInput,
      timestamp: new Date(),
    }
    setMessages(prev => [...prev, userMessage])
    setInput('')
    setIsLoading(true)

    // Create abort controller for cancellation
    abortControllerRef.current = new AbortController()

    try {
      if (isCommand(trimmedInput)) {
        // Execute as workflow command
        const { command, args } = parseCommand(trimmedInput)
        const response = await apiRequest<CommandResponse>('/interactive/command', {
          method: 'POST',
          body: JSON.stringify({ command, args }),
          signal: abortControllerRef.current.signal,
        })

        const systemMessage: Message = {
          id: generateId(),
          role: 'system',
          content: response.message || (response.success ? 'Command executed' : 'Command failed'),
          timestamp: new Date(),
        }
        setMessages(prev => [...prev, systemMessage])

        if (response.state) {
          // Publish state change for other components
          window.dispatchEvent(new CustomEvent('workflow-state-change', {
            detail: { state: response.state }
          }))
        }
      } else {
        // Send as chat message to agent
        // Add a placeholder message for streaming
        const assistantMessageId = generateId()
        setMessages(prev => [...prev, {
          id: assistantMessageId,
          role: 'assistant',
          content: '',
          timestamp: new Date(),
          isStreaming: true,
        }])

        const response = await apiRequest<ChatResponse>('/interactive/chat', {
          method: 'POST',
          body: JSON.stringify({ message: trimmedInput }),
          signal: abortControllerRef.current.signal,
        })

        // Update the streaming message with final content
        if (response.messages && response.messages.length > 0) {
          const lastMessage = response.messages[response.messages.length - 1]
          if (lastMessage.role === 'assistant') {
            setMessages(prev => prev.map(msg =>
              msg.id === assistantMessageId
                ? { ...msg, content: lastMessage.content, isStreaming: false }
                : msg
            ))
          } else {
            // Remove placeholder if no assistant response
            setMessages(prev => prev.filter(msg => msg.id !== assistantMessageId))
          }
        } else if (response.message) {
          // Fallback to message field
          setMessages(prev => prev.map(msg =>
            msg.id === assistantMessageId
              ? { ...msg, content: response.message!, isStreaming: false }
              : msg
          ))
        } else {
          // Remove placeholder if empty response
          setMessages(prev => prev.filter(msg => msg.id !== assistantMessageId))
        }
      }
    } catch (err) {
      if (err instanceof Error && err.name === 'AbortError') {
        setMessages(prev => [...prev, {
          id: generateId(),
          role: 'system',
          content: 'Request cancelled',
          timestamp: new Date(),
        }])
      } else {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error'
        setMessages(prev => [...prev, {
          id: generateId(),
          role: 'error',
          content: errorMessage,
          timestamp: new Date(),
        }])
        setError(errorMessage)
      }
    } finally {
      setIsLoading(false)
      abortControllerRef.current = null
    }
  }

  const stopRequest = async () => {
    // Cancel the fetch request
    abortControllerRef.current?.abort()

    // Also call the server's stop endpoint
    try {
      await apiRequest('/interactive/stop', { method: 'POST' })
    } catch {
      // Ignore errors on stop
    }
  }

  const clearMessages = () => {
    setMessages([])
    setError(null)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  const formatTime = (date: Date) => {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <MessageSquare size={24} className="text-primary" />
          <h1 className="text-2xl font-bold">Interactive Chat</h1>
        </div>
        <button
          className="btn btn-ghost btn-sm"
          onClick={clearMessages}
          disabled={messages.length === 0}
        >
          <Trash2 size={16} />
          Clear
        </button>
      </div>

      {/* Messages area */}
      <div className="flex-1 overflow-y-auto rounded-lg bg-base-200/50 p-4 space-y-3">
        {messages.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-base-content/50">
            <Terminal size={48} className="mb-4" />
            <p className="text-lg font-medium">Interactive Chat</p>
            <p className="text-sm mt-2">
              Chat with the AI agent or run workflow commands.
            </p>
            <div className="mt-4 text-xs space-y-1">
              <p><code className="bg-base-300 px-2 py-0.5 rounded">help</code> - Show available commands</p>
              <p><code className="bg-base-300 px-2 py-0.5 rounded">status</code> - Show current task status</p>
              <p>Or just type a message to chat with the agent.</p>
            </div>
          </div>
        ) : (
          <>
            {messages.map((msg) => (
              <MessageBubble key={msg.id} message={msg} formatTime={formatTime} />
            ))}
            <div ref={messagesEndRef} />
          </>
        )}
      </div>

      {/* Error display */}
      {error && (
        <div className="alert alert-error mt-2">
          <AlertCircle size={16} />
          <span className="text-sm">{error}</span>
          <button className="btn btn-ghost btn-xs" onClick={() => setError(null)}>
            Dismiss
          </button>
        </div>
      )}

      {/* Input area */}
      <div className="mt-4 flex gap-2">
        <textarea
          ref={inputRef}
          className="textarea textarea-bordered flex-1 resize-none"
          placeholder={isLoading ? 'Waiting for response...' : 'Type a message or command...'}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isLoading}
          rows={2}
        />
        {isLoading ? (
          <button
            className="btn btn-error"
            onClick={stopRequest}
            title="Stop request"
          >
            <StopCircle size={20} />
          </button>
        ) : (
          <button
            className="btn btn-primary"
            onClick={sendMessage}
            disabled={!input.trim()}
            title="Send (Enter)"
          >
            <Send size={20} />
          </button>
        )}
      </div>

      {/* Quick commands */}
      <div className="mt-2 flex flex-wrap gap-1">
        {['status', 'help', 'plan', 'implement', 'review', 'finish'].map((cmd) => (
          <button
            key={cmd}
            className="btn btn-ghost btn-xs"
            onClick={() => {
              setInput(cmd)
              inputRef.current?.focus()
            }}
            disabled={isLoading}
          >
            {cmd}
          </button>
        ))}
      </div>
    </div>
  )
}

interface MessageBubbleProps {
  message: Message
  formatTime: (date: Date) => string
}

function MessageBubble({ message, formatTime }: MessageBubbleProps) {
  const { role, content, timestamp, isStreaming } = message

  const bubbleClasses = {
    user: 'chat-end',
    assistant: 'chat-start',
    system: 'chat-start',
    error: 'chat-start',
  }

  const bubbleColorClasses = {
    user: 'chat-bubble-primary',
    assistant: 'chat-bubble',
    system: 'chat-bubble chat-bubble-info',
    error: 'chat-bubble chat-bubble-error',
  }

  const roleLabels = {
    user: 'You',
    assistant: 'Agent',
    system: 'System',
    error: 'Error',
  }

  return (
    <div className={`chat ${bubbleClasses[role]}`}>
      <div className="chat-header mb-1">
        <span className="font-medium text-xs">{roleLabels[role]}</span>
        <time className="text-xs opacity-50 ml-2">{formatTime(timestamp)}</time>
      </div>
      <div className={`chat-bubble ${bubbleColorClasses[role]} whitespace-pre-wrap`}>
        {content || (isStreaming && <Loader2 className="w-4 h-4 animate-spin" />)}
      </div>
    </div>
  )
}
