// frontend/src/pages/Scans.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

interface Scan {
  id: string
  name: string
  scan_type: string
  status: string
  created_at: string
}

export function Scans() {
  const { data: scans = [] } = useQuery({ queryKey: ['scans'], queryFn: () => api.get<Scan[]>('/scans') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Scans</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Name</th>
            <th className="text-left p-3 text-slate-400">Type</th>
            <th className="text-left p-3 text-slate-400">Status</th>
            <th className="text-left p-3 text-slate-400">Created</th>
          </tr></thead>
          <tbody>
            {scans.map((s) => (
              <tr key={s.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{s.name}</td>
                <td className="p-3 text-slate-400">{s.scan_type}</td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs ${s.status === 'completed' ? 'bg-green-900 text-green-300' : s.status === 'running' ? 'bg-blue-900 text-blue-300' : s.status === 'failed' ? 'bg-red-900 text-red-300' : 'bg-slate-700 text-slate-300'}`}>{s.status}</span></td>
                <td className="p-3 text-slate-400">{new Date(s.created_at).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
