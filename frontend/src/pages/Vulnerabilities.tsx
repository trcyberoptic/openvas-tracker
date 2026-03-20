import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { TableFilter, useTableFilter, SortHeader, useSortable, useSorted } from '@/components/TableFilter'

const BADGE_COLORS: Record<string, string> = {
  critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600', info: 'bg-gray-600',
}
const SEV_ORDER: Record<string, number> = { critical: 1, high: 2, medium: 3, low: 4, info: 5 }

interface Vuln {
  id: string
  scan_id: string
  severity: string
  severity_order?: number
  title: string
  affected_host: string
  affected_port?: number
  protocol?: string
  cve_id?: string
  cvss_score?: number
  description?: string
  solution?: string
  status: string
}

function VulnRow({ v }: { v: Vuln }) {
  const [open, setOpen] = useState(false)
  return (
    <>
      <tr onClick={() => setOpen(!open)} className="border-b border-slate-800/50 hover:bg-slate-800/30 cursor-pointer">
        <td className="p-3">
          {open ? <ChevronDown size={16} className="inline mr-1" /> : <ChevronRight size={16} className="inline mr-1" />}
          <span className={`px-2 py-1 rounded text-xs font-medium text-white ${BADGE_COLORS[v.severity] || 'bg-gray-600'}`}>{v.severity}</span>
        </td>
        <td className="p-3">{v.title}</td>
        <td className="p-3 text-slate-400">{v.affected_host}</td>
        <td className="p-3 text-slate-400">{v.cve_id || '-'}</td>
        <td className="p-3">{v.cvss_score ?? '-'}</td>
        <td className="p-3"><Link to={`/scans/${v.scan_id}`} onClick={e => e.stopPropagation()} className="text-blue-400 hover:underline">Scan</Link></td>
        <td className="p-3 text-slate-400">{v.status}</td>
      </tr>
      {open && (
        <tr className="bg-slate-800/20 border-b border-slate-800/30">
          <td colSpan={7} className="p-4 pl-10">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <h4 className="text-slate-400 font-medium mb-1">Description</h4>
                <p className="text-slate-300 whitespace-pre-wrap">{v.description || 'No description available'}</p>
              </div>
              <div>
                <h4 className="text-slate-400 font-medium mb-1">Solution</h4>
                <p className="text-slate-300 whitespace-pre-wrap">{v.solution || 'No solution available'}</p>
                <div className="mt-3 space-y-1 text-slate-400">
                  <p><span className="text-slate-500">Port:</span> {v.affected_port ? `${v.affected_port}/${v.protocol || ''}` : '-'}</p>
                  <p><span className="text-slate-500">CVE:</span> {v.cve_id || '-'}</p>
                  <p><span className="text-slate-500">CVSS:</span> {v.cvss_score ?? '-'}</p>
                </div>
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

export function Vulnerabilities() {
  const { data: raw = [] } = useQuery({ queryKey: ['vulnerabilities'], queryFn: () => api.get<Vuln[]>('/vulnerabilities') })
  const { values, setValues } = useTableFilter(['search', 'severity', 'status'])
  const { sort, toggle } = useSortable()

  const vulns = useMemo(() => raw.map(v => ({ ...v, severity_order: SEV_ORDER[v.severity] || 9 })), [raw])

  const filtered = useMemo(() => {
    let result = vulns
    if (values.severity) result = result.filter(v => v.severity === values.severity)
    if (values.status) result = result.filter(v => v.status === values.status)
    if (values.search) {
      const q = values.search.toLowerCase()
      result = result.filter(v => v.title.toLowerCase().includes(q) || v.affected_host?.toLowerCase().includes(q) || v.cve_id?.toLowerCase().includes(q))
    }
    return result
  }, [vulns, values])

  const sorted = useSorted(filtered, sort)

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Vulnerabilities</h1>
      <TableFilter
        filters={[
          { key: 'search', label: 'Search title, host, CVE...' },
          { key: 'severity', label: 'Severity', options: ['critical', 'high', 'medium', 'low', 'info'] },
          { key: 'status', label: 'Status', options: ['open', 'confirmed', 'mitigated', 'resolved', 'false_positive'] },
        ]}
        values={values} onChange={setValues}
      />
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <SortHeader label="Severity" sortKey="severity_order" sort={sort} onToggle={toggle} />
            <SortHeader label="Title" sortKey="title" sort={sort} onToggle={toggle} />
            <SortHeader label="Host" sortKey="affected_host" sort={sort} onToggle={toggle} />
            <SortHeader label="CVE" sortKey="cve_id" sort={sort} onToggle={toggle} />
            <SortHeader label="CVSS" sortKey="cvss_score" sort={sort} onToggle={toggle} />
            <th className="text-left p-3 text-slate-400">Scan</th>
            <SortHeader label="Status" sortKey="status" sort={sort} onToggle={toggle} />
          </tr></thead>
          <tbody>
            {sorted.map((v) => <VulnRow key={v.id} v={v} />)}
            {sorted.length === 0 && (
              <tr><td colSpan={7} className="p-6 text-center text-slate-500">{vulns.length > 0 ? 'No matches' : 'No vulnerabilities found'}</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{filtered.length} of {vulns.length} vulnerabilities</p>
    </div>
  )
}
