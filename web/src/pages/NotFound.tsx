import { Link, useLocation } from 'react-router-dom'
import { FileQuestion, Home } from 'lucide-react'

export default function NotFound() {
  const location = useLocation()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Page Not Found</h1>
      </div>

      <div className="card bg-base-100 shadow-sm">
        <div className="card-body items-center text-center py-16">
          <FileQuestion aria-hidden="true" className="w-16 h-16 text-base-content/30 mb-4" />
          <h2 className="text-6xl font-bold text-primary mb-2">404</h2>
          <h3 className="text-xl font-medium mb-2">Page Not Found</h3>
          <p className="text-base-content/60 max-w-md mb-2">
            The page you're looking for doesn't exist or has been moved.
          </p>
          <p className="text-sm text-base-content/40 font-mono mb-6">
            {location.pathname}
          </p>
          <Link to="/" className="btn btn-primary gap-2">
            <Home size={18} aria-hidden="true" />
            Back to Dashboard
          </Link>
        </div>
      </div>
    </div>
  )
}
