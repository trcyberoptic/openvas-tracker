// frontend/src/pages/Reports.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

interface Report {
  id: string
  name: string
  report_type: string
  format: string
  status: string
}

export function Reports() {
  const { data: reports = [] } = useQuery({ queryKey: ['reports'], queryFn: () => api.get<Report[]>('/reports') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Reports</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Name</th>
            <th className="text-left p-3 text-slate-400">Type</th>
            <th className="text-left p-3 text-slate-400">Format</th>
            <th className="text-left p-3 text-slate-400">Status</th>
          </tr></thead>
          <tbody>
            {reports.map((r) => (
              <tr key={r.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{r.name}</td>
                <td className="p-3 capitalize">{r.report_type}</td>
                <td className="p-3 uppercase text-slate-400">{r.format}</td>
                <td className="p-3 capitalize">{r.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
