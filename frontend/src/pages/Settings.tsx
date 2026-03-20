// frontend/src/pages/Settings.tsx
import { useAuth } from '@/hooks/useAuth'

export function Settings() {
  const { user } = useAuth()
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Settings</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6 max-w-lg">
        <h2 className="text-lg font-semibold mb-4">Profile</h2>
        <div className="space-y-3 text-sm">
          <div><span className="text-slate-400">Email:</span> <span>{user?.email}</span></div>
          <div><span className="text-slate-400">Username:</span> <span>{user?.username}</span></div>
          <div><span className="text-slate-400">Role:</span> <span className="capitalize">{user?.role}</span></div>
        </div>
      </div>
    </div>
  )
}
