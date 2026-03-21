import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '@/api/client'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { TableFilter, useTableFilter, SortHeader, useSortable, useSorted } from '@/components/TableFilter'

const STATUS_COLORS: Record<string, string> = { open: 'bg-red-900 text-red-300', fixed: 'bg-green-900 text-green-300', risk_accepted: 'bg-yellow-900 text-yellow-300', false_positive: 'bg-slate-700 text-slate-300' }
const PRIORITY_COLORS: Record<string, string> = { critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600' }

interface HostSummary {
  host: string; hostname?: string; vuln_count: number; critical_count: number; high_count: number; max_cvss?: number
  open_tickets: number; fixed_tickets: number; risk_accepted_tickets: number; false_positive_tickets: number
}
interface HostTicket {
  id: string; title: string; status: string; priority: string; cvss_score?: number; cve_id?: string
  first_seen_at?: string; last_seen_at?: string
}

function HostRow({ h }: { h: HostSummary }) {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()
  const { data: tickets = [] } = useQuery({
    queryKey: ['host-tickets', h.host],
    queryFn: () => api.get<HostTicket[]>(`/hosts/${h.host}/tickets`),
    enabled: open,
  })
  return (
    <>
      <tr onClick={() => setOpen(!open)} className="border-b border-slate-800/50 hover:bg-slate-800/30 cursor-pointer">
        <td className="p-3">
          {open ? <ChevronDown size={16} className="inline mr-2" /> : <ChevronRight size={16} className="inline mr-2" />}
          <span className="font-mono">{h.host}</span>
          {h.hostname && <span className="text-slate-500 text-xs ml-2">({h.hostname})</span>}
        </td>
        <td className="p-3">{h.vuln_count}</td>
        <td className="p-3">{h.critical_count > 0 ? <span className="px-2 py-1 rounded text-xs bg-red-600 text-white">{h.critical_count}</span> : <span className="text-slate-500">0</span>}</td>
        <td className="p-3">{h.high_count > 0 ? <span className="px-2 py-1 rounded text-xs bg-orange-600 text-white">{h.high_count}</span> : <span className="text-slate-500">0</span>}</td>
        <td className="p-3">{h.max_cvss?.toFixed(1) ?? '-'}</td>
        <td className="p-3">
          {h.open_tickets > 0 && <span className="px-2 py-0.5 rounded text-xs bg-red-900 text-red-300 mr-1">{h.open_tickets} open</span>}
          {h.fixed_tickets > 0 && <span className="px-2 py-0.5 rounded text-xs bg-green-900 text-green-300 mr-1">{h.fixed_tickets} fixed</span>}
          {h.risk_accepted_tickets > 0 && <span className="px-2 py-0.5 rounded text-xs bg-yellow-900 text-yellow-300 mr-1">{h.risk_accepted_tickets} accepted</span>}
          {h.false_positive_tickets > 0 && <span className="px-2 py-0.5 rounded text-xs bg-slate-700 text-slate-300">{h.false_positive_tickets} FP</span>}
          {h.open_tickets === 0 && h.fixed_tickets === 0 && h.risk_accepted_tickets === 0 && h.false_positive_tickets === 0 && <span className="text-slate-500">—</span>}
        </td>
      </tr>
      {open && tickets.map(t => (
        <tr key={t.id} onClick={() => navigate(`/tickets/${t.id}`)} className="bg-slate-800/20 border-b border-slate-800/30 hover:bg-slate-800/40 cursor-pointer">
          <td className="p-3 pl-10">
            <span className={`px-2 py-1 rounded text-xs font-medium text-white ${PRIORITY_COLORS[t.priority] || 'bg-gray-600'}`}>{t.priority}</span>
            <span className="ml-3">{t.title}</span>
          </td>
          <td className="p-3 text-slate-400">{t.cve_id || '-'}</td>
          <td className="p-3">{t.cvss_score?.toFixed(1) ?? '-'}</td>
          <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${STATUS_COLORS[t.status] || 'bg-slate-700 text-slate-300'}`}>{t.status.replace(/_/g, ' ')}</span></td>
          <td className="p-3 text-slate-400 text-xs">{t.last_seen_at ? new Date(t.last_seen_at).toLocaleDateString() : '-'}</td>
          <td></td>
        </tr>
      ))}
      {open && tickets.length === 0 && (
        <tr className="bg-slate-800/20 border-b border-slate-800/30">
          <td colSpan={6} className="p-3 pl-10 text-slate-500 text-sm">No tickets for this host</td>
        </tr>
      )}
    </>
  )
}

export function Targets() {
  const { data: hosts = [] } = useQuery({ queryKey: ['hosts'], queryFn: () => api.get<HostSummary[]>('/hosts') })
  const { values, setValues } = useTableFilter(['search'])
  const { sort, toggle } = useSortable()

  const filtered = useMemo(() => {
    if (!values.search) return hosts
    const q = values.search.toLowerCase()
    return hosts.filter(h => h.host.toLowerCase().includes(q) || h.hostname?.toLowerCase().includes(q))
  }, [hosts, values])

  const sorted = useSorted(filtered, sort)

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Hosts</h1>
      <TableFilter filters={[{ key: 'search', label: 'Search hosts...' }]} values={values} onChange={setValues} />
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <SortHeader label="Host" sortKey="host" sort={sort} onToggle={toggle} />
            <SortHeader label="Vulns" sortKey="vuln_count" sort={sort} onToggle={toggle} />
            <SortHeader label="Critical" sortKey="critical_count" sort={sort} onToggle={toggle} />
            <SortHeader label="High" sortKey="high_count" sort={sort} onToggle={toggle} />
            <SortHeader label="Max CVSS" sortKey="max_cvss" sort={sort} onToggle={toggle} />
            <SortHeader label="Tickets" sortKey="open_tickets" sort={sort} onToggle={toggle} />
          </tr></thead>
          <tbody>
            {sorted.map(h => <HostRow key={h.host} h={h} />)}
            {sorted.length === 0 && (
              <tr><td colSpan={6} className="p-6 text-center text-slate-500">{hosts.length > 0 ? 'No matches' : 'No hosts found'}</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{filtered.length} of {hosts.length} hosts</p>
    </div>
  )
}
