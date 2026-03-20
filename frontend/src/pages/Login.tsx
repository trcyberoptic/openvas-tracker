// frontend/src/pages/Login.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import { api } from '@/api/client'

interface LoginResponse {
  token: string
  user: {
    id: string
    email: string
    username: string
    role: string
  }
}

export function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isRegister, setIsRegister] = useState(false)
  const [username, setUsername] = useState('')
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const endpoint = isRegister ? '/auth/register' : '/auth/login'
      const body = isRegister ? { email, username, password } : { email, password }
      const res = await api.post<LoginResponse>(endpoint, body)
      login(res.token, res.user)
      navigate('/')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Authentication failed')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-950">
      <div className="w-full max-w-md bg-slate-900 rounded-lg border border-slate-800 p-8">
        <h1 className="text-2xl font-bold text-white mb-6">VulnTrack Pro</h1>
        {error && <div className="bg-red-900/50 text-red-300 p-3 rounded mb-4">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <input type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)}
            className="w-full p-3 bg-slate-800 border border-slate-700 rounded text-white" required />
          {isRegister && (
            <input type="text" placeholder="Username" value={username} onChange={e => setUsername(e.target.value)}
              className="w-full p-3 bg-slate-800 border border-slate-700 rounded text-white" required />
          )}
          <input type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)}
            className="w-full p-3 bg-slate-800 border border-slate-700 rounded text-white" required />
          <button type="submit" className="w-full p-3 bg-blue-600 hover:bg-blue-700 rounded text-white font-medium">
            {isRegister ? 'Register' : 'Sign In'}
          </button>
        </form>
        <button onClick={() => setIsRegister(!isRegister)} className="mt-4 text-sm text-slate-400 hover:text-white">
          {isRegister ? 'Already have an account? Sign in' : 'Need an account? Register'}
        </button>
      </div>
    </div>
  )
}
