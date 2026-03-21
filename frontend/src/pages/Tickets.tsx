import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '@/api/client'
import { TableFilter, useTableFilter, SortHeader, useSortable, useSorted } from '@/components/TableFilter'

const PRIORITY_COLORS: Record<string, string> = { critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600' }
const STATUS_COLORS: Record<string, string> = { open: 'bg-red-900 text-red-300', fixed: 'bg-green-900 text-green-300', risk_accepted: 'bg-yellow-900 text-yellow-300' }
const PRIO_ORDER: Record<string, number> = { critical: 1, high: 2, medium: 3, low: 4 }

interface Ticket {
  id: string; title: string; priority: string; priority_order?: number; status: string
  affected_host?: string; assigned_to?: string; first_seen_at?: string; last_seen_at?: string; created_at: string
}
interface UserRef { id: string; username: string; email: string }

export function Tickets() {
  const navigate = useNavigate()
  const { data: raw = [] } = useQuery({ queryKey: ['tickets'], queryFn: () => api.get<Ticket[]>('/tickets') })
  const { data: users = [] } = useQuery({ queryKey: ['users'], queryFn: () => api.get<UserRef[]>('/settings/users') })
  const { values, setValues } = useTableFilter(['search', 'priority', 'status', 'host'])
  const { sort, toggle } = useSortable()

  const tickets = useMemo(() => raw.map(t => ({ ...t, priority_order: PRIO_ORDER[t.priority] || 9 })), [raw])

  const hosts = useMemo(() => {
    const set = new Set(tickets.map(t => t.affected_host).filter(Boolean) as string[])
    return [...set].sort()
  }, [tickets])

  const filtered = useMemo(() => {
    let result = tickets
    if (values.priority) result = result.filter(t => t.priority === values.priority)
    if (values.status) result = result.filter(t => t.status === values.status)
    if (values.host) result = result.filter(t => t.affected_host === values.host)
    if (values.search) { const q = values.search.toLowerCase(); result = result.filter(t => t.title.toLowerCase().includes(q) || t.affected_host?.toLowerCase().includes(q)) }
    return result
  }, [tickets, values])

  const sorted = useSorted(filtered, sort)

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Tickets</h1>
      <TableFilter
        filters={[
          { key: 'search', label: 'Search tickets...' },
          { key: 'priority', label: 'Priority', options: ['critical', 'high', 'medium', 'low'] },
          { key: 'status', label: 'Status', options: ['open', 'fixed', 'risk_accepted'] },
          { key: 'host', label: 'Host', options: hosts },
        ]}
        values={values} onChange={setValues}
      />
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <SortHeader label="Title" sortKey="title" sort={sort} onToggle={toggle} />
            <SortHeader label="Host" sortKey="affected_host" sort={sort} onToggle={toggle} />
            <SortHeader label="Priority" sortKey="priority_order" sort={sort} onToggle={toggle} />
            <SortHeader label="Status" sortKey="status" sort={sort} onToggle={toggle} />
            <SortHeader label="Assigned To" sortKey="assigned_to" sort={sort} onToggle={toggle} />
            <SortHeader label="First Seen" sortKey="first_seen_at" sort={sort} onToggle={toggle} />
            <SortHeader label="Last Seen" sortKey="last_seen_at" sort={sort} onToggle={toggle} />
          </tr></thead>
          <tbody>
            {sorted.map(t => (
              <tr key={t.id} onClick={() => navigate(`/tickets/${t.id}`)} className="border-b border-slate-800/50 hover:bg-slate-800/30 cursor-pointer">
                <td className="p-3">{t.title}</td>
                <td className="p-3 font-mono text-slate-400">{t.affected_host || '—'}</td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs font-medium text-white ${PRIORITY_COLORS[t.priority] || 'bg-gray-600'}`}>{t.priority}</span></td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs ${STATUS_COLORS[t.status] || 'bg-slate-700 text-slate-300'}`}>{t.status.replace('_', ' ')}</span></td>
                <td className="p-3 text-slate-400">{t.assigned_to ? users.find(u => u.id === t.assigned_to)?.username || '...' : <span className="text-slate-600">—</span>}</td>
                <td className="p-3 text-slate-400">{t.first_seen_at ? new Date(t.first_seen_at).toLocaleString() : '-'}</td>
                <td className="p-3 text-slate-400">{t.last_seen_at ? new Date(t.last_seen_at).toLocaleString() : '-'}</td>
              </tr>
            ))}
            {sorted.length === 0 && (<tr><td colSpan={7} className="p-6 text-center text-slate-500">{tickets.length > 0 ? 'No matches' : 'No tickets found'}</td></tr>)}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{filtered.length} of {tickets.length} tickets</p>
    </div>
  )
}
