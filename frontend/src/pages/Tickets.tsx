import { useState, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { api } from '@/api/client'
import { useAuth } from '@/hooks/useAuth'
import { TableFilter, useTableFilter, SortHeader, useSortable, useSorted } from '@/components/TableFilter'

const PRIORITY_COLORS: Record<string, string> = { critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600' }
const STATUS_COLORS: Record<string, string> = { open: 'bg-red-900 text-red-300', fixed: 'bg-green-900 text-green-300', risk_accepted: 'bg-yellow-900 text-yellow-300', false_positive: 'bg-slate-700 text-slate-300' }
const PRIO_ORDER: Record<string, number> = { critical: 1, high: 2, medium: 3, low: 4 }

function cvssColor(score: number | null | undefined): string {
  if (score == null) return 'text-slate-500'
  if (score >= 9) return 'text-red-400 font-bold'
  if (score >= 7) return 'text-orange-400 font-semibold'
  if (score >= 4) return 'text-yellow-400'
  return 'text-slate-400'
}

interface Ticket {
  id: string; title: string; priority: string; priority_order?: number; status: string
  affected_host?: string; hostname?: string; cvss_score?: number; cve_id?: string; assigned_to?: string
  first_seen_at?: string; last_seen_at?: string; created_at: string
}
interface UserRef { id: string; username: string; email: string }

export function Tickets() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { user } = useAuth()
  const qc = useQueryClient()
  const { data: raw = [] } = useQuery({ queryKey: ['tickets'], queryFn: () => api.get<Ticket[]>('/tickets') })
  const { data: users = [] } = useQuery({ queryKey: ['users'], queryFn: () => api.get<UserRef[]>('/settings/users') })
  const { values, setValues } = useTableFilter(['search', 'priority', 'status', 'host'])
  const { sort, toggle } = useSortable()
  const [selected, setSelected] = useState<Set<string>>(new Set())

  const assignedFilter = searchParams.get('assigned') // "me" | "unassigned" | null

  const tickets = useMemo(() => raw.map(t => ({ ...t, priority_order: PRIO_ORDER[t.priority] || 9 })), [raw])

  const hosts = useMemo(() => {
    const set = new Set(tickets.map(t => t.affected_host).filter(Boolean) as string[])
    return [...set].sort()
  }, [tickets])

  const filtered = useMemo(() => {
    let result = tickets
    if (assignedFilter === 'me' && user) result = result.filter(t => t.assigned_to === user.id)
    if (assignedFilter === 'unassigned') result = result.filter(t => !t.assigned_to)
    if (values.priority) result = result.filter(t => t.priority === values.priority)
    if (values.status) result = result.filter(t => t.status === values.status)
    if (values.host) result = result.filter(t => t.affected_host === values.host)
    if (values.search) {
      const q = values.search.toLowerCase()
      result = result.filter(t => {
        const assignedName = t.assigned_to ? users.find(u => u.id === t.assigned_to)?.username?.toLowerCase() : ''
        return t.title.toLowerCase().includes(q)
          || t.affected_host?.toLowerCase().includes(q)
          || t.hostname?.toLowerCase().includes(q)
          || t.priority.toLowerCase().includes(q)
          || t.status.toLowerCase().includes(q)
          || t.cve_id?.toLowerCase().includes(q)
          || t.cvss_score?.toFixed(1).includes(q)
          || assignedName?.includes(q)
      })
    }
    return result
  }, [tickets, values, assignedFilter, user])

  const effectiveSort = sort.key ? sort : { key: 'cvss_score', dir: 'desc' as const }
  const sorted = useSorted(filtered, effectiveSort)

  const bulkMut = useMutation({
    mutationFn: (body: { ticket_ids: string[]; status?: string; assigned_to?: string }) =>
      api.post('/tickets/bulk', body),
    onSuccess: () => { setSelected(new Set()); qc.invalidateQueries({ queryKey: ['tickets'] }) },
  })

  const toggleSelect = (id: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setSelected(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  const toggleAll = () => {
    if (selected.size === sorted.length) {
      setSelected(new Set())
    } else {
      setSelected(new Set(sorted.map(t => t.id)))
    }
  }

  const ids = [...selected]

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">Tickets</h1>
          {assignedFilter && (
            <span className="flex items-center gap-1.5 px-2 py-1 rounded bg-blue-900/50 text-blue-300 text-xs">
              {assignedFilter === 'me' ? 'My tickets' : 'Unassigned'}
              <button onClick={() => setSearchParams({})} className="hover:text-white">&times;</button>
            </span>
          )}
        </div>
        {selected.size > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-sm text-slate-400">{selected.size} selected</span>
            <button onClick={() => bulkMut.mutate({ ticket_ids: ids, status: 'fixed' })}
              className="px-3 py-1.5 rounded text-xs bg-green-900 text-green-300 hover:bg-green-800">Mark Fixed</button>
            <button onClick={() => bulkMut.mutate({ ticket_ids: ids, status: 'risk_accepted' })}
              className="px-3 py-1.5 rounded text-xs bg-yellow-900 text-yellow-300 hover:bg-yellow-800">Accept Risk</button>
            <button onClick={() => bulkMut.mutate({ ticket_ids: ids, status: 'false_positive' })}
              className="px-3 py-1.5 rounded text-xs bg-slate-700 text-slate-300 hover:bg-slate-600">False Positive</button>
            <select onChange={e => { if (e.target.value) bulkMut.mutate({ ticket_ids: ids, assigned_to: e.target.value }); e.target.value = '' }}
              className="bg-slate-800 border border-slate-700 rounded px-2 py-1.5 text-xs text-slate-300 focus:outline-none focus:border-blue-500">
              <option value="">Assign to...</option>
              {users.filter(u => u.username !== 'openvas-import').map(u => (
                <option key={u.id} value={u.id}>{u.username}</option>
              ))}
            </select>
          </div>
        )}
      </div>
      <TableFilter
        filters={[
          { key: 'search', label: 'Search tickets...' },
          { key: 'priority', label: 'Priority', options: ['critical', 'high', 'medium', 'low'] },
          { key: 'status', label: 'Status', options: ['open', 'fixed', 'risk_accepted', 'false_positive'] },
          { key: 'host', label: 'Host', options: hosts },
        ]}
        values={values} onChange={setValues}
      />
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="p-3 w-8">
              <input type="checkbox" checked={sorted.length > 0 && selected.size === sorted.length}
                onChange={toggleAll} className="rounded border-slate-600 bg-slate-800" />
            </th>
            <SortHeader label="CVSS" sortKey="cvss_score" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="Title" sortKey="title" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="Host" sortKey="affected_host" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="Priority" sortKey="priority_order" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="Status" sortKey="status" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="Assigned" sortKey="assigned_to" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="First Seen" sortKey="first_seen_at" sort={effectiveSort} onToggle={toggle} />
            <SortHeader label="Last Seen" sortKey="last_seen_at" sort={effectiveSort} onToggle={toggle} />
          </tr></thead>
          <tbody>
            {sorted.map(t => (
              <tr key={t.id} onClick={() => navigate(`/tickets/${t.id}`)} className={`border-b border-slate-800/50 hover:bg-slate-800/30 cursor-pointer ${selected.has(t.id) ? 'bg-blue-900/20' : ''}`}>
                <td className="p-3" onClick={e => toggleSelect(t.id, e)}>
                  <input type="checkbox" checked={selected.has(t.id)} readOnly className="rounded border-slate-600 bg-slate-800" />
                </td>
                <td className={`p-3 ${cvssColor(t.cvss_score)}`}>{t.cvss_score?.toFixed(1) ?? '—'}</td>
                <td className="p-3">{t.title}</td>
                <td className="p-3 text-slate-400">{t.hostname ? <><span className="font-mono">{t.affected_host}</span> <span className="text-slate-500 text-xs">({t.hostname})</span></> : <span className="font-mono">{t.affected_host || '—'}</span>}</td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs font-medium text-white ${PRIORITY_COLORS[t.priority] || 'bg-gray-600'}`}>{t.priority}</span></td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs ${STATUS_COLORS[t.status] || 'bg-slate-700 text-slate-300'}`}>{t.status.replace(/_/g, ' ')}</span></td>
                <td className="p-3 text-slate-400">{t.assigned_to ? users.find(u => u.id === t.assigned_to)?.username || '...' : <span className="text-slate-600">—</span>}</td>
                <td className="p-3 text-slate-400">{t.first_seen_at ? new Date(t.first_seen_at).toLocaleString() : '-'}</td>
                <td className="p-3 text-slate-400">{t.last_seen_at ? new Date(t.last_seen_at).toLocaleString() : '-'}</td>
              </tr>
            ))}
            {sorted.length === 0 && (<tr><td colSpan={9} className="p-6 text-center text-slate-500">{tickets.length > 0 ? 'No matches' : 'No tickets found'}</td></tr>)}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{filtered.length} of {tickets.length} tickets</p>
    </div>
  )
}
