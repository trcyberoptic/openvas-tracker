import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'

interface RiskRule {
  id: string; fingerprint: string; host_pattern: string
  reason: string; expires_at?: string; created_by: string; created_at: string
}

export function RiskRules() {
  const qc = useQueryClient()
  const { data: rules = [] } = useQuery({ queryKey: ['risk-rules'], queryFn: () => api.get<RiskRule[]>('/settings/risk-rules') })

  const deleteMut = useMutation({
    mutationFn: (id: string) => api.delete(`/settings/risk-rules/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['risk-rules'] }),
  })

  const formatFingerprint = (fp: string) => {
    if (fp.startsWith('title:')) return fp.slice(6)
    return fp // CVE ID
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Auto-Accept Rules</h1>
      <p className="text-slate-400 text-sm mb-6">
        Findings matching these rules are automatically set to <span className="text-yellow-300">risk accepted</span> on import.
        Rules can be created from any ticket's detail page.
      </p>

      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-800">
              <th className="text-left p-3 text-slate-400">Finding</th>
              <th className="text-left p-3 text-slate-400">Scope</th>
              <th className="text-left p-3 text-slate-400">Reason</th>
              <th className="text-left p-3 text-slate-400">Expires</th>
              <th className="text-left p-3 text-slate-400">Created</th>
              <th className="text-left p-3 text-slate-400 w-20"></th>
            </tr>
          </thead>
          <tbody>
            {rules.map(r => (
              <tr key={r.id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                <td className="p-3">
                  <div className="font-medium">{formatFingerprint(r.fingerprint)}</div>
                  {r.fingerprint.startsWith('title:')
                    ? <span className="text-xs text-slate-500">by title</span>
                    : <span className="text-xs text-blue-400">{r.fingerprint}</span>
                  }
                </td>
                <td className="p-3">
                  {r.host_pattern === '*'
                    ? <span className="px-2 py-0.5 rounded text-xs bg-yellow-900 text-yellow-300">All hosts</span>
                    : <span className="font-mono text-slate-300 text-xs">{r.host_pattern}</span>
                  }
                </td>
                <td className="p-3 text-slate-300 max-w-xs">{r.reason}</td>
                <td className="p-3 text-slate-400">
                  {r.expires_at
                    ? <span className={new Date(r.expires_at) < new Date() ? 'text-red-400' : ''}>{new Date(r.expires_at).toLocaleDateString()}</span>
                    : <span className="text-slate-500">Never</span>
                  }
                </td>
                <td className="p-3 text-slate-500 text-xs">{new Date(r.created_at).toLocaleDateString()}</td>
                <td className="p-3">
                  <button onClick={() => { if (confirm('Delete this rule? Existing tickets will not be changed.')) deleteMut.mutate(r.id) }}
                    className="text-red-400 hover:text-red-300 text-xs">Delete</button>
                </td>
              </tr>
            ))}
            {rules.length === 0 && (
              <tr><td colSpan={6} className="p-6 text-center text-slate-500">No auto-accept rules configured. Create one from any ticket's detail page.</td></tr>
            )}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{rules.length} rule{rules.length !== 1 ? 's' : ''}</p>
    </div>
  )
}
