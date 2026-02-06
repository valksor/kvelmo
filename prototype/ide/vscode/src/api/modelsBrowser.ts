// ============================================================================
// Browser Models
// ============================================================================

export interface BrowserStatusResponse {
  connected: boolean;
  host?: string;
  port?: number;
  tabs?: BrowserTab[];
  error?: string;
}

export interface BrowserTab {
  id: string;
  title: string;
  url: string;
}

export interface BrowserTabsResponse {
  tabs: BrowserTab[];
  count: number;
}

export interface BrowserGotoRequest {
  url: string;
}

export interface BrowserGotoResponse {
  success: boolean;
  tab?: BrowserTab;
}

export interface BrowserNavigateRequest {
  tab_id?: string;
  url: string;
}

export interface BrowserNavigateResponse {
  success: boolean;
  message?: string;
}

export interface BrowserClickRequest {
  tab_id?: string;
  selector: string;
}

export interface BrowserClickResponse {
  success: boolean;
  selector?: string;
}

export interface BrowserTypeRequest {
  tab_id?: string;
  selector: string;
  text: string;
  clear?: boolean;
}

export interface BrowserTypeResponse {
  success: boolean;
  selector?: string;
}

export interface BrowserEvalRequest {
  tab_id?: string;
  expression: string;
}

export interface BrowserEvalResponse {
  success: boolean;
  result?: unknown;
}

export interface BrowserDOMRequest {
  tab_id?: string;
  selector: string;
  all?: boolean;
  html?: boolean;
  limit?: number;
}

export interface BrowserDOMElement {
  tag_name: string;
  text_content?: string;
  outer_html?: string;
  visible: boolean;
}

export interface BrowserDOMResponse {
  success: boolean;
  element?: BrowserDOMElement;
  elements?: BrowserDOMElement[];
  count?: number;
  showing?: number;
}

export interface BrowserScreenshotRequest {
  tab_id?: string;
  format?: string;
  quality?: number;
  full_page?: boolean;
}

export interface BrowserScreenshotResponse {
  success: boolean;
  format?: string;
  data?: string;
  size?: number;
  encoding?: string;
}

export interface BrowserReloadRequest {
  tab_id?: string;
  hard?: boolean;
}

export interface BrowserReloadResponse {
  success: boolean;
  message?: string;
}

export interface BrowserCloseRequest {
  tab_id: string;
}

export interface BrowserCloseResponse {
  success: boolean;
  message?: string;
}

export interface BrowserConsoleRequest {
  tab_id?: string;
  duration?: number;
  level?: string;
}

export interface BrowserConsoleMessage {
  level: string;
  text: string;
  timestamp?: string;
}

export interface BrowserConsoleResponse {
  success: boolean;
  messages?: BrowserConsoleMessage[];
  count?: number;
}

export interface BrowserNetworkRequest {
  tab_id?: string;
  duration?: number;
  capture_body?: boolean;
  max_body_size?: number;
}

export interface BrowserNetworkEntry {
  method: string;
  url: string;
  status?: number;
  status_text?: string;
  timestamp: string;
  request_body?: string;
  response_body?: string;
}

export interface BrowserNetworkResponse {
  success: boolean;
  requests?: BrowserNetworkEntry[];
  count?: number;
}
