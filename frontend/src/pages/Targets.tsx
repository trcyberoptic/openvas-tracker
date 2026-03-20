import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

interface HostSummary {
  host: string
  vuln_count: number
  critical_count: number
  high_count: number
  max_cvss?: number
}

export function Targets() {
  const { data: hosts = [] } = useQuery({ queryKey: ['hosts'], queryFn: () => api.get<HostSummary[]>('/hosts') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Hosts</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Host</th>
            <th className="text-left p-3 text-slate-400">Vulnerabilities</th>
            <th className="text-left p-3 text-slate-400">Critical</th>
            <th className="text-left p-3 text-slate-400">High</th>
            <th className="text-left p-3 text-slate-400">Max CVSS</th>
          </tr></thead>
          <tbody>
            {hosts.map((h) => (
              <tr key={h.host} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3 font-mono">{h.host}</td>
                <td className="p-3">{h.vuln_count}</td>
                <td className="p-3">{h.critical_count > 0 ? <span className="px-2 py-1 rounded text-xs bg-red-600 text-white">{h.critical_count}</span> : <span className="text-slate-500">0</span>}</td>
                <td className="p-3">{h.high_count > 0 ? <span className="px-2 py-1 rounded text-xs bg-orange-600 text-white">{h.high_count}</span> : <span className="text-slate-500">0</span>}</td>
                <td className="p-3">{h.max_cvss?.toFixed(1) ?? '-'}</td>
              </tr>
            ))}
            {hosts.length === 0 && (
              <tr><td colSpan={5} className="p-6 text-center text-slate-500">No hosts found</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
