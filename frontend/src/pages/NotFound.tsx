// frontend/src/pages/NotFound.tsx
import { Link } from 'react-router-dom'

export function NotFound() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-slate-600">404</h1>
        <p className="text-slate-400 mt-4">Page not found</p>
        <Link to="/" className="mt-6 inline-block px-4 py-2 bg-blue-600 rounded text-white hover:bg-blue-700">
          Back to Dashboard
        </Link>
      </div>
    </div>
  )
}
