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
    mutationFn: (url: string) =>
      apiRequest('/browser/goto', {
        method: 'POST',
        body: JSON.stringify({ url }),
      }),
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
    mutationFn: (data: NavigateRequest) =>
      apiRequest('/browser/navigate', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
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
    mutationFn: (data: ScreenshotRequest = {}) =>
      apiRequest<ScreenshotResponse>('/browser/screenshot', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for clicking an element
 */
export function useBrowserClick() {
  return useMutation({
    mutationFn: (data: ClickRequest) =>
      apiRequest('/browser/click', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for typing text
 */
export function useBrowserType() {
  return useMutation({
    mutationFn: (data: TypeRequest) =>
      apiRequest('/browser/type', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for evaluating JavaScript
 */
export function useBrowserEval() {
  return useMutation({
    mutationFn: (data: EvalRequest) =>
      apiRequest<EvalResponse>('/browser/eval', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for querying DOM elements
 */
export function useBrowserDOM() {
  return useMutation({
    mutationFn: (data: DOMRequest) =>
      apiRequest<DOMResponse>('/browser/dom', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for reloading the page
 */
export function useBrowserReload() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: ReloadRequest = {}) =>
      apiRequest('/browser/reload', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
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
    mutationFn: (data: CloseRequest) =>
      apiRequest('/browser/close', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
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
    mutationFn: (data: NetworkRequest = {}) =>
      apiRequest<NetworkResponse>('/browser/network', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for monitoring console logs
 */
export function useBrowserConsole() {
  return useMutation({
    mutationFn: (data: ConsoleRequest = {}) =>
      apiRequest<ConsoleResponse>('/browser/console', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for monitoring WebSocket frames
 */
export function useBrowserWebSocket() {
  return useMutation({
    mutationFn: (data: WebSocketRequest = {}) =>
      apiRequest<WebSocketResponse>('/browser/websocket', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for getting page source
 */
export function useBrowserSource() {
  return useMutation({
    mutationFn: (data: SourceRequest = {}) =>
      apiRequest<SourceResponse>('/browser/source', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for listing scripts
 */
export function useBrowserScripts() {
  return useMutation({
    mutationFn: (data: ScriptsRequest = {}) =>
      apiRequest<ScriptsResponse>('/browser/scripts', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for inspecting CSS styles
 */
export function useBrowserStyles() {
  return useMutation({
    mutationFn: (data: StylesRequest) =>
      apiRequest<StylesResponse>('/browser/styles', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}

/**
 * Hook for measuring code coverage
 */
export function useBrowserCoverage() {
  return useMutation({
    mutationFn: (data: CoverageRequest = {}) =>
      apiRequest<CoverageResponse>('/browser/coverage', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  })
}
