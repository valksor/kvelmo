/* eslint-disable react-refresh/only-export-components */
import type { ReactElement, ReactNode } from 'react'
import { render, type RenderOptions } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'

// Create a fresh QueryClient for each test with test-optimized settings
function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  })
}

interface WrapperOptions {
  route?: string
  queryClient?: QueryClient
}

// All-providers wrapper for page tests
function AllProviders({
  children,
  route = '/',
  queryClient,
}: WrapperOptions & { children: ReactNode }) {
  const client = queryClient ?? createTestQueryClient()

  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

// Custom render with all providers
function customRender(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'> & WrapperOptions
) {
  const { route, queryClient, ...renderOptions } = options ?? {}

  return render(ui, {
    wrapper: ({ children }) => (
      <AllProviders route={route} queryClient={queryClient}>
        {children}
      </AllProviders>
    ),
    ...renderOptions,
  })
}

// Re-export everything from testing-library
export * from '@testing-library/react'
export { customRender as render }
export { createTestQueryClient }
