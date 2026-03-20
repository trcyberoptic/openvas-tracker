import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import { TableFilter, useTableFilter } from '@/components/TableFilter'

const PRIORITY_COLORS: Record<string, string> = {
  critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600',
}
const STATUS_COLORS: Record<string, string> = {
  open: 'bg-red-900 text-red-300', in_progress: 'bg-blue-900 text-blue-300', review: 'bg-purple-900 text-purple-300',
  resolved: 'bg-green-900 text-green-300', closed: 'bg-slate-700 text-slate-300',
}

interface Ticket {
  id: string
  title: string
  priority: string
  status: string
  assigned_to?: string
  first_seen_at?: string
  last_seen_at?: string
  created_at: string
}

export function Tickets() {
  const { data: tickets = [] } = useQuery({ queryKey: ['tickets'], queryFn: () => api.get<Ticket[]>('/tickets') })
  const { values, setValues } = useTableFilter(['search', 'priority', 'status'])

  const filtered = useMemo(() => {
    let result = tickets
    if (values.priority) result = result.filter(t => t.priority === values.priority)
    if (values.status) result = result.filter(t => t.status === values.status)
    if (values.search) {
      const q = values.search.toLowerCase()
      result = result.filter(t => t.title.toLowerCase().includes(q))
    }
    return result
  }, [tickets, values])

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Tickets</h1>
      <TableFilter
        filters={[
          { key: 'search', label: 'Search tickets...' },
          { key: 'priority', label: 'Priority', options: ['critical', 'high', 'medium', 'low'] },
          { key: 'status', label: 'Status', options: ['open', 'in_progress', 'review', 'resolved', 'closed'] },
        ]}
        values={values}
        onChange={setValues}
      />
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Title</th>
            <th className="text-left p-3 text-slate-400">Priority</th>
            <th className="text-left p-3 text-slate-400">Status</th>
            <th className="text-left p-3 text-slate-400">First Seen</th>
            <th className="text-left p-3 text-slate-400">Last Seen</th>
          </tr></thead>
          <tbody>
            {filtered.map((t) => (
              <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{t.title}</td>
                <td className="p-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium text-white ${PRIORITY_COLORS[t.priority] || 'bg-gray-600'}`}>{t.priority}</span>
                </td>
                <td className="p-3">
                  <span className={`px-2 py-1 rounded text-xs ${STATUS_COLORS[t.status] || 'bg-slate-700 text-slate-300'}`}>{t.status.replace('_', ' ')}</span>
                </td>
                <td className="p-3 text-slate-400">{t.first_seen_at ? new Date(t.first_seen_at).toLocaleString() : '-'}</td>
                <td className="p-3 text-slate-400">{t.last_seen_at ? new Date(t.last_seen_at).toLocaleString() : '-'}</td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr><td colSpan={5} className="p-6 text-center text-slate-500">{tickets.length > 0 ? 'No matches' : 'No tickets found'}</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{filtered.length} of {tickets.length} tickets</p>
    </div>
  )
}
