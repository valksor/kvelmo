import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Layout from '@/components/layout/Layout'
import Dashboard from '@/pages/Dashboard'
import Chat from '@/pages/Chat'
import History from '@/pages/History'
import Settings from '@/pages/Settings'
import Tools from '@/pages/Tools'
import TaskDetail from '@/pages/TaskDetail'
import Login from '@/pages/Login'
import Project from '@/pages/Project'
import Find from '@/pages/Find'
import Library from '@/pages/Library'
import Commit from '@/pages/Commit'
import Links from '@/pages/Links'
import License from '@/pages/License'
import Review from '@/pages/Review'
import Simplify from '@/pages/Simplify'
import Quick from '@/pages/Quick'
import Automation from '@/pages/Automation'

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
          {/* Public routes */}
          <Route path="/login" element={<Login />} />

          {/* Protected routes with layout */}
          <Route element={<Layout />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/task/:id" element={<TaskDetail />} />
            <Route path="/chat" element={<Chat />} />
            <Route path="/history" element={<History />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/tools" element={<Tools />} />
            <Route path="/project" element={<Project />} />
            <Route path="/find" element={<Find />} />
            <Route path="/library" element={<Library />} />
            <Route path="/commit" element={<Commit />} />
            <Route path="/links" element={<Links />} />
            <Route path="/license" element={<License />} />
            <Route path="/review" element={<Review />} />
            <Route path="/simplify" element={<Simplify />} />
            <Route path="/quick" element={<Quick />} />
            <Route path="/automation" element={<Automation />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
