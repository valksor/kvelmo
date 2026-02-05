import { useWorkflowDiagram } from '@/api/task'
import { Loader2, GitBranch } from 'lucide-react'
import type { WorkflowState } from '@/types/api'

interface WorkflowDiagramProps {
  currentState?: WorkflowState
}

export function WorkflowDiagram({ currentState }: WorkflowDiagramProps) {
  const { data: svgContent, isLoading, error } = useWorkflowDiagram()

  if (isLoading) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body py-4">
          <div className="flex items-center gap-2 text-sm font-medium text-base-content/80">
            <GitBranch size={16} />
            Workflow
          </div>
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-6 h-6 animate-spin text-primary" />
          </div>
        </div>
      </div>
    )
  }

  if (error || !svgContent) {
    return (
      <div className="card bg-base-100 shadow-sm">
        <div className="card-body py-4">
          <div className="flex items-center gap-2 text-sm font-medium text-base-content/80">
            <GitBranch size={16} />
            Workflow
          </div>
          <p className="text-sm text-base-content/50 text-center py-4">
            Unable to load workflow diagram
          </p>
        </div>
      </div>
    )
  }

  // Process SVG to highlight current state
  const processedSvg = currentState ? highlightCurrentState(svgContent, currentState) : svgContent

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm font-medium text-base-content/80">
            <GitBranch size={16} />
            Workflow
          </div>
          {currentState && <span className="badge badge-sm badge-primary">{currentState}</span>}
        </div>
        {/* SVG from internal Go backend - trusted content, not user input */}
        <div
          className="workflow-diagram mt-2 overflow-x-auto [&_svg]:max-w-full [&_svg]:h-auto"
          dangerouslySetInnerHTML={{ __html: processedSvg }}
        />
      </div>
    </div>
  )
}

/**
 * Highlight the current state in the SVG diagram.
 * Adds CSS styling to nodes matching the current state name.
 */
function highlightCurrentState(svg: string, state: WorkflowState): string {
  // Add CSS styles for highlighting current state
  const styles = `
    <style>
      .workflow-current { fill: oklch(var(--p)) !important; }
      .workflow-current text { fill: oklch(var(--pc)) !important; font-weight: bold; }
      .workflow-current rect, .workflow-current ellipse, .workflow-current circle {
        fill: oklch(var(--p)) !important;
        stroke: oklch(var(--pf)) !important;
        stroke-width: 2px;
      }
    </style>
  `

  // Insert styles after opening svg tag
  let result = svg.replace(/<svg([^>]*)>/, `<svg$1>${styles}`)

  // Try to highlight nodes containing the state name
  // Pattern: <g id="node_planning"> or similar
  const stateRegex = new RegExp(`(<g[^>]*id=["'][^"']*${state}[^"']*["'])([^>]*)>`, 'gi')
  result = result.replace(stateRegex, '$1$2 class="workflow-current">')

  return result
}
