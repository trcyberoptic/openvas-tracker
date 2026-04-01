// frontend/src/components/layout/Shell.tsx
import { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Sidebar } from './Sidebar'
import { useAuth } from '@/hooks/useAuth'
import { api } from '@/api/client'

export function Shell() {
  const { user, logout } = useAuth()
  const { data: setup } = useQuery({
    queryKey: ['setup'],
    queryFn: () => api.get<{ bugreport_url?: string }>('/settings/setup'),
    staleTime: Infinity,
  })

  useEffect(() => {
    const url = setup?.bugreport_url
    if (!url) return

    const existing = document.querySelector('script[data-app="openvas-tracker"]')
    if (existing) {
      existing.setAttribute('data-user', user?.email || '')
      return
    }

    const script = document.createElement('script')
    script.src = `${url}/widget/bugreport.js`
    script.setAttribute('data-app', 'openvas-tracker')
    script.setAttribute('data-api', url)
    script.setAttribute('data-user', user?.email || '')
    document.body.appendChild(script)

    return () => { script.remove() }
  }, [setup?.bugreport_url, user?.email])

  return (
    <div className="flex min-h-screen bg-slate-950 text-white">
      <Sidebar />
      <div className="flex-1 flex flex-col">
        <header className="h-14 border-b border-slate-800 flex items-center justify-between px-6">
          <div />
          <div className="flex items-center gap-4">
            <span className="text-sm text-slate-400">{user?.email}</span>
            <button onClick={logout} className="text-sm text-slate-400 hover:text-white">Logout</button>
          </div>
        </header>
        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
