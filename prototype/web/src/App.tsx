import { lazy, Suspense, type ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Layout from '@/components/layout/Layout'

// Eagerly loaded (critical path)
import Dashboard from '@/pages/Dashboard'
// DISABLED: remote serve temporarily unavailable
// import Login from '@/pages/Login'
import NotFound from '@/pages/NotFound'

// Lazy loaded pages - largest first for maximum impact
const Tools = lazy(() => import('@/pages/Tools'))
const Settings = lazy(() => import('@/pages/Settings'))
const Quick = lazy(() => import('@/pages/Quick'))
// DISABLED: automation temporarily unavailable (requires remote serve)
// const Automation = lazy(() => import('@/pages/Automation'))
const Project = lazy(() => import('@/pages/Project'))
const Chat = lazy(() => import('@/pages/Chat'))
const History = lazy(() => import('@/pages/History'))
const Review = lazy(() => import('@/pages/Review'))
const Simplify = lazy(() => import('@/pages/Simplify'))
const Library = lazy(() => import('@/pages/Library'))
const Find = lazy(() => import('@/pages/Find'))
const Commit = lazy(() => import('@/pages/Commit'))
const Links = lazy(() => import('@/pages/Links'))
const License = lazy(() => import('@/pages/License'))
const TaskDetail = lazy(() => import('@/pages/TaskDetail'))

function PageLoader() {
  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <span className="loading loading-spinner loading-lg text-primary" />
    </div>
  )
}

function LazyRoute({ children }: { children: ReactNode }) {
  return <Suspense fallback={<PageLoader />}>{children}</Suspense>
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5000,
      refetchOnWindowFocus: false,
    },
  },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* DISABLED: remote serve temporarily unavailable */}
          {/* <Route path="/login" element={<Login />} /> */}

          {/* Protected routes with layout */}
          <Route element={<Layout />}>
            {/* Eagerly loaded */}
            <Route path="/" element={<Dashboard />} />

            {/* Lazy loaded */}
            <Route path="/task/:id" element={<LazyRoute><TaskDetail /></LazyRoute>} />
            <Route path="/chat" element={<LazyRoute><Chat /></LazyRoute>} />
            <Route path="/history" element={<LazyRoute><History /></LazyRoute>} />
            <Route path="/settings" element={<LazyRoute><Settings /></LazyRoute>} />
            <Route path="/tools" element={<LazyRoute><Tools /></LazyRoute>} />
            <Route path="/project" element={<LazyRoute><Project /></LazyRoute>} />
            <Route path="/find" element={<LazyRoute><Find /></LazyRoute>} />
            <Route path="/library" element={<LazyRoute><Library /></LazyRoute>} />
            <Route path="/commit" element={<LazyRoute><Commit /></LazyRoute>} />
            <Route path="/links" element={<LazyRoute><Links /></LazyRoute>} />
            <Route path="/license" element={<LazyRoute><License /></LazyRoute>} />
            <Route path="/review" element={<LazyRoute><Review /></LazyRoute>} />
            <Route path="/simplify" element={<LazyRoute><Simplify /></LazyRoute>} />
            <Route path="/quick" element={<LazyRoute><Quick /></LazyRoute>} />
            {/* DISABLED: automation temporarily unavailable (requires remote serve) */}
            {/* <Route path="/automation" element={<LazyRoute><Automation /></LazyRoute>} /> */}

            {/* 404 - eagerly loaded */}
            <Route path="*" element={<NotFound />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
