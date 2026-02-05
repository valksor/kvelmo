import { useState, useEffect, useCallback } from 'react'
import { Bell, X, CheckCircle, AlertCircle, HelpCircle, Trash2 } from 'lucide-react'
import { useWorkflowSSE, type QuestionData } from '@/hooks/useWorkflowSSE'

export interface Notification {
  id: string
  type: 'success' | 'error' | 'question' | 'info'
  title: string
  message?: string
  timestamp: Date
  read: boolean
}

export function NotificationCenter() {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [isOpen, setIsOpen] = useState(false)

  const addNotification = useCallback((notif: Omit<Notification, 'id' | 'timestamp' | 'read'>) => {
    const newNotification: Notification = {
      ...notif,
      id: crypto.randomUUID(),
      timestamp: new Date(),
      read: false,
    }
    setNotifications((prev) => [newNotification, ...prev].slice(0, 50)) // Keep max 50
  }, [])

  // Subscribe to SSE events
  useWorkflowSSE({
    onStateChange: (state) => {
      if (state === 'done') {
        addNotification({
          type: 'success',
          title: 'Workflow Complete',
          message: 'Task has been completed successfully',
        })
      } else if (state === 'failed') {
        addNotification({
          type: 'error',
          title: 'Workflow Failed',
          message: 'Task encountered an error',
        })
      }
    },
    onQuestion: (question: QuestionData) => {
      addNotification({
        type: 'question',
        title: 'Question Pending',
        message: question.question?.slice(0, 100) + (question.question && question.question.length > 100 ? '...' : ''),
      })
    },
    onError: (error: string) => {
      addNotification({
        type: 'error',
        title: 'Error',
        message: error,
      })
    },
  })

  const unreadCount = notifications.filter((n) => !n.read).length

  const markAsRead = (id: string) => {
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read: true } : n))
    )
  }

  const markAllAsRead = () => {
    setNotifications((prev) => prev.map((n) => ({ ...n, read: true })))
  }

  const dismiss = (id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id))
  }

  const clearAll = () => {
    setNotifications([])
    setIsOpen(false)
  }

  const getIcon = (type: Notification['type']) => {
    switch (type) {
      case 'success':
        return <CheckCircle size={16} className="text-success" />
      case 'error':
        return <AlertCircle size={16} className="text-error" />
      case 'question':
        return <HelpCircle size={16} className="text-warning" />
      case 'info':
        return <Bell size={16} className="text-info" />
    }
  }

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement
      if (!target.closest('.notification-center')) {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('click', handleClickOutside)
      return () => document.removeEventListener('click', handleClickOutside)
    }
  }, [isOpen])

  return (
    <div className="notification-center dropdown dropdown-end">
      <button
        tabIndex={0}
        className="btn btn-ghost btn-sm btn-circle relative"
        onClick={() => setIsOpen(!isOpen)}
      >
        <Bell size={18} />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 badge badge-sm badge-error">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div
          tabIndex={0}
          className="dropdown-content z-[100] mt-2 w-80 bg-base-100 rounded-box shadow-lg border border-base-300"
        >
          {/* Header */}
          <div className="flex items-center justify-between p-3 border-b border-base-300">
            <h3 className="font-medium">Notifications</h3>
            <div className="flex items-center gap-1">
              {unreadCount > 0 && (
                <button
                  onClick={markAllAsRead}
                  className="btn btn-ghost btn-xs"
                >
                  Mark all read
                </button>
              )}
              {notifications.length > 0 && (
                <button
                  onClick={clearAll}
                  className="btn btn-ghost btn-xs text-error"
                >
                  <Trash2 size={14} />
                </button>
              )}
            </div>
          </div>

          {/* Notifications list */}
          <div className="max-h-80 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="p-8 text-center text-base-content/60">
                <Bell size={24} className="mx-auto mb-2 opacity-50" />
                <p className="text-sm">No notifications</p>
              </div>
            ) : (
              <ul className="divide-y divide-base-200">
                {notifications.map((notif) => (
                  <li
                    key={notif.id}
                    className={`p-3 hover:bg-base-200 transition-colors ${
                      !notif.read ? 'bg-base-200/50' : ''
                    }`}
                    onClick={() => markAsRead(notif.id)}
                  >
                    <div className="flex items-start gap-2">
                      {getIcon(notif.type)}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <span className={`text-sm ${!notif.read ? 'font-medium' : ''}`}>
                            {notif.title}
                          </span>
                          <button
                            onClick={(e) => {
                              e.stopPropagation()
                              dismiss(notif.id)
                            }}
                            className="btn btn-ghost btn-xs opacity-50 hover:opacity-100"
                          >
                            <X size={12} />
                          </button>
                        </div>
                        {notif.message && (
                          <p className="text-xs text-base-content/60 truncate mt-0.5">
                            {notif.message}
                          </p>
                        )}
                        <p className="text-xs text-base-content/40 mt-1">
                          {formatRelativeTime(notif.timestamp)}
                        </p>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function formatRelativeTime(date: Date): string {
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (seconds < 60) return 'Just now'
  if (minutes < 60) return `${minutes}m ago`
  if (hours < 24) return `${hours}h ago`
  return date.toLocaleDateString()
}
