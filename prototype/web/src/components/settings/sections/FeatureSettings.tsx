import {
  TextInput,
  NumberInput,
  Checkbox,
  Select,
  CollapseSection,
} from '@/components/settings/FormField'
import type { SettingsSectionProps } from './types'

export function FeatureSettings({ data, updateField }: SettingsSectionProps) {
  return (
    <div className="space-y-4">
      {/* Browser */}
      <CollapseSection title="Browser Automation">
        <Checkbox
          label="Enable Browser"
          hint="Allow AI agents to control a browser"
          checked={data.browser?.enabled}
          onChange={(v) => updateField(['browser', 'enabled'], v)}
        />
        {data.browser?.enabled && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
            <NumberInput
              label="Port"
              hint="0 = random, 9222 = existing Chrome"
              value={data.browser?.port}
              onChange={(v) => updateField(['browser', 'port'], v)}
              min={0}
            />
            <NumberInput
              label="Timeout (seconds)"
              value={data.browser?.timeout}
              onChange={(v) => updateField(['browser', 'timeout'], v)}
              min={5}
              max={300}
            />
            <TextInput
              label="Screenshot Directory"
              value={data.browser?.screenshot_dir}
              onChange={(v) => updateField(['browser', 'screenshot_dir'], v)}
            />
            <Checkbox
              label="Headless"
              hint="Run browser without UI"
              checked={data.browser?.headless}
              onChange={(v) => updateField(['browser', 'headless'], v)}
            />
            <Checkbox
              label="Auto-load Cookies"
              checked={data.browser?.cookie_auto_load}
              onChange={(v) => updateField(['browser', 'cookie_auto_load'], v)}
            />
            <Checkbox
              label="Auto-save Cookies"
              checked={data.browser?.cookie_auto_save}
              onChange={(v) => updateField(['browser', 'cookie_auto_save'], v)}
            />
          </div>
        )}
      </CollapseSection>

      {/* MCP */}
      <CollapseSection title="MCP (Model Context Protocol)">
        <Checkbox
          label="Enable MCP Server"
          hint="Allow AI agents to call Mehrhof commands via MCP"
          checked={data.mcp?.enabled}
          onChange={(v) => updateField(['mcp', 'enabled'], v)}
        />
        {data.mcp?.enabled && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
            <NumberInput
              label="Rate Limit (req/sec)"
              value={data.mcp?.rate_limit?.rate}
              onChange={(v) => updateField(['mcp', 'rate_limit', 'rate'], v)}
              min={1}
              max={100}
            />
            <NumberInput
              label="Burst Size"
              value={data.mcp?.rate_limit?.burst}
              onChange={(v) => updateField(['mcp', 'rate_limit', 'burst'], v)}
              min={1}
              max={200}
            />
          </div>
        )}
      </CollapseSection>

      {/* Security */}
      <CollapseSection title="Security Scanning">
        <Checkbox
          label="Enable Security Scanning"
          hint="Scan code for vulnerabilities and secrets"
          checked={data.security?.enabled}
          onChange={(v) => updateField(['security', 'enabled'], v)}
        />
        {data.security?.enabled && (
          <>
            <h4 className="font-medium text-sm mt-4 mb-2">Run On</h4>
            <div className="grid grid-cols-3 gap-4">
              <Checkbox
                label="Planning"
                checked={data.security?.run_on?.planning}
                onChange={(v) => updateField(['security', 'run_on', 'planning'], v)}
              />
              <Checkbox
                label="Implementing"
                checked={data.security?.run_on?.implementing}
                onChange={(v) => updateField(['security', 'run_on', 'implementing'], v)}
              />
              <Checkbox
                label="Reviewing"
                checked={data.security?.run_on?.reviewing}
                onChange={(v) => updateField(['security', 'run_on', 'reviewing'], v)}
              />
            </div>
            <h4 className="font-medium text-sm mt-4 mb-2">Fail On</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Severity Level"
                value={data.security?.fail_on?.level}
                onChange={(v) => updateField(['security', 'fail_on', 'level'], v)}
                options={[
                  { value: 'critical', label: 'Critical' },
                  { value: 'high', label: 'High' },
                  { value: 'medium', label: 'Medium' },
                  { value: 'low', label: 'Low' },
                  { value: 'any', label: 'Any' },
                ]}
              />
              <Checkbox
                label="Block Finish"
                hint="Block task completion on failures"
                checked={data.security?.fail_on?.block_finish}
                onChange={(v) => updateField(['security', 'fail_on', 'block_finish'], v)}
              />
            </div>
            <h4 className="font-medium text-sm mt-4 mb-2">Scanners</h4>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <Checkbox
                label="SAST"
                checked={data.security?.scanners?.sast?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'sast', 'enabled'], v)}
              />
              <Checkbox
                label="Secrets"
                checked={data.security?.scanners?.secrets?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'secrets', 'enabled'], v)}
              />
              <Checkbox
                label="Dependencies"
                checked={data.security?.scanners?.dependencies?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'dependencies', 'enabled'], v)}
              />
              <Checkbox
                label="License"
                checked={data.security?.scanners?.license?.enabled}
                onChange={(v) => updateField(['security', 'scanners', 'license', 'enabled'], v)}
              />
            </div>
          </>
        )}
      </CollapseSection>

      {/* Memory */}
      <CollapseSection title="Memory System">
        <Checkbox
          label="Enable Memory"
          hint="Semantic search and learning from past tasks"
          checked={data.memory?.enabled}
          onChange={(v) => updateField(['memory', 'enabled'], v)}
        />
        {data.memory?.enabled && (
          <>
            <h4 className="font-medium text-sm mt-4 mb-2">Vector Database</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Backend"
                value={data.memory?.vector_db?.backend}
                onChange={(v) => updateField(['memory', 'vector_db', 'backend'], v)}
                options={[
                  { value: 'chromadb', label: 'ChromaDB' },
                  { value: 'pinecone', label: 'Pinecone' },
                  { value: 'weaviate', label: 'Weaviate' },
                  { value: 'qdrant', label: 'Qdrant' },
                ]}
              />
              <TextInput
                label="Connection String"
                value={data.memory?.vector_db?.connection_string}
                onChange={(v) => updateField(['memory', 'vector_db', 'connection_string'], v)}
              />
            </div>
            <h4 className="font-medium text-sm mt-4 mb-2">Embedding Model</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Select
                label="Model Type"
                hint="ONNX provides true semantic similarity"
                value={data.memory?.vector_db?.embedding_model || 'default'}
                onChange={(v) => updateField(['memory', 'vector_db', 'embedding_model'], v)}
                options={[
                  { value: 'default', label: 'Hash-based (default)' },
                  { value: 'onnx', label: 'ONNX Neural (semantic)' },
                ]}
              />
              {data.memory?.vector_db?.embedding_model === 'onnx' && (
                <Select
                  label="ONNX Model"
                  hint="Downloaded on first use"
                  value={data.memory?.vector_db?.onnx?.model || 'all-MiniLM-L6-v2'}
                  onChange={(v) => updateField(['memory', 'vector_db', 'onnx', 'model'], v)}
                  options={[
                    { value: 'all-MiniLM-L6-v2', label: 'all-MiniLM-L6-v2 (22MB, fast)' },
                    { value: 'all-MiniLM-L12-v2', label: 'all-MiniLM-L12-v2 (33MB, better)' },
                  ]}
                />
              )}
            </div>
            {data.memory?.vector_db?.embedding_model === 'onnx' && (
              <p className="text-xs text-muted-foreground mt-2">
                Switching embedding models invalidates existing vectors. Run <code className="bg-muted px-1 rounded">mehr memory clear</code> after changing.
              </p>
            )}
            <h4 className="font-medium text-sm mt-4 mb-2">Search</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <NumberInput
                label="Max Results"
                value={data.memory?.search?.max_results}
                onChange={(v) => updateField(['memory', 'search', 'max_results'], v)}
                min={1}
                max={50}
              />
              <NumberInput
                label="Similarity Threshold"
                hint="0-1"
                value={data.memory?.search?.similarity_threshold}
                onChange={(v) => updateField(['memory', 'search', 'similarity_threshold'], v)}
                min={0}
                max={1}
                step={0.1}
              />
            </div>
          </>
        )}
      </CollapseSection>

      {/* Sandbox */}
      <CollapseSection title="Sandbox">
        <Checkbox
          label="Enable Sandbox"
          hint="Isolate agent execution for security"
          checked={data.sandbox?.enabled}
          onChange={(v) => updateField(['sandbox', 'enabled'], v)}
        />
        {data.sandbox?.enabled && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
            <Checkbox
              label="Allow Network"
              hint="Required for LLM APIs"
              checked={data.sandbox?.network}
              onChange={(v) => updateField(['sandbox', 'network'], v)}
            />
            <TextInput
              label="Tmp Directory"
              value={data.sandbox?.tmp_dir}
              onChange={(v) => updateField(['sandbox', 'tmp_dir'], v)}
            />
          </div>
        )}
      </CollapseSection>

      {/* Quality */}
      <CollapseSection title="Quality & Linters">
        <Checkbox
          label="Enable Quality Checks"
          checked={data.quality?.enabled}
          onChange={(v) => updateField(['quality', 'enabled'], v)}
        />
        <Checkbox
          label="Use Defaults"
          hint="Auto-enable default linters for detected languages"
          checked={data.quality?.use_defaults}
          onChange={(v) => updateField(['quality', 'use_defaults'], v)}
        />
      </CollapseSection>

      {/* Links */}
      <CollapseSection title="Links (Bidirectional Linking)">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Checkbox
            label="Enabled"
            checked={data.links?.enabled}
            onChange={(v) => updateField(['links', 'enabled'], v)}
          />
          <Checkbox
            label="Auto Index"
            checked={data.links?.auto_index}
            onChange={(v) => updateField(['links', 'auto_index'], v)}
          />
          <Checkbox
            label="Case Sensitive"
            checked={data.links?.case_sensitive}
            onChange={(v) => updateField(['links', 'case_sensitive'], v)}
          />
          <NumberInput
            label="Max Context Length"
            value={data.links?.max_context_length}
            onChange={(v) => updateField(['links', 'max_context_length'], v)}
            min={50}
            max={500}
          />
        </div>
      </CollapseSection>

      {/* Context */}
      <CollapseSection title="Hierarchical Context">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <Checkbox
            label="Include Parent"
            checked={data.context?.include_parent}
            onChange={(v) => updateField(['context', 'include_parent'], v)}
          />
          <Checkbox
            label="Include Siblings"
            checked={data.context?.include_siblings}
            onChange={(v) => updateField(['context', 'include_siblings'], v)}
          />
          <NumberInput
            label="Max Siblings"
            value={data.context?.max_siblings}
            onChange={(v) => updateField(['context', 'max_siblings'], v)}
            min={1}
            max={20}
          />
          <NumberInput
            label="Description Limit"
            value={data.context?.description_limit}
            onChange={(v) => updateField(['context', 'description_limit'], v)}
            min={100}
            max={2000}
          />
        </div>
      </CollapseSection>

      {/* Labels */}
      <CollapseSection title="Labels">
        <Checkbox
          label="Enable Labels"
          checked={data.labels?.enabled}
          onChange={(v) => updateField(['labels', 'enabled'], v)}
        />
      </CollapseSection>

      {/* Library */}
      <CollapseSection title="Library (Documentation)">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <NumberInput
            label="Auto Include Max"
            hint="Max collections to auto-include"
            value={data.library?.auto_include_max}
            onChange={(v) => updateField(['library', 'auto_include_max'], v)}
            min={0}
            max={10}
          />
          <NumberInput
            label="Max Pages Per Prompt"
            value={data.library?.max_pages_per_prompt}
            onChange={(v) => updateField(['library', 'max_pages_per_prompt'], v)}
            min={1}
            max={100}
          />
          <NumberInput
            label="Max Token Budget"
            value={data.library?.max_token_budget}
            onChange={(v) => updateField(['library', 'max_token_budget'], v)}
            min={1000}
            max={50000}
          />
        </div>
      </CollapseSection>
    </div>
  )
}

// =============================================================================
// Automation Settings Tab
// =============================================================================
