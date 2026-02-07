import { useState } from 'react'
import { Brain, Globe, Layers, Shield } from 'lucide-react'
import {
  BrowserPanel,
  MemoryPanel,
  SecurityPanel,
  StackPanel,
} from '@/components/tools/ToolPanels'

type TabType = 'browser' | 'memory' | 'security' | 'stack'
const TAB_ORDER: TabType[] = ['browser', 'memory', 'security', 'stack']

export default function Tools() {
  const [activeTab, setActiveTab] = useState<TabType>('browser')

  const tabs: { id: TabType; label: string; icon: React.ReactNode }[] = [
    { id: 'browser', label: 'Browser', icon: <Globe size={16} aria-hidden="true" /> },
    { id: 'memory', label: 'Memory', icon: <Brain size={16} aria-hidden="true" /> },
    { id: 'security', label: 'Security', icon: <Shield size={16} aria-hidden="true" /> },
    { id: 'stack', label: 'Stack', icon: <Layers size={16} aria-hidden="true" /> },
  ]

  const handleTabKeyDown = (e: React.KeyboardEvent<HTMLButtonElement>, current: TabType) => {
    const currentIndex = TAB_ORDER.indexOf(current)
    if (currentIndex < 0) return

    if (e.key === 'ArrowRight') {
      e.preventDefault()
      setActiveTab(TAB_ORDER[(currentIndex + 1) % TAB_ORDER.length])
      return
    }
    if (e.key === 'ArrowLeft') {
      e.preventDefault()
      setActiveTab(TAB_ORDER[(currentIndex - 1 + TAB_ORDER.length) % TAB_ORDER.length])
      return
    }
    if (e.key === 'Home') {
      e.preventDefault()
      setActiveTab(TAB_ORDER[0])
      return
    }
    if (e.key === 'End') {
      e.preventDefault()
      setActiveTab(TAB_ORDER[TAB_ORDER.length - 1])
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Tools</h1>

      <div role="tablist" aria-label="Tools panels" className="tabs tabs-boxed">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            role="tab"
            id={`tools-tab-${tab.id}`}
            aria-selected={activeTab === tab.id}
            aria-controls={`tools-panel-${tab.id}`}
            tabIndex={activeTab === tab.id ? 0 : -1}
            className={`tab gap-2 ${activeTab === tab.id ? 'tab-active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
            onKeyDown={(e) => handleTabKeyDown(e, tab.id)}
          >
            {tab.icon}
            {tab.label}
          </button>
        ))}
      </div>

      {tabs.map((tab) => (
        <div
          key={tab.id}
          id={`tools-panel-${tab.id}`}
          role="tabpanel"
          aria-labelledby={`tools-tab-${tab.id}`}
          hidden={activeTab !== tab.id}
          className="mt-4"
        >
          {activeTab === tab.id && tab.id === 'browser' && <BrowserPanel />}
          {activeTab === tab.id && tab.id === 'memory' && <MemoryPanel />}
          {activeTab === tab.id && tab.id === 'security' && <SecurityPanel />}
          {activeTab === tab.id && tab.id === 'stack' && <StackPanel />}
        </div>
      ))}
    </div>
  )
}
