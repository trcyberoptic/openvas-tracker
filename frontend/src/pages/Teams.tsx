// frontend/src/pages/Teams.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

interface Team {
  id: string
  name: string
  description?: string
}

export function Teams() {
  const { data: teams = [] } = useQuery({ queryKey: ['teams'], queryFn: () => api.get<Team[]>('/teams') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Teams</h1>
      <div className="grid grid-cols-3 gap-4">
        {teams.map((t) => (
          <div key={t.id} className="bg-slate-900 rounded-lg border border-slate-800 p-4">
            <h3 className="font-semibold">{t.name}</h3>
            <p className="text-sm text-slate-400 mt-1">{t.description || 'No description'}</p>
          </div>
        ))}
        {teams.length === 0 && <p className="text-slate-500 col-span-3">No teams yet</p>}
      </div>
    </div>
  )
}
