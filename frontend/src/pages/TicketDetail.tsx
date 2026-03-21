import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/api/client'

const PRIORITY_COLORS: Record<string, string> = { critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600' }
const STATUS_COLORS: Record<string, string> = { open: 'bg-red-900 text-red-300', fixed: 'bg-green-900 text-green-300', risk_accepted: 'bg-yellow-900 text-yellow-300', false_positive: 'bg-slate-700 text-slate-300' }

interface Ticket {
  id: string; title: string; description?: string; priority: string; status: string
  vulnerability_id?: string; assigned_to?: string; risk_accepted_until?: string; affected_host?: string
  first_seen_at?: string; last_seen_at?: string; created_at: string
}
interface Comment { id: string; user_id: string; content: string; created_at: string }
interface Activity { id: string; action: string; old_value?: string; new_value?: string; changed_by: string; note?: string; created_at: string }
interface UserRef { id: string; username: string; email: string }
interface AlsoAffected { host: string; ticket_id: string; status: string }

export function TicketDetail() {
  const { id } = useParams<{ id: string }>()
  const qc = useQueryClient()
  const [comment, setComment] = useState('')
  const [riskUntil, setRiskUntil] = useState('')

  const { data: ticket } = useQuery({ queryKey: ['ticket', id], queryFn: () => api.get<Ticket>(`/tickets/${id}`) })
  const { data: comments = [] } = useQuery({ queryKey: ['ticket-comments', id], queryFn: () => api.get<Comment[]>(`/tickets/${id}/comments`) })
  const { data: activity = [] } = useQuery({ queryKey: ['ticket-activity', id], queryFn: () => api.get<Activity[]>(`/tickets/${id}/activity`) })
  const { data: users = [] } = useQuery({ queryKey: ['users'], queryFn: () => api.get<UserRef[]>('/settings/users') })
  const { data: alsoAffected = [] } = useQuery({ queryKey: ['ticket-also-affected', id], queryFn: () => api.get<AlsoAffected[]>(`/tickets/${id}/also-affected`) })

  const invalidateTicket = () => {
    qc.invalidateQueries({ queryKey: ['ticket', id] })
    qc.invalidateQueries({ queryKey: ['ticket-activity', id] })
  }

  const statusMut = useMutation({
    mutationFn: ({ status, risk_accepted_until }: { status: string; risk_accepted_until?: string }) =>
      api.patch(`/tickets/${id}/status`, { status, risk_accepted_until }),
    onSuccess: invalidateTicket,
  })

  const assignMut = useMutation({
    mutationFn: (assigned_to: string | null) => api.patch(`/tickets/${id}/assign`, { assigned_to }),
    onSuccess: invalidateTicket,
  })

  const commentMut = useMutation({
    mutationFn: (content: string) => api.post(`/tickets/${id}/comments`, { content }),
    onSuccess: () => { setComment(''); qc.invalidateQueries({ queryKey: ['ticket-comments', id] }); qc.invalidateQueries({ queryKey: ['ticket-activity', id] }) },
  })

  if (!ticket) return null

  const assignedUser = users.find(u => u.id === ticket.assigned_to)

  return (
    <div className="max-w-4xl">
      <Link to="/tickets" className="text-blue-400 hover:underline text-sm mb-4 inline-block">&larr; Back to Tickets</Link>

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold mb-2">{ticket.title}</h1>
          <div className="flex items-center gap-3">
            <span className={`px-2 py-1 rounded text-xs font-medium text-white ${PRIORITY_COLORS[ticket.priority]}`}>{ticket.priority}</span>
            <span className={`px-2 py-1 rounded text-xs ${STATUS_COLORS[ticket.status] || 'bg-slate-700 text-slate-300'}`}>{ticket.status.replace('_', ' ')}</span>
            {ticket.first_seen_at && <span className="text-slate-500 text-xs">First seen: {new Date(ticket.first_seen_at).toLocaleString()}</span>}
            {ticket.last_seen_at && <span className="text-slate-500 text-xs">Last seen: {new Date(ticket.last_seen_at).toLocaleString()}</span>}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-6">
        {/* Status actions */}
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Status</h3>
          <div className="flex flex-col gap-2">
            <div className="flex gap-2">
              {ticket.status !== 'open' && ticket.status !== 'false_positive' && (
                <button onClick={() => statusMut.mutate({ status: 'open' })} className="px-3 py-1.5 rounded text-sm bg-red-900 text-red-300 hover:bg-red-800">Reopen</button>
              )}
              {ticket.status === 'false_positive' && (
                <button onClick={() => statusMut.mutate({ status: 'open' })} className="px-3 py-1.5 rounded text-sm bg-red-900 text-red-300 hover:bg-red-800">Not False Positive</button>
              )}
              {ticket.status === 'open' && (
                <>
                  <button onClick={() => statusMut.mutate({ status: 'fixed' })} className="px-3 py-1.5 rounded text-sm bg-green-900 text-green-300 hover:bg-green-800">Mark Fixed</button>
                  <button onClick={() => statusMut.mutate({ status: 'risk_accepted', risk_accepted_until: riskUntil || undefined })} className="px-3 py-1.5 rounded text-sm bg-yellow-900 text-yellow-300 hover:bg-yellow-800">Accept Risk</button>
                  <button onClick={() => statusMut.mutate({ status: 'false_positive' })} className="px-3 py-1.5 rounded text-sm bg-slate-700 text-slate-300 hover:bg-slate-600">False Positive</button>
                </>
              )}
            </div>
            {ticket.status === 'open' && (
              <div className="flex items-center gap-2 mt-1">
                <label className="text-xs text-slate-500">Risk accepted until:</label>
                <input type="date" value={riskUntil} onChange={e => setRiskUntil(e.target.value)}
                  className="bg-slate-800 border border-slate-700 rounded px-2 py-1 text-xs text-slate-300 focus:outline-none focus:border-blue-500" />
              </div>
            )}
            {ticket.status === 'risk_accepted' && ticket.risk_accepted_until && (
              <p className="text-xs text-yellow-400 mt-1">Expires: {new Date(ticket.risk_accepted_until).toLocaleDateString()}</p>
            )}
          </div>
        </div>

        {/* Assignment */}
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-4">
          <h3 className="text-sm font-medium text-slate-400 mb-3">Assigned To</h3>
          <select
            value={ticket.assigned_to || ''}
            onChange={e => assignMut.mutate(e.target.value || null)}
            className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-300 focus:outline-none focus:border-blue-500 w-full"
          >
            <option value="">Unassigned</option>
            {users.filter(u => u.username !== 'openvas-import').map(u => (
              <option key={u.id} value={u.id}>{u.username} ({u.email})</option>
            ))}
          </select>
          {assignedUser && <p className="text-xs text-slate-500 mt-1">{assignedUser.email}</p>}
        </div>
      </div>

      {/* Description */}
      {ticket.description && (
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-4 mb-6">
          <h3 className="text-sm font-medium text-slate-400 mb-2">Description</h3>
          <p className="text-slate-300 whitespace-pre-wrap text-sm">{ticket.description}</p>
        </div>
      )}

      {/* Linked vulnerability */}
      {ticket.affected_host && (
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-4 mb-6">
          <h3 className="text-sm font-medium text-slate-400 mb-2">Host</h3>
          <Link to="/hosts" className="text-blue-400 hover:underline text-sm font-mono">{ticket.affected_host}</Link>
          {alsoAffected.length > 0 && (
            <div className="mt-3 border-t border-slate-800 pt-3">
              <h4 className="text-xs text-slate-500 mb-2">Also affected ({alsoAffected.length})</h4>
              <div className="flex flex-wrap gap-2">
                {alsoAffected.map(a => (
                  <Link key={a.ticket_id} to={`/tickets/${a.ticket_id}`}
                    className="inline-flex items-center gap-1.5 px-2 py-1 rounded bg-slate-800 hover:bg-slate-700 text-xs">
                    <span className="font-mono text-slate-300">{a.host}</span>
                    <span className={`px-1.5 py-0.5 rounded text-[10px] ${STATUS_COLORS[a.status] || 'bg-slate-700 text-slate-300'}`}>{a.status.replace(/_/g, ' ')}</span>
                  </Link>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Comments */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-4 mb-6">
        <h3 className="text-sm font-medium text-slate-400 mb-3">Notes</h3>
        <div className="space-y-3 mb-4">
          {comments.map(c => (
            <div key={c.id} className="bg-slate-800/50 rounded p-3">
              <div className="flex justify-between text-xs text-slate-500 mb-1">
                <span>{users.find(u => u.id === c.user_id)?.username || c.user_id.slice(0, 8) + '...'}</span>
                <span>{new Date(c.created_at).toLocaleString()}</span>
              </div>
              <p className="text-sm text-slate-300">{c.content}</p>
            </div>
          ))}
          {comments.length === 0 && <p className="text-slate-500 text-sm">No notes yet</p>}
        </div>
        <div className="flex gap-2">
          <input
            type="text"
            value={comment}
            onChange={e => setComment(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && comment.trim() && commentMut.mutate(comment.trim())}
            placeholder="Add a note..."
            className="flex-1 bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-300 placeholder-slate-500 focus:outline-none focus:border-blue-500"
          />
          <button
            onClick={() => comment.trim() && commentMut.mutate(comment.trim())}
            className="px-4 py-2 rounded text-sm bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50"
            disabled={!comment.trim()}
          >Add</button>
        </div>
      </div>

      {/* Activity log */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-4">
        <h3 className="text-sm font-medium text-slate-400 mb-3">Activity</h3>
        <div className="space-y-2">
          {activity.map(a => (
            <div key={a.id} className="flex items-start gap-3 text-xs">
              <span className="text-slate-500 whitespace-nowrap">{new Date(a.created_at).toLocaleString()}</span>
              <span className="text-slate-400">
                <span className="font-medium text-slate-300">
                  {a.changed_by === 'Automatic' ? 'Automatic' : users.find(u => u.id === a.changed_by)?.username || a.changed_by.slice(0, 8) + '...'}
                </span>
                {' '}{a.action.replace('_', ' ')}
                {a.old_value && a.new_value && (
                  <> from <span className="text-slate-300">{users.find(u => u.id === a.old_value)?.username || a.old_value}</span> to <span className="text-slate-300">{users.find(u => u.id === a.new_value)?.username || a.new_value}</span></>
                )}
                {a.note && <span className="text-slate-500 ml-1">— {a.note}</span>}
              </span>
            </div>
          ))}
          {activity.length === 0 && <p className="text-slate-500 text-sm">No activity yet</p>}
        </div>
      </div>
    </div>
  )
}
