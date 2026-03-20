// frontend/src/pages/Targets.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

interface Target {
  id: string
  host: string
  ip_address: string
  hostname?: string
  os_guess?: string
}

export function Targets() {
  const { data: targets = [] } = useQuery({ queryKey: ['targets'], queryFn: () => api.get<Target[]>('/targets') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Targets</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Host</th>
            <th className="text-left p-3 text-slate-400">IP Address</th>
            <th className="text-left p-3 text-slate-400">Hostname</th>
            <th className="text-left p-3 text-slate-400">OS</th>
          </tr></thead>
          <tbody>
            {targets.map((t) => (
              <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{t.host}</td>
                <td className="p-3 text-slate-400">{t.ip_address}</td>
                <td className="p-3 text-slate-400">{t.hostname || '-'}</td>
                <td className="p-3 text-slate-400">{t.os_guess || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
