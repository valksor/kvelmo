import { useState } from 'react'
import { Brain, Globe, Layers, Shield } from 'lucide-react'
import {
  BrowserPanel,
  MemoryPanel,
  SecurityPanel,
  StackPanel,
} from '@/components/tools/ToolPanels'

type TabType = 'browser' | 'memory' | 'security' | 'stack'

export default function Tools() {
  const [activeTab, setActiveTab] = useState<TabType>('browser')

  const tabs: { id: TabType; label: string; icon: React.ReactNode }[] = [
    { id: 'browser', label: 'Browser', icon: <Globe size={16} /> },
    { id: 'memory', label: 'Memory', icon: <Brain size={16} /> },
    { id: 'security', label: 'Security', icon: <Shield size={16} /> },
    { id: 'stack', label: 'Stack', icon: <Layers size={16} /> },
  ]

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Tools</h1>

      <div role="tablist" className="tabs tabs-boxed">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            role="tab"
            className={`tab gap-2 ${activeTab === tab.id ? 'tab-active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.icon}
            {tab.label}
          </button>
        ))}
      </div>

      <div className="mt-4">
        {activeTab === 'browser' && <BrowserPanel />}
        {activeTab === 'memory' && <MemoryPanel />}
        {activeTab === 'security' && <SecurityPanel />}
        {activeTab === 'stack' && <StackPanel />}
      </div>
    </div>
  )
}
