// frontend/src/pages/Tickets.tsx
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

interface Ticket {
  id: string
  title: string
  priority: string
  status: string
  due_date?: string
}

export function Tickets() {
  const { data: tickets = [] } = useQuery({ queryKey: ['tickets'], queryFn: () => api.get<Ticket[]>('/tickets') })
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Tickets</h1>
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <th className="text-left p-3 text-slate-400">Title</th>
            <th className="text-left p-3 text-slate-400">Priority</th>
            <th className="text-left p-3 text-slate-400">Status</th>
            <th className="text-left p-3 text-slate-400">Due Date</th>
          </tr></thead>
          <tbody>
            {tickets.map((t) => (
              <tr key={t.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">{t.title}</td>
                <td className="p-3 capitalize">{t.priority}</td>
                <td className="p-3 capitalize">{t.status}</td>
                <td className="p-3 text-slate-400">{t.due_date ? new Date(t.due_date).toLocaleDateString() : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
