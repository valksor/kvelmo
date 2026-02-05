import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from './client'

// ============================================================================
// Types
// ============================================================================

export interface BrowserTab {
  id: string
  title: string
  url: string
}

export interface BrowserStatus {
  connected: boolean
  error?: string
  host?: string
  port?: number
  tabs?: BrowserTab[]
}

export interface ScreenshotRequest {
  tab_id?: string
  format?: 'png' | 'jpeg'
  quality?: number
  full_page?: boolean
}

export interface ScreenshotResponse {
  success: boolean
  format: string
  data: string // base64
  size: number
  encoding: string
}

export interface ClickRequest {
  tab_id?: string
  selector: string
}

export interface TypeRequest {
  tab_id?: string
  selector: string
  text: string
  clear?: boolean
}

export interface EvalRequest {
  tab_id?: string
  expression: string
}

export interface EvalResponse {
  success: boolean
  result: unknown
}

export interface DOMElement {
  tagName: string
  textContent: string
  visible: boolean
  outerHTML?: string
}

export interface DOMRequest {
  tab_id?: string
  selector: string
  all?: boolean
  html?: boolean
  limit?: number
}

export interface DOMResponse {
  success: boolean
  element?: DOMElement
  elements?: DOMElement[]
  count?: number
  showing?: number
}

export interface ReloadRequest {
  tab_id?: string
  hard?: boolean
}

export interface CloseRequest {
  tab_id: string
}

export interface NavigateRequest {
  tab_id?: string
  url: string
}

// DevTools types

export interface NetworkRequest {
  tab_id?: string
  duration?: number
  capture_body?: boolean
  max_body_size?: number
}

export interface NetworkEntry {
  url: string
  method: string
  status?: number
  type?: string
  size?: number
  time?: number
  request_headers?: Record<string, string>
  response_headers?: Record<string, string>
  request_body?: string
  response_body?: string
}

export interface NetworkResponse {
  success: boolean
  requests: NetworkEntry[]
  count: number
}

export interface ConsoleRequest {
  tab_id?: string
  duration?: number
  level?: string
}

export interface ConsoleMessage {
  level: string
  text: string
  timestamp: string
  url?: string
  line?: number
}

export interface ConsoleResponse {
  success: boolean
  messages: ConsoleMessage[]
  count: number
}

export interface WebSocketRequest {
  tab_id?: string
  duration?: number
}

export interface WebSocketFrame {
  direction: 'sent' | 'received'
  data: string
  timestamp: string
  opcode?: number
}

export interface WebSocketResponse {
  success: boolean
  frames: WebSocketFrame[]
  count: number
}

export interface SourceRequest {
  tab_id?: string
}

export interface SourceResponse {
  success: boolean
  source: string
  length: number
}

export interface ScriptsRequest {
  tab_id?: string
}

export interface ScriptEntry {
  url: string
  source?: string
}

export interface ScriptsResponse {
  success: boolean
  scripts: ScriptEntry[]
  count: number
}

export interface StylesRequest {
  tab_id?: string
  selector: string
  computed?: boolean
  matched?: boolean
}

export interface StylesResponse {
  success: boolean
  selector: string
  computed?: Record<string, string>
  matched?: Array<{
    selector: string
    properties: Record<string, string>
  }>
}

export interface CoverageRequest {
  tab_id?: string
  duration?: number
  track_js?: boolean
  track_css?: boolean
}

export interface CoverageEntry {
  url: string
  total_bytes: number
  used_bytes: number
  percentage: number
}

export interface CoverageResponse {
  success: boolean
  summary: {
    js_total: number
    js_used: number
    js_percentage: number
    css_total: number
    css_used: number
    css_percentage: number
  }
  js_entries: CoverageEntry[]
  css_entries: CoverageEntry[]
}

// ============================================================================
// Hooks
// ============================================================================

/**
 * Hook for getting browser status
 */
export function useBrowserStatus() {
  return useQuery({
    queryKey: ['browser', 'status'],
    queryFn: () => apiRequest<BrowserStatus>('/browser/status'),
    refetchInterval: 10000,
  })
}

/**
 * Hook for navigating to a URL (opens new tab)
 */
export function useBrowserGoto() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (url: string) => {
      const response = await fetch('/api/v1/browser/goto', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ url }),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to navigate')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['browser'] })
    },
  })
}

/**
 * Hook for navigating an existing tab
 */
export function useBrowserNavigate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: NavigateRequest) => {
      const response = await fetch('/api/v1/browser/navigate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to navigate')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['browser'] })
    },
  })
}

/**
 * Hook for taking a screenshot
 */
export function useBrowserScreenshot() {
  return useMutation({
    mutationFn: async (data: ScreenshotRequest = {}) => {
      const response = await fetch('/api/v1/browser/screenshot', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to take screenshot')
      }
      return response.json() as Promise<ScreenshotResponse>
    },
  })
}

/**
 * Hook for clicking an element
 */
export function useBrowserClick() {
  return useMutation({
    mutationFn: async (data: ClickRequest) => {
      const response = await fetch('/api/v1/browser/click', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Click failed')
      }
      return response.json()
    },
  })
}

/**
 * Hook for typing text
 */
export function useBrowserType() {
  return useMutation({
    mutationFn: async (data: TypeRequest) => {
      const response = await fetch('/api/v1/browser/type', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Type failed')
      }
      return response.json()
    },
  })
}

/**
 * Hook for evaluating JavaScript
 */
export function useBrowserEval() {
  return useMutation({
    mutationFn: async (data: EvalRequest) => {
      const response = await fetch('/api/v1/browser/eval', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Eval failed')
      }
      return response.json() as Promise<EvalResponse>
    },
  })
}

/**
 * Hook for querying DOM elements
 */
export function useBrowserDOM() {
  return useMutation({
    mutationFn: async (data: DOMRequest) => {
      const response = await fetch('/api/v1/browser/dom', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'DOM query failed')
      }
      return response.json() as Promise<DOMResponse>
    },
  })
}

/**
 * Hook for reloading the page
 */
export function useBrowserReload() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: ReloadRequest = {}) => {
      const response = await fetch('/api/v1/browser/reload', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Reload failed')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['browser'] })
    },
  })
}

/**
 * Hook for closing a tab
 */
export function useBrowserClose() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: CloseRequest) => {
      const response = await fetch('/api/v1/browser/close', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Close failed')
      }
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['browser'] })
    },
  })
}

// ============================================================================
// DevTools Hooks
// ============================================================================

/**
 * Hook for monitoring network requests
 */
export function useBrowserNetwork() {
  return useMutation({
    mutationFn: async (data: NetworkRequest = {}) => {
      const response = await fetch('/api/v1/browser/network', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Network monitoring failed')
      }
      return response.json() as Promise<NetworkResponse>
    },
  })
}

/**
 * Hook for monitoring console logs
 */
export function useBrowserConsole() {
  return useMutation({
    mutationFn: async (data: ConsoleRequest = {}) => {
      const response = await fetch('/api/v1/browser/console', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Console monitoring failed')
      }
      return response.json() as Promise<ConsoleResponse>
    },
  })
}

/**
 * Hook for monitoring WebSocket frames
 */
export function useBrowserWebSocket() {
  return useMutation({
    mutationFn: async (data: WebSocketRequest = {}) => {
      const response = await fetch('/api/v1/browser/websocket', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'WebSocket monitoring failed')
      }
      return response.json() as Promise<WebSocketResponse>
    },
  })
}

/**
 * Hook for getting page source
 */
export function useBrowserSource() {
  return useMutation({
    mutationFn: async (data: SourceRequest = {}) => {
      const response = await fetch('/api/v1/browser/source', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to get source')
      }
      return response.json() as Promise<SourceResponse>
    },
  })
}

/**
 * Hook for listing scripts
 */
export function useBrowserScripts() {
  return useMutation({
    mutationFn: async (data: ScriptsRequest = {}) => {
      const response = await fetch('/api/v1/browser/scripts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to get scripts')
      }
      return response.json() as Promise<ScriptsResponse>
    },
  })
}

/**
 * Hook for inspecting CSS styles
 */
export function useBrowserStyles() {
  return useMutation({
    mutationFn: async (data: StylesRequest) => {
      const response = await fetch('/api/v1/browser/styles', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to get styles')
      }
      return response.json() as Promise<StylesResponse>
    },
  })
}

/**
 * Hook for measuring code coverage
 */
export function useBrowserCoverage() {
  return useMutation({
    mutationFn: async (data: CoverageRequest = {}) => {
      const response = await fetch('/api/v1/browser/coverage', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        const err = await response.text()
        throw new Error(err || 'Failed to get coverage')
      }
      return response.json() as Promise<CoverageResponse>
    },
  })
}
