import { useState } from 'react'
import {
  Globe,
  Brain,
  Shield,
  Layers,
  Loader2,
  ExternalLink,
  Search,
  AlertTriangle,
  CheckCircle,
  XCircle,
  RefreshCw,
  Play,
  AlertCircle,
  Camera,
  MousePointer,
  Type,
  Code,
  Network,
  Terminal,
  FileCode,
  BarChart3,
  Copy,
  Download,
  ChevronDown,
  ChevronUp,
  Trash2,
} from 'lucide-react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiRequest } from '@/api/client'
import {
  useBrowserStatus,
  useBrowserGoto,
  useBrowserScreenshot,
  useBrowserClick,
  useBrowserType,
  useBrowserEval,
  useBrowserDOM,
  useBrowserReload,
  useBrowserClose,
  useBrowserNetwork,
  useBrowserConsole,
  useBrowserWebSocket,
  useBrowserSource,
  useBrowserCoverage,
  type DOMElement,
  type NetworkEntry,
  type ConsoleMessage,
  type WebSocketFrame,
} from '@/api/browser'

// =============================================================================
// Browser Panel
// =============================================================================

type DevToolsTab = 'network' | 'console' | 'websocket' | 'source' | 'coverage'

export function BrowserPanel() {
  const [url, setUrl] = useState('')
  const [showInteractions, setShowInteractions] = useState(false)
  const [showDevTools, setShowDevTools] = useState(false)
  const [devToolsTab, setDevToolsTab] = useState<DevToolsTab>('network')

  // Interaction states
  const [selector, setSelector] = useState('')
  const [typeText, setTypeText] = useState('')
  const [typeClear, setTypeClear] = useState(false)
  const [evalExpr, setEvalExpr] = useState('')
  const [domQueryAll, setDomQueryAll] = useState(false)
  const [domIncludeHtml, setDomIncludeHtml] = useState(false)
  const [screenshotFullPage, setScreenshotFullPage] = useState(false)
  const [screenshotFormat, setScreenshotFormat] = useState<'png' | 'jpeg'>('png')

  // DevTools states
  const [networkDuration, setNetworkDuration] = useState(5)
  const [networkCaptureBody, setNetworkCaptureBody] = useState(false)
  const [consoleDuration, setConsoleDuration] = useState(5)
  const [consoleLevel, setConsoleLevel] = useState('')
  const [wsDuration, setWsDuration] = useState(5)
  const [coverageDuration, setCoverageDuration] = useState(5)
  const [coverageTrackJs, setCoverageTrackJs] = useState(true)
  const [coverageTrackCss, setCoverageTrackCss] = useState(true)

  // Results
  const [screenshotData, setScreenshotData] = useState<string | null>(null)
  const [evalResult, setEvalResult] = useState<unknown>(null)
  const [domElements, setDomElements] = useState<DOMElement[]>([])
  const [networkRequests, setNetworkRequests] = useState<NetworkEntry[]>([])
  const [consoleMessages, setConsoleMessages] = useState<ConsoleMessage[]>([])
  const [wsFrames, setWsFrames] = useState<WebSocketFrame[]>([])
  const [pageSource, setPageSource] = useState<string | null>(null)

  const { data: status, isLoading, refetch } = useBrowserStatus()
  const gotoMutation = useBrowserGoto()
  const screenshotMutation = useBrowserScreenshot()
  const clickMutation = useBrowserClick()
  const typeMutation = useBrowserType()
  const evalMutation = useBrowserEval()
  const domMutation = useBrowserDOM()
  const reloadMutation = useBrowserReload()
  const closeMutation = useBrowserClose()
  const networkMutation = useBrowserNetwork()
  const consoleMutation = useBrowserConsole()
  const wsMutation = useBrowserWebSocket()
  const sourceMutation = useBrowserSource()
  const coverageMutation = useBrowserCoverage()

  const handleGoto = (e: React.FormEvent) => {
    e.preventDefault()
    if (url.trim()) {
      gotoMutation.mutate(url.trim(), { onSuccess: () => setUrl('') })
    }
  }

  const handleScreenshot = async () => {
    const result = await screenshotMutation.mutateAsync({
      format: screenshotFormat,
      full_page: screenshotFullPage,
    })
    setScreenshotData(`data:image/${result.format};base64,${result.data}`)
  }

  const handleClick = async () => {
    if (!selector.trim()) return
    await clickMutation.mutateAsync({ selector: selector.trim() })
  }

  const handleType = async () => {
    if (!selector.trim()) return
    await typeMutation.mutateAsync({
      selector: selector.trim(),
      text: typeText,
      clear: typeClear,
    })
  }

  const handleEval = async () => {
    if (!evalExpr.trim()) return
    const result = await evalMutation.mutateAsync({ expression: evalExpr.trim() })
    setEvalResult(result.result)
  }

  const handleDomQuery = async () => {
    if (!selector.trim()) return
    const result = await domMutation.mutateAsync({
      selector: selector.trim(),
      all: domQueryAll,
      html: domIncludeHtml,
    })
    if (result.elements) {
      setDomElements(result.elements)
    } else if (result.element) {
      setDomElements([result.element])
    } else {
      setDomElements([])
    }
  }

  const handleNetworkMonitor = async () => {
    const result = await networkMutation.mutateAsync({
      duration: networkDuration,
      capture_body: networkCaptureBody,
    })
    setNetworkRequests(result.requests)
  }

  const handleConsoleMonitor = async () => {
    const result = await consoleMutation.mutateAsync({
      duration: consoleDuration,
      level: consoleLevel || undefined,
    })
    setConsoleMessages(result.messages)
  }

  const handleWsMonitor = async () => {
    const result = await wsMutation.mutateAsync({ duration: wsDuration })
    setWsFrames(result.frames)
  }

  const handleGetSource = async () => {
    const result = await sourceMutation.mutateAsync({})
    setPageSource(result.source)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  const isConnected = status?.connected

  return (
    <div className="space-y-4">
      {/* Status card */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex items-center justify-between">
            <h3 className="card-title">Browser Status</h3>
            <div className="flex gap-2">
              {isConnected && (
                <>
                  <button
                    className="btn btn-ghost btn-sm"
                    onClick={() => reloadMutation.mutate({})}
                    disabled={reloadMutation.isPending}
                    title="Reload page"
                  >
                    <RefreshCw size={16} />
                  </button>
                </>
              )}
              <button className="btn btn-ghost btn-sm" onClick={() => refetch()}>
                <RefreshCw size={16} />
              </button>
            </div>
          </div>

          {isConnected ? (
            <div className="flex items-center gap-2 text-success">
              <CheckCircle size={16} />
              <span>Connected to {status.host}:{status.port}</span>
            </div>
          ) : (
            <div className="flex items-center gap-2 text-error">
              <XCircle size={16} />
              <span>{status?.error || 'Not connected'}</span>
            </div>
          )}
        </div>
      </div>

      {/* Navigate form */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title">Navigate</h3>
          <form onSubmit={handleGoto} className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="form-control flex-1">
              <label className="label py-1" htmlFor="browser-navigate-url">
                <span className="label-text">URL</span>
              </label>
              <input
                id="browser-navigate-url"
                type="url"
                className="input input-bordered w-full"
                placeholder="https://example.com"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                disabled={!isConnected}
              />
            </div>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={!isConnected || !url.trim() || gotoMutation.isPending}
            >
              {gotoMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Go'}
            </button>
          </form>
        </div>
      </div>

      {/* Tabs list */}
      {isConnected && status.tabs && status.tabs.length > 0 && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <h3 className="card-title">Open Tabs ({status.tabs.length})</h3>
            <div className="space-y-2">
              {status.tabs.map((tab) => (
                <div
                  key={tab.id}
                  className="flex items-center gap-3 p-2 rounded-lg bg-base-200/50 hover:bg-base-200"
                >
                  <Globe size={16} className="text-base-content/50 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="font-medium truncate">{tab.title || 'Untitled'}</p>
                    <p className="text-xs text-base-content/50 truncate">{tab.url}</p>
                  </div>
                  <button
                    className="btn btn-ghost btn-xs text-error"
                    onClick={() => closeMutation.mutate({ tab_id: tab.id })}
                    title="Close tab"
                  >
                    <Trash2 size={14} />
                  </button>
                  <a
                    href={tab.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="btn btn-ghost btn-xs"
                  >
                    <ExternalLink size={14} />
                  </a>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Interactions (Collapsible) */}
      {isConnected && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <button
              type="button"
              className="flex items-center justify-between w-full text-left"
              onClick={() => setShowInteractions(!showInteractions)}
            >
              <h3 className="card-title">
                <MousePointer size={18} />
                Interactions
              </h3>
              {showInteractions ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
            </button>

            {showInteractions && (
              <div className="mt-4 space-y-6">
                {/* Screenshot */}
                <div className="p-4 rounded-lg bg-base-200/50">
                  <h4 className="font-medium flex items-center gap-2 mb-3">
                    <Camera size={16} />
                    Screenshot
                  </h4>
                  <div className="grid grid-cols-1 md:grid-cols-[220px_1fr_auto] gap-4 items-end">
                    <div className="form-control">
                      <label className="label py-1">
                        <span className="label-text">Format</span>
                      </label>
                      <select
                        className="select select-bordered"
                        value={screenshotFormat}
                        onChange={(e) => setScreenshotFormat(e.target.value as 'png' | 'jpeg')}
                      >
                        <option value="png">PNG</option>
                        <option value="jpeg">JPEG</option>
                      </select>
                    </div>
                    <div className="form-control">
                      <label className="label cursor-pointer justify-start gap-3 py-1">
                        <input
                          type="checkbox"
                          checked={screenshotFullPage}
                          onChange={(e) => setScreenshotFullPage(e.target.checked)}
                          className="checkbox checkbox-primary"
                        />
                        <span className="label-text">Full page</span>
                      </label>
                    </div>
                    <button
                      className="btn btn-primary"
                      onClick={handleScreenshot}
                      disabled={screenshotMutation.isPending}
                    >
                      {screenshotMutation.isPending ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Camera size={16} />
                      )}
                      Capture
                    </button>
                  </div>
                  {screenshotData && (
                    <div className="mt-3">
                      <img src={screenshotData} alt="Screenshot" className="max-w-full rounded border" />
                      <a
                        href={screenshotData}
                        download={`screenshot.${screenshotFormat}`}
                        className="btn btn-sm btn-ghost mt-2"
                      >
                        <Download size={14} />
                        Download
                      </a>
                    </div>
                  )}
                </div>

                {/* Click & Type */}
                <div className="p-4 rounded-lg bg-base-200/50">
                  <h4 className="font-medium flex items-center gap-2 mb-3">
                    <Type size={16} />
                    Click & Type
                  </h4>
                  <div className="space-y-3">
                    <div className="form-control">
                      <label className="label py-1">
                        <span className="label-text">CSS Selector</span>
                      </label>
                      <input
                        type="text"
                        className="input input-bordered"
                        placeholder="#submit, .btn-primary, [data-testid='login']"
                        value={selector}
                        onChange={(e) => setSelector(e.target.value)}
                      />
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <button
                        className="btn btn-outline"
                        onClick={handleClick}
                        disabled={!selector.trim() || clickMutation.isPending}
                      >
                        {clickMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <MousePointer size={16} />}
                        Click
                      </button>
                    </div>
                    <div className="grid grid-cols-1 lg:grid-cols-[1fr_auto_auto] gap-3 items-end">
                      <div className="form-control">
                        <label className="label py-1">
                          <span className="label-text">Text to type</span>
                        </label>
                        <input
                          type="text"
                          className="input input-bordered"
                          placeholder="Hello world"
                          value={typeText}
                          onChange={(e) => setTypeText(e.target.value)}
                        />
                      </div>
                      <div className="form-control">
                        <label className="label cursor-pointer justify-start gap-3 py-1">
                          <input
                            type="checkbox"
                            checked={typeClear}
                            onChange={(e) => setTypeClear(e.target.checked)}
                            className="checkbox checkbox-primary"
                          />
                          <span className="label-text">Clear first</span>
                        </label>
                      </div>
                      <button
                        className="btn btn-outline"
                        onClick={handleType}
                        disabled={!selector.trim() || typeMutation.isPending}
                      >
                        {typeMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Type size={16} />}
                        Type
                      </button>
                    </div>
                  </div>
                </div>

                {/* Eval JS */}
                <div className="p-4 rounded-lg bg-base-200/50">
                  <h4 className="font-medium flex items-center gap-2 mb-3">
                    <Code size={16} />
                    Evaluate JavaScript
                  </h4>
                  <div className="space-y-3">
                    <textarea
                      className="textarea textarea-bordered w-full h-24 font-mono text-sm"
                      placeholder="document.title"
                      value={evalExpr}
                      onChange={(e) => setEvalExpr(e.target.value)}
                    />
                    <button
                      className="btn btn-sm btn-primary"
                      onClick={handleEval}
                      disabled={!evalExpr.trim() || evalMutation.isPending}
                    >
                      {evalMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play size={14} />}
                      Run
                    </button>
                    {evalResult !== null && (
                      <div className="p-2 rounded bg-base-300 font-mono text-sm overflow-x-auto">
                        <pre>{JSON.stringify(evalResult, null, 2)}</pre>
                      </div>
                    )}
                  </div>
                </div>

                {/* DOM Query */}
                <div className="p-4 rounded-lg bg-base-200/50">
                  <h4 className="font-medium flex items-center gap-2 mb-3">
                    <Search size={16} />
                    DOM Query
                  </h4>
                  <div className="space-y-3">
                    <p className="text-xs text-base-content/60">
                      Uses the CSS selector from the Click & Type section above.
                    </p>
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                      <div className="flex flex-wrap gap-4">
                        <label className="label cursor-pointer gap-2 py-0">
                          <input
                            type="checkbox"
                            checked={domQueryAll}
                            onChange={(e) => setDomQueryAll(e.target.checked)}
                            className="checkbox checkbox-primary"
                          />
                          <span className="label-text text-sm">Query all</span>
                        </label>
                        <label className="label cursor-pointer gap-2 py-0">
                          <input
                            type="checkbox"
                            checked={domIncludeHtml}
                            onChange={(e) => setDomIncludeHtml(e.target.checked)}
                            className="checkbox checkbox-primary"
                          />
                          <span className="label-text text-sm">Include HTML</span>
                        </label>
                      </div>
                      <button
                        className="btn btn-primary"
                        onClick={handleDomQuery}
                        disabled={!selector.trim() || domMutation.isPending}
                      >
                        {domMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search size={16} />}
                        Query
                      </button>
                    </div>
                    {domElements.length > 0 && (
                      <div className="space-y-2 mt-2">
                        {domElements.map((el, i) => (
                          <div key={i} className="p-2 rounded bg-base-300 text-sm">
                            <div className="flex items-center gap-2">
                              <span className="badge badge-sm">{el.tagName}</span>
                              {el.visible ? (
                                <span className="badge badge-success badge-xs">visible</span>
                              ) : (
                                <span className="badge badge-ghost badge-xs">hidden</span>
                              )}
                            </div>
                            <p className="mt-1 text-base-content/70 line-clamp-2">{el.textContent}</p>
                            {el.outerHTML && (
                              <pre className="mt-1 text-xs overflow-x-auto max-h-24 bg-base-200 p-1 rounded">
                                {el.outerHTML}
                              </pre>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* DevTools (Collapsible) */}
      {isConnected && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <button
              type="button"
              className="flex items-center justify-between w-full text-left"
              onClick={() => setShowDevTools(!showDevTools)}
            >
              <h3 className="card-title">
                <Terminal size={18} />
                DevTools
              </h3>
              {showDevTools ? <ChevronUp size={18} /> : <ChevronDown size={18} />}
            </button>

            {showDevTools && (
              <div className="mt-4">
                {/* DevTools tabs */}
                <div role="tablist" className="tabs tabs-boxed mb-4">
                  <button
                    role="tab"
                    className={`tab gap-1 ${devToolsTab === 'network' ? 'tab-active' : ''}`}
                    onClick={() => setDevToolsTab('network')}
                  >
                    <Network size={14} />
                    Network
                  </button>
                  <button
                    role="tab"
                    className={`tab gap-1 ${devToolsTab === 'console' ? 'tab-active' : ''}`}
                    onClick={() => setDevToolsTab('console')}
                  >
                    <Terminal size={14} />
                    Console
                  </button>
                  <button
                    role="tab"
                    className={`tab gap-1 ${devToolsTab === 'websocket' ? 'tab-active' : ''}`}
                    onClick={() => setDevToolsTab('websocket')}
                  >
                    <Globe size={14} />
                    WebSocket
                  </button>
                  <button
                    role="tab"
                    className={`tab gap-1 ${devToolsTab === 'source' ? 'tab-active' : ''}`}
                    onClick={() => setDevToolsTab('source')}
                  >
                    <FileCode size={14} />
                    Source
                  </button>
                  <button
                    role="tab"
                    className={`tab gap-1 ${devToolsTab === 'coverage' ? 'tab-active' : ''}`}
                    onClick={() => setDevToolsTab('coverage')}
                  >
                    <BarChart3 size={14} />
                    Coverage
                  </button>
                </div>

                {/* Network tab */}
                {devToolsTab === 'network' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-[220px_1fr_auto] gap-4 items-end">
                      <div className="form-control">
                        <label className="label py-1">
                          <span className="label-text">Duration (seconds)</span>
                        </label>
                        <input
                          type="number"
                          className="input input-bordered w-24"
                          value={networkDuration}
                          onChange={(e) => setNetworkDuration(Number(e.target.value))}
                          min={1}
                          max={30}
                        />
                      </div>
                      <div className="form-control">
                        <label className="label cursor-pointer justify-start gap-3 py-1">
                          <input
                            type="checkbox"
                            checked={networkCaptureBody}
                            onChange={(e) => setNetworkCaptureBody(e.target.checked)}
                            className="checkbox checkbox-primary"
                          />
                          <span className="label-text">Capture bodies</span>
                        </label>
                      </div>
                      <button
                        className="btn btn-primary"
                        onClick={handleNetworkMonitor}
                        disabled={networkMutation.isPending}
                      >
                        {networkMutation.isPending ? (
                          <>
                            <Loader2 className="w-4 h-4 animate-spin" />
                            Monitoring...
                          </>
                        ) : (
                          <>
                            <Network size={14} />
                            Monitor
                          </>
                        )}
                      </button>
                    </div>
                    {networkRequests.length > 0 && (
                      <div className="overflow-x-auto">
                        <table className="table table-xs">
                          <thead>
                            <tr>
                              <th>Method</th>
                              <th>URL</th>
                              <th>Status</th>
                              <th>Type</th>
                              <th>Size</th>
                              <th>Time</th>
                            </tr>
                          </thead>
                          <tbody>
                            {networkRequests.map((req, i) => (
                              <tr key={i} className="hover">
                                <td><span className="badge badge-sm">{req.method}</span></td>
                                <td className="max-w-xs truncate font-mono text-xs">{req.url}</td>
                                <td>
                                  <span className={`badge badge-sm ${req.status && req.status >= 400 ? 'badge-error' : 'badge-success'}`}>
                                    {req.status || '-'}
                                  </span>
                                </td>
                                <td className="text-xs">{req.type || '-'}</td>
                                <td className="text-xs">{req.size ? `${(req.size / 1024).toFixed(1)}KB` : '-'}</td>
                                <td className="text-xs">{req.time ? `${req.time}ms` : '-'}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                )}

                {/* Console tab */}
                {devToolsTab === 'console' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-[220px_260px_auto] gap-4 items-end">
                      <div className="form-control">
                        <label className="label py-1">
                          <span className="label-text">Duration (seconds)</span>
                        </label>
                        <input
                          type="number"
                          className="input input-bordered w-24"
                          value={consoleDuration}
                          onChange={(e) => setConsoleDuration(Number(e.target.value))}
                          min={1}
                          max={30}
                        />
                      </div>
                      <div className="form-control">
                        <label className="label py-1">
                          <span className="label-text">Level filter</span>
                        </label>
                        <select
                          className="select select-bordered"
                          value={consoleLevel}
                          onChange={(e) => setConsoleLevel(e.target.value)}
                        >
                          <option value="">All</option>
                          <option value="log">Log</option>
                          <option value="warning">Warning</option>
                          <option value="error">Error</option>
                        </select>
                      </div>
                      <button
                        className="btn btn-primary"
                        onClick={handleConsoleMonitor}
                        disabled={consoleMutation.isPending}
                      >
                        {consoleMutation.isPending ? (
                          <>
                            <Loader2 className="w-4 h-4 animate-spin" />
                            Monitoring...
                          </>
                        ) : (
                          <>
                            <Terminal size={14} />
                            Monitor
                          </>
                        )}
                      </button>
                    </div>
                    {consoleMessages.length > 0 && (
                      <div className="space-y-1 max-h-64 overflow-y-auto">
                        {consoleMessages.map((msg, i) => (
                          <div
                            key={i}
                            className={`p-2 rounded text-sm font-mono ${
                              msg.level === 'error' ? 'bg-error/10 text-error' :
                              msg.level === 'warning' ? 'bg-warning/10 text-warning' :
                              'bg-base-200'
                            }`}
                          >
                            <span className="badge badge-xs mr-2">{msg.level}</span>
                            {msg.text}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}

                {/* WebSocket tab */}
                {devToolsTab === 'websocket' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-[220px_auto] gap-4 items-end">
                      <div className="form-control">
                        <label className="label py-1">
                          <span className="label-text">Duration (seconds)</span>
                        </label>
                        <input
                          type="number"
                          className="input input-bordered w-24"
                          value={wsDuration}
                          onChange={(e) => setWsDuration(Number(e.target.value))}
                          min={1}
                          max={30}
                        />
                      </div>
                      <button
                        className="btn btn-primary"
                        onClick={handleWsMonitor}
                        disabled={wsMutation.isPending}
                      >
                        {wsMutation.isPending ? (
                          <>
                            <Loader2 className="w-4 h-4 animate-spin" />
                            Monitoring...
                          </>
                        ) : (
                          <>
                            <Globe size={14} />
                            Monitor
                          </>
                        )}
                      </button>
                    </div>
                    {wsFrames.length > 0 ? (
                      <div className="space-y-1 max-h-64 overflow-y-auto">
                        {wsFrames.map((frame, i) => (
                          <div key={i} className="p-2 rounded bg-base-200 text-sm">
                            <span className={`badge badge-xs mr-2 ${frame.direction === 'sent' ? 'badge-info' : 'badge-success'}`}>
                              {frame.direction}
                            </span>
                            <span className="font-mono text-xs">{frame.data.substring(0, 100)}</span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-base-content/60">No WebSocket frames captured.</p>
                    )}
                  </div>
                )}

                {/* Source tab */}
                {devToolsTab === 'source' && (
                  <div className="space-y-4">
                    <button
                      className="btn btn-sm btn-primary"
                      onClick={handleGetSource}
                      disabled={sourceMutation.isPending}
                    >
                      {sourceMutation.isPending ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <FileCode size={14} />
                      )}
                      Get Page Source
                    </button>
                    {pageSource && (
                      <div className="relative">
                        <button
                          className="btn btn-ghost btn-xs absolute top-2 right-2"
                          onClick={() => copyToClipboard(pageSource)}
                        >
                          <Copy size={14} />
                        </button>
                        <pre className="p-3 rounded bg-base-200 text-xs font-mono overflow-x-auto max-h-96">
                          {pageSource.substring(0, 5000)}
                          {pageSource.length > 5000 && '...'}
                        </pre>
                        <p className="text-xs text-base-content/50 mt-1">
                          {pageSource.length.toLocaleString()} characters total
                        </p>
                      </div>
                    )}
                  </div>
                )}

                {/* Coverage tab */}
                {devToolsTab === 'coverage' && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-[220px_auto_auto_auto] gap-4 items-end">
                      <div className="form-control">
                        <label className="label py-1">
                          <span className="label-text">Duration (seconds)</span>
                        </label>
                        <input
                          type="number"
                          className="input input-bordered w-24"
                          value={coverageDuration}
                          onChange={(e) => setCoverageDuration(Number(e.target.value))}
                          min={1}
                          max={30}
                        />
                      </div>
                      <div className="form-control">
                        <label className="label cursor-pointer justify-start gap-3 py-1">
                          <input
                            type="checkbox"
                            checked={coverageTrackJs}
                            onChange={(e) => setCoverageTrackJs(e.target.checked)}
                            className="checkbox checkbox-primary"
                          />
                          <span className="label-text">Track JS</span>
                        </label>
                      </div>
                      <div className="form-control">
                        <label className="label cursor-pointer justify-start gap-3 py-1">
                          <input
                            type="checkbox"
                            checked={coverageTrackCss}
                            onChange={(e) => setCoverageTrackCss(e.target.checked)}
                            className="checkbox checkbox-primary"
                          />
                          <span className="label-text">Track CSS</span>
                        </label>
                      </div>
                      <button
                        className="btn btn-primary"
                        onClick={() => coverageMutation.mutate({
                          duration: coverageDuration,
                          track_js: coverageTrackJs,
                          track_css: coverageTrackCss,
                        })}
                        disabled={coverageMutation.isPending}
                      >
                        {coverageMutation.isPending ? (
                          <>
                            <Loader2 className="w-4 h-4 animate-spin" />
                            Measuring...
                          </>
                        ) : (
                          <>
                            <BarChart3 size={14} />
                            Measure
                          </>
                        )}
                      </button>
                    </div>
                    {coverageMutation.data && (
                      <div className="space-y-4">
                        <div className="stats stats-horizontal shadow-sm bg-base-200/50">
                          <div className="stat py-2 px-4">
                            <div className="stat-title text-xs">JS Used</div>
                            <div className="stat-value text-lg text-success">
                              {coverageMutation.data.summary.js_percentage.toFixed(1)}%
                            </div>
                          </div>
                          <div className="stat py-2 px-4">
                            <div className="stat-title text-xs">CSS Used</div>
                            <div className="stat-value text-lg text-info">
                              {coverageMutation.data.summary.css_percentage.toFixed(1)}%
                            </div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Mutation errors */}
      {gotoMutation.isError && (
        <div className="alert alert-error">
          <AlertCircle size={16} />
          <span>{gotoMutation.error.message}</span>
        </div>
      )}
    </div>
  )
}

// =============================================================================
// Memory Panel
// =============================================================================

interface MemoryResult {
  task_id: string
  type: string
  score: number
  content: string
  metadata?: Record<string, unknown>
}

interface MemorySearchResponse {
  results: MemoryResult[]
  count: number
}

export function MemoryPanel() {
  const [query, setQuery] = useState('')
  const [limit, setLimit] = useState(5)
  const [types, setTypes] = useState<string[]>([])

  const searchMutation = useMutation({
    mutationFn: (searchQuery: string) => {
      const params = new URLSearchParams({ q: searchQuery, limit: limit.toString() })
      if (types.length > 0) {
        params.set('types', types.join(','))
      }
      return apiRequest<MemorySearchResponse>(`/memory/search?${params.toString()}`)
    },
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      searchMutation.mutate(query.trim())
    }
  }

  const toggleType = (type: string) => {
    setTypes((prev) =>
      prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
    )
  }

  const typeOptions = [
    { value: 'code', label: 'Code' },
    { value: 'spec', label: 'Specifications' },
    { value: 'session', label: 'Sessions' },
    { value: 'solution', label: 'Solutions' },
    { value: 'decision', label: 'Decisions' },
    { value: 'error', label: 'Errors' },
  ]

  return (
    <div className="space-y-4">
      {/* Search form */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title">
            <Brain size={20} />
            Memory Search
          </h3>
          <p className="text-sm text-base-content/60">
            Search for similar tasks, decisions, and code patterns from past work.
          </p>

          <form onSubmit={handleSearch} className="space-y-4 mt-4">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
              <div className="form-control flex-1">
                <label className="label py-1" htmlFor="memory-search-query">
                  <span className="label-text">Search query</span>
                </label>
                <input
                  id="memory-search-query"
                  type="text"
                  className="input input-bordered w-full"
                  placeholder="Search query..."
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              </div>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={!query.trim() || searchMutation.isPending}
              >
                {searchMutation.isPending ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <Search size={16} />
                )}
                Search
              </button>
            </div>

            <div className="flex flex-wrap gap-2">
              <span className="text-sm text-base-content/60">Filter by type:</span>
              {typeOptions.map((opt) => (
                <label key={opt.value} className="label cursor-pointer gap-2">
                  <input
                    type="checkbox"
                    className="checkbox checkbox-primary"
                    checked={types.includes(opt.value)}
                    onChange={() => toggleType(opt.value)}
                  />
                  <span className="label-text text-sm">{opt.label}</span>
                </label>
              ))}
            </div>

            <div className="form-control w-32">
              <label className="label py-1">
                <span className="label-text">Results limit</span>
              </label>
              <input
                type="number"
                className="input input-bordered w-full"
                value={limit}
                onChange={(e) => setLimit(Number(e.target.value))}
                min={1}
                max={50}
              />
            </div>
          </form>
        </div>
      </div>

      {/* Results */}
      {searchMutation.data && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <h3 className="card-title">Results ({searchMutation.data.count})</h3>
            {searchMutation.data.results.length === 0 ? (
              <p className="text-base-content/60">No matching results found.</p>
            ) : (
              <div className="space-y-3">
                {searchMutation.data.results.map((result, idx) => (
                  <div key={idx} className="p-3 rounded-lg bg-base-200/50">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <span className="badge badge-sm">{result.type}</span>
                        <span className="text-xs text-base-content/50">
                          Task: {result.task_id.substring(0, 8)}
                        </span>
                      </div>
                      <span className="text-xs font-medium">
                        {Math.round(result.score * 100)}% match
                      </span>
                    </div>
                    <p className="text-sm whitespace-pre-wrap line-clamp-3">
                      {result.content}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {searchMutation.error && (
        <div className="alert alert-error">
          <AlertCircle size={16} />
          <span>{searchMutation.error instanceof Error ? searchMutation.error.message : 'Search failed'}</span>
        </div>
      )}
    </div>
  )
}

// =============================================================================
// Security Panel
// =============================================================================

interface ScanFinding {
  scanner: string
  severity: string
  message: string
  file: string
  line: number
  column?: number
  rule_id?: string
}

interface ScanResponse {
  findings: ScanFinding[]
  summary: {
    total: number
    critical: number
    high: number
    medium: number
    low: number
  }
  passed: boolean
}

export function SecurityPanel() {
  const [scanners, setScanners] = useState<string[]>(['gosec', 'gitleaks'])
  const [failLevel, setFailLevel] = useState('critical')

  const scanMutation = useMutation({
    mutationFn: () =>
      apiRequest<ScanResponse>('/security/scan', {
        method: 'POST',
        body: JSON.stringify({ scanners, fail_level: failLevel }),
      }),
  })

  const toggleScanner = (scanner: string) => {
    setScanners((prev) =>
      prev.includes(scanner) ? prev.filter((s) => s !== scanner) : [...prev, scanner]
    )
  }

  const scannerOptions = [
    { value: 'gosec', label: 'GoSec (Go SAST)' },
    { value: 'gitleaks', label: 'GitLeaks (Secrets)' },
    { value: 'govulncheck', label: 'Go Vuln Check' },
    { value: 'semgrep', label: 'Semgrep' },
    { value: 'npm-audit', label: 'NPM Audit' },
    { value: 'eslint-security', label: 'ESLint Security' },
    { value: 'bandit', label: 'Bandit (Python)' },
    { value: 'pip-audit', label: 'Pip Audit' },
  ]

  const severityColors: Record<string, string> = {
    critical: 'badge-error',
    high: 'badge-warning',
    medium: 'badge-info',
    low: 'badge-ghost',
  }

  return (
    <div className="space-y-4">
      {/* Scan configuration */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <h3 className="card-title">
            <Shield size={20} />
            Security Scan
          </h3>
          <p className="text-sm text-base-content/60">
            Run security scanners to detect vulnerabilities, secrets, and code issues.
          </p>

          <div className="space-y-4 mt-4">
            <div>
              <label className="label">
                <span className="label-text font-medium">Scanners</span>
              </label>
              <div className="flex flex-wrap gap-3">
                {scannerOptions.map((opt) => (
                  <label key={opt.value} className="label cursor-pointer gap-2">
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary"
                      checked={scanners.includes(opt.value)}
                      onChange={() => toggleScanner(opt.value)}
                    />
                    <span className="label-text">{opt.label}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="form-control w-48">
              <label className="label">
                <span className="label-text font-medium">Fail Level</span>
              </label>
              <select
                className="select select-bordered"
                value={failLevel}
                onChange={(e) => setFailLevel(e.target.value)}
              >
                <option value="critical">Critical only</option>
                <option value="high">High and above</option>
                <option value="medium">Medium and above</option>
                <option value="low">Low and above</option>
                <option value="any">Any finding</option>
              </select>
            </div>

            <button
              className="btn btn-primary"
              onClick={() => scanMutation.mutate()}
              disabled={scanners.length === 0 || scanMutation.isPending}
            >
              {scanMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Scanning...
                </>
              ) : (
                <>
                  <Play size={16} />
                  Run Scan
                </>
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Scan results */}
      {scanMutation.data && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <div className="flex items-center justify-between">
              <h3 className="card-title">Scan Results</h3>
              {scanMutation.data.passed ? (
                <span className="badge badge-success gap-1">
                  <CheckCircle size={14} />
                  Passed
                </span>
              ) : (
                <span className="badge badge-error gap-1">
                  <XCircle size={14} />
                  Issues Found
                </span>
              )}
            </div>

            {/* Summary */}
            <div className="stats stats-horizontal shadow-sm bg-base-200/50">
              <div className="stat py-2 px-4">
                <div className="stat-title text-xs">Total</div>
                <div className="stat-value text-lg">{scanMutation.data.summary.total}</div>
              </div>
              <div className="stat py-2 px-4">
                <div className="stat-title text-xs">Critical</div>
                <div className="stat-value text-lg text-error">{scanMutation.data.summary.critical}</div>
              </div>
              <div className="stat py-2 px-4">
                <div className="stat-title text-xs">High</div>
                <div className="stat-value text-lg text-warning">{scanMutation.data.summary.high}</div>
              </div>
              <div className="stat py-2 px-4">
                <div className="stat-title text-xs">Medium</div>
                <div className="stat-value text-lg text-info">{scanMutation.data.summary.medium}</div>
              </div>
              <div className="stat py-2 px-4">
                <div className="stat-title text-xs">Low</div>
                <div className="stat-value text-lg">{scanMutation.data.summary.low}</div>
              </div>
            </div>

            {/* Findings */}
            {scanMutation.data.findings.length > 0 && (
              <div className="space-y-2 mt-4">
                {scanMutation.data.findings.map((finding, idx) => (
                  <div key={idx} className="p-3 rounded-lg bg-base-200/50">
                    <div className="flex items-start gap-2">
                      <AlertTriangle
                        size={16}
                        className={
                          finding.severity === 'critical'
                            ? 'text-error'
                            : finding.severity === 'high'
                            ? 'text-warning'
                            : 'text-info'
                        }
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`badge badge-sm ${severityColors[finding.severity.toLowerCase()] || 'badge-ghost'}`}>
                            {finding.severity}
                          </span>
                          <span className="text-xs text-base-content/50">{finding.scanner}</span>
                          {finding.rule_id && (
                            <span className="text-xs text-base-content/50 font-mono">{finding.rule_id}</span>
                          )}
                        </div>
                        <p className="text-sm">{finding.message}</p>
                        <p className="text-xs text-base-content/50 mt-1 font-mono">
                          {finding.file}:{finding.line}
                          {finding.column ? `:${finding.column}` : ''}
                        </p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {scanMutation.error && (
        <div className="alert alert-error">
          <AlertCircle size={16} />
          <span>{scanMutation.error instanceof Error ? scanMutation.error.message : 'Scan failed'}</span>
        </div>
      )}
    </div>
  )
}

// =============================================================================
// Stack Panel
// =============================================================================

interface StackTask {
  id: string
  branch: string
  state: string
  pr_number?: number
  pr_url?: string
  depends_on?: string
  state_icon: string
}

interface StackSummary {
  id: string
  root_task: string
  task_count: number
  tasks: StackTask[]
  created_at: string
  updated_at: string
  has_rebase: boolean
  has_conflict: boolean
}

interface StackListResponse {
  stacks: StackSummary[]
  count: number
}

export function StackPanel() {
  const queryClient = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['stacks'],
    queryFn: () => apiRequest<StackListResponse>('/stack'),
  })

  const syncMutation = useMutation({
    mutationFn: () => apiRequest('/stack/sync', { method: 'POST' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['stacks'] }),
  })

  const rebaseMutation = useMutation({
    mutationFn: (stackId: string) =>
      apiRequest('/stack/rebase', {
        method: 'POST',
        body: JSON.stringify({ stack_id: stackId }),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['stacks'] }),
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="alert alert-error">
        <AlertCircle size={16} />
        <span>{error instanceof Error ? error.message : 'Failed to load stacks'}</span>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="card-title">
                <Layers size={20} />
                Task Stacks
              </h3>
              <p className="text-sm text-base-content/60">
                Manage dependent task branches and keep them synchronized.
              </p>
            </div>
            <div className="flex gap-2">
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => refetch()}
                disabled={isLoading}
              >
                <RefreshCw size={16} />
              </button>
              <button
                className="btn btn-primary btn-sm"
                onClick={() => syncMutation.mutate()}
                disabled={syncMutation.isPending}
              >
                {syncMutation.isPending ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  'Sync All'
                )}
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Stacks list */}
      {!data?.stacks || data.stacks.length === 0 ? (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body text-center py-12">
            <Layers size={48} className="mx-auto text-base-content/30 mb-4" />
            <p className="text-base-content/60">No task stacks found.</p>
            <p className="text-sm text-base-content/40">
              Stacks are created when tasks have dependencies on other tasks.
            </p>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          {data.stacks.map((stack) => (
            <div key={stack.id} className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <h4 className="font-medium">Stack: {stack.id.substring(0, 8)}</h4>
                    <span className="badge badge-sm">{stack.task_count} tasks</span>
                    {stack.has_conflict && (
                      <span className="badge badge-error badge-sm gap-1">
                        <AlertTriangle size={12} />
                        Conflict
                      </span>
                    )}
                    {stack.has_rebase && !stack.has_conflict && (
                      <span className="badge badge-warning badge-sm">Needs Rebase</span>
                    )}
                  </div>
                  <button
                    className="btn btn-outline btn-sm"
                    onClick={() => rebaseMutation.mutate(stack.id)}
                    disabled={rebaseMutation.isPending}
                  >
                    {rebaseMutation.isPending ? (
                      <Loader2 className="w-4 h-4 animate-spin" />
                    ) : (
                      'Rebase'
                    )}
                  </button>
                </div>

                {/* Tasks in stack */}
                <div className="space-y-2">
                  {stack.tasks.map((task, idx) => (
                    <div
                      key={task.id}
                      className="flex items-center gap-3 p-2 rounded bg-base-200/50"
                    >
                      <span className="text-base-content/40 w-6 text-center">{idx + 1}</span>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm truncate">{task.branch}</span>
                          <span className="badge badge-sm badge-ghost">{task.state}</span>
                        </div>
                        {task.pr_url && (
                          <a
                            href={task.pr_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-primary hover:underline"
                          >
                            PR #{task.pr_number}
                          </a>
                        )}
                      </div>
                      {task.depends_on && (
                        <span className="text-xs text-base-content/50">
                          → {task.depends_on.substring(0, 8)}
                        </span>
                      )}
                    </div>
                  ))}
                </div>

                <div className="text-xs text-base-content/40 mt-2">
                  Updated: {stack.updated_at}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
