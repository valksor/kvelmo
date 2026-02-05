import type { WorkflowState } from '../types/api'

/**
 * Configuration for workflow state display.
 * Consolidates state styling used across multiple components.
 */
export interface StateConfig {
  /** Emoji icon for the state */
  icon: string
  /** DaisyUI badge class */
  badge: string
  /** Background color class (solid) */
  color: string
  /** Background color class (translucent, for cards) */
  bgClass: string
}

/**
 * Unified state configuration for all workflow states.
 * Used by TaskCard, RecentTasksCard, TaskSummaryCard, ActiveWorkCard, History, etc.
 */
export const stateConfig: Record<WorkflowState, StateConfig> = {
  idle: {
    icon: '⏸️',
    badge: 'badge-ghost',
    color: 'bg-base-300',
    bgClass: 'bg-base-200',
  },
  planning: {
    icon: '📝',
    badge: 'badge-info',
    color: 'bg-info',
    bgClass: 'bg-info/10',
  },
  implementing: {
    icon: '🔨',
    badge: 'badge-primary',
    color: 'bg-primary',
    bgClass: 'bg-primary/10',
  },
  reviewing: {
    icon: '🔍',
    badge: 'badge-secondary',
    color: 'bg-secondary',
    bgClass: 'bg-secondary/10',
  },
  waiting: {
    icon: '⏳',
    badge: 'badge-warning',
    color: 'bg-warning',
    bgClass: 'bg-warning/10',
  },
  checkpointing: {
    icon: '💾',
    badge: 'badge-info',
    color: 'bg-info',
    bgClass: 'bg-info/10',
  },
  reverting: {
    icon: '↩️',
    badge: 'badge-warning',
    color: 'bg-warning',
    bgClass: 'bg-warning/10',
  },
  restoring: {
    icon: '↪️',
    badge: 'badge-warning',
    color: 'bg-warning',
    bgClass: 'bg-warning/10',
  },
  done: {
    icon: '✅',
    badge: 'badge-success',
    color: 'bg-success',
    bgClass: 'bg-success/10',
  },
  failed: {
    icon: '❌',
    badge: 'badge-error',
    color: 'bg-error',
    bgClass: 'bg-error/10',
  },
}

/**
 * Helper to get state config with fallback for unknown states.
 */
export function getStateConfig(state: string): StateConfig {
  return stateConfig[state as WorkflowState] ?? stateConfig.idle
}
