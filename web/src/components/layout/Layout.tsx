import { Outlet, NavLink, useLocation } from 'react-router-dom'
import {
  Home,
  MessageSquare,
  History,
  Settings,
  Wrench,
  FolderKanban,
  Search,
  BookOpen,
  GitCommit,
  Link2,
  Scale,
  Eye,
  Sparkles,
  Zap,
  ArrowLeft,
  Layers,
  HelpCircle,
} from 'lucide-react'
import { useRef, useEffect, type ComponentType } from 'react'
import { useStatus } from '@/api/workflow'
import { useSwitchToGlobal } from '@/api/projects'
import { useDocsURL } from '@/api/settings'
import { ThemeToggle } from '@/components/ui/ThemeToggle'
import { NotificationCenter } from '@/components/ui/NotificationCenter'
import { SkipLink } from '@/components/ui/SkipLink'

// Type definitions for navigation structure
type NavItem = {
  to: string
  icon: ComponentType<{ size: number }>
  label: string
}

type NavDropdown = {
  label: string
  icon: ComponentType<{ size: number }>
  items: NavItem[]
}

type NavEntry = NavItem | NavDropdown

function isDropdown(entry: NavEntry): entry is NavDropdown {
  return 'items' in entry
}

// Dropdown component with active state detection
function NavDropdownMenu({
  dropdown,
  isLast = false,
}: {
  dropdown: NavDropdown
  isLast?: boolean
}) {
  const location = useLocation()
  const isChildActive = dropdown.items.some(
    (item) => location.pathname === item.to
  )
  const Icon = dropdown.icon

  return (
    <li>
      <details
        onToggle={(e) => {
          if ((e.target as HTMLDetailsElement).open) {
            // Close other open details in the same menu (accordion behavior)
            const menu = (e.target as HTMLElement).closest('ul.menu')
            menu?.querySelectorAll('details[open]').forEach((el) => {
              if (el !== e.target) el.removeAttribute('open')
            })
          }
        }}
      >
        <summary className={isChildActive ? 'menu-active' : ''}>
          <Icon size={18} aria-hidden="true" />
          <span className="hidden sm:inline">{dropdown.label}</span>
        </summary>
        <ul
          className={`p-2 bg-base-100 rounded-box shadow-lg border border-base-300 z-[50] ${isLast ? 'right-0' : ''}`}
        >
          {dropdown.items.map(({ to, icon: ItemIcon, label }) => (
            <li key={to}>
              <NavLink
                to={to}
                onClick={(e) => {
                  (e.target as HTMLElement).closest('details')?.removeAttribute('open')
                }}
                className={({ isActive }) => (isActive ? 'menu-active' : '')}
              >
                <ItemIcon size={16} aria-hidden="true" />
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
      </details>
    </li>
  )
}

// Admin dropdown - shared between global and project modes
const adminDropdown: NavDropdown = {
  label: 'Admin',
  icon: Settings,
  items: [
    { to: '/settings', icon: Settings, label: 'Settings' },
    { to: '/license', icon: Scale, label: 'License' },
  ],
}

// Global mode navigation (simplified)
const globalNavItems: NavEntry[] = [
  { to: '/', icon: Home, label: 'Dashboard' },
  adminDropdown,
]

const workDropdown: NavDropdown = {
  label: 'Work',
  icon: Layers,
  items: [
    { to: '/project', icon: FolderKanban, label: 'Project' },
    { to: '/quick', icon: Zap, label: 'Quick' },
    { to: '/history', icon: History, label: 'History' },
  ],
}

const advancedDropdown: NavDropdown = {
  label: 'Advanced',
  icon: Wrench,
  items: [
    { to: '/find', icon: Search, label: 'Find' },
    { to: '/review', icon: Eye, label: 'Review' },
    { to: '/commit', icon: GitCommit, label: 'Commit' },
    { to: '/simplify', icon: Sparkles, label: 'Simplify' },
    { to: '/chat', icon: MessageSquare, label: 'Chat' },
    { to: '/library', icon: BookOpen, label: 'Library' },
    { to: '/links', icon: Link2, label: 'Links' },
    { to: '/tools', icon: Wrench, label: 'Tools' },
  ],
}

// Project mode navigation (work-first, advanced hidden under dropdown)
const projectNavItems: NavEntry[] = [
  { to: '/', icon: Home, label: 'Dashboard' },
  workDropdown,
  advancedDropdown,
  // Admin dropdown
  adminDropdown,
]

export default function Layout() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const { data: docsData } = useDocsURL()
  // Default to global mode while loading (safer - fewer nav items, no project-specific routes)
  const isGlobalMode = statusLoading || status?.mode === 'global'
  const activeProject = isGlobalMode ? undefined : status?.project
  // Only show "Projects" button if server started in global mode (has project list to return to)
  const canSwitchToGlobal = status?.canSwitchToGlobal ?? false
  const switchToGlobal = useSwitchToGlobal()
  const navRef = useRef<HTMLUListElement>(null)

  // Close all dropdowns when clicking outside navigation
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (navRef.current && !navRef.current.contains(event.target as Node)) {
        navRef.current.querySelectorAll('details[open]').forEach((el) => {
          el.removeAttribute('open')
        })
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // In global mode, only show global items
  // In project mode, show project items with dropdowns
  const navItems = isGlobalMode ? globalNavItems : projectNavItems

  return (
    <div className="min-h-screen bg-base-200">
      <SkipLink />
      {/* Navbar */}
      <nav className="navbar bg-base-100 shadow-sm border-b border-base-300" aria-label="Main navigation">
        <div className="flex-1 flex items-center gap-2">
          {canSwitchToGlobal && !isGlobalMode && (
            <button
              onClick={() => switchToGlobal.mutate()}
              disabled={switchToGlobal.isPending}
              className="btn btn-ghost btn-sm gap-1"
              aria-label="Back to project list"
            >
              <ArrowLeft size={16} aria-hidden="true" />
              <span className="hidden sm:inline">Projects</span>
            </button>
          )}
          <a href="/" className="btn btn-ghost text-xl">
            Mehrhof
          </a>
          {activeProject && (
            <div className="flex items-center gap-2 rounded-lg border border-base-300 bg-base-200 px-3 py-1 max-w-[55vw] min-w-0">
              <FolderKanban size={14} className="text-primary shrink-0" aria-hidden="true" />
              <span className="text-sm font-medium truncate">{activeProject.name}</span>
              <span className="text-base-content/40">|</span>
              <span
                className="text-xs font-mono text-base-content/60 truncate"
                title={activeProject.remote_url || activeProject.path}
              >
                {activeProject.remote_url || activeProject.path}
              </span>
            </div>
          )}
        </div>
        <div className="flex-none flex items-center gap-2">
          <ul ref={navRef} className="menu menu-horizontal px-1">
            {navItems.map((entry, index) => {
              const isLast = index === navItems.length - 1

              if (isDropdown(entry)) {
                return (
                  <NavDropdownMenu
                    key={entry.label}
                    dropdown={entry}
                    isLast={isLast}
                  />
                )
              }

              const { to, icon: Icon, label } = entry
              return (
                <li key={to}>
                  <NavLink
                    to={to}
                    className={({ isActive }) => (isActive ? 'menu-active' : '')}
                    aria-label={label}
                  >
                    <Icon size={18} aria-hidden="true" />
                    <span className="hidden sm:inline">{label}</span>
                  </NavLink>
                </li>
              )
            })}
          </ul>
          <NotificationCenter />
          {docsData?.url && (
            <a
              href={docsData.url}
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-ghost btn-sm btn-circle"
              title="Documentation"
              aria-label="Open documentation"
            >
              <HelpCircle size={18} aria-hidden="true" />
            </a>
          )}
          <ThemeToggle />
        </div>
      </nav>

      {/* Main content */}
      <main id="main-content" className="container mx-auto p-4 max-w-7xl" tabIndex={-1}>
        <Outlet />
      </main>
    </div>
  )
}
