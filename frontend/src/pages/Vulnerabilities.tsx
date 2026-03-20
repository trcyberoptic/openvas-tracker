// frontend/src/pages/Vulnerabilities.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

const BADGE_COLORS: Record<string, string> = {
  critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600', info: 'bg-gray-600',
}

interface Vuln {
  id: string
  severity: string
  title: string
  affected_host: string
  cve_id?: string
  cvss_score?: number
  status: string
}

export function Vulnerabilities() {
  const { data: vulns = [] } = useQuery({
    queryKey: ['vulnerabilities'],
    queryFn: () => api.get<Vuln[]>('/vulnerabilities'),
  })

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Vulnerabilities</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800">
              <th className="text-left p-3 text-slate-400">Severity</th>
              <th className="text-left p-3 text-slate-400">Title</th>
              <th className="text-left p-3 text-slate-400">Host</th>
              <th className="text-left p-3 text-slate-400">CVE</th>
              <th className="text-left p-3 text-slate-400">CVSS</th>
              <th className="text-left p-3 text-slate-400">Status</th>
            </tr>
          </thead>
          <tbody>
            {vulns.map((v) => (
              <tr key={v.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium text-white ${BADGE_COLORS[v.severity] || 'bg-gray-600'}`}>
                    {v.severity}
                  </span>
                </td>
                <td className="p-3">{v.title}</td>
                <td className="p-3 text-slate-400">{v.affected_host}</td>
                <td className="p-3 text-slate-400">{v.cve_id || '-'}</td>
                <td className="p-3">{v.cvss_score ?? '-'}</td>
                <td className="p-3 text-slate-400">{v.status}</td>
              </tr>
            ))}
            {vulns.length === 0 && (
              <tr><td colSpan={6} className="p-6 text-center text-slate-500">No vulnerabilities found</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
