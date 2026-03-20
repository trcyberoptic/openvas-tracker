import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { ChevronDown, ChevronRight } from 'lucide-react'

const BADGE_COLORS: Record<string, string> = {
  critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600', info: 'bg-gray-600',
}

interface HostSummary {
  host: string
  vuln_count: number
  critical_count: number
  high_count: number
  max_cvss?: number
}

interface Vuln {
  id: string
  severity: string
  title: string
  cve_id?: string
  cvss_score?: number
  affected_port?: number
  protocol?: string
  solution?: string
  status: string
}

function HostRow({ h }: { h: HostSummary }) {
  const [open, setOpen] = useState(false)
  const { data: vulns = [] } = useQuery({
    queryKey: ['host-vulns', h.host],
    queryFn: () => api.get<Vuln[]>(`/hosts/${h.host}/vulnerabilities`),
    enabled: open,
  })

  return (
    <>
      <tr onClick={() => setOpen(!open)} className="border-b border-slate-800/50 hover:bg-slate-800/30 cursor-pointer">
        <td className="p-3">
          {open ? <ChevronDown size={16} className="inline mr-2" /> : <ChevronRight size={16} className="inline mr-2" />}
          <span className="font-mono">{h.host}</span>
        </td>
        <td className="p-3">{h.vuln_count}</td>
        <td className="p-3">{h.critical_count > 0 ? <span className="px-2 py-1 rounded text-xs bg-red-600 text-white">{h.critical_count}</span> : <span className="text-slate-500">0</span>}</td>
        <td className="p-3">{h.high_count > 0 ? <span className="px-2 py-1 rounded text-xs bg-orange-600 text-white">{h.high_count}</span> : <span className="text-slate-500">0</span>}</td>
        <td className="p-3">{h.max_cvss?.toFixed(1) ?? '-'}</td>
      </tr>
      {open && vulns.map((v) => (
        <tr key={v.id} className="bg-slate-800/20 border-b border-slate-800/30">
          <td className="p-3 pl-10">
            <span className={`px-2 py-1 rounded text-xs font-medium text-white ${BADGE_COLORS[v.severity] || 'bg-gray-600'}`}>
              {v.severity}
            </span>
            <span className="ml-3">{v.title}</span>
          </td>
          <td className="p-3 text-slate-400">{v.cve_id || '-'}</td>
          <td className="p-3">{v.cvss_score ?? '-'}</td>
          <td className="p-3 text-slate-400">{v.affected_port ? `${v.affected_port}/${v.protocol || ''}` : '-'}</td>
          <td className="p-3 text-slate-400 text-xs max-w-xs truncate">{v.solution || '-'}</td>
        </tr>
      ))}
    </>
  )
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
            <th className="text-left p-3 text-slate-400">Vulns</th>
            <th className="text-left p-3 text-slate-400">Critical</th>
            <th className="text-left p-3 text-slate-400">High</th>
            <th className="text-left p-3 text-slate-400">Max CVSS</th>
          </tr></thead>
          <tbody>
            {hosts.map((h) => <HostRow key={h.host} h={h} />)}
            {hosts.length === 0 && (
              <tr><td colSpan={5} className="p-6 text-center text-slate-500">No hosts found</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
