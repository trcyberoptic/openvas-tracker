import { useState, useMemo, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

const BADGE: Record<string, string> = {
  new: 'bg-red-900 text-red-300',
  pending_fix: 'bg-amber-900 text-amber-300',
  fixed: 'bg-green-900 text-green-300',
  risk_accepted: 'bg-sky-900 text-sky-300',
  unchanged: 'bg-slate-700 text-slate-300',
}
const LABEL: Record<string, string> = {
  new: 'new',
  pending_fix: 'pending fix',
  fixed: 'fixed',
  risk_accepted: 'risk accepted',
  unchanged: 'unchanged',
}
const SEV_COLORS: Record<string, string> = {
  critical: 'bg-red-600', high: 'bg-orange-600', medium: 'bg-yellow-600', low: 'bg-blue-600', info: 'bg-gray-600',
}
const SCAN_TYPE_COLORS: Record<string, string> = {
  openvas: 'bg-green-900 text-green-300',
  zap: 'bg-blue-900 text-blue-300',
}

interface Scan { id: string; name: string; scan_type: string; created_at: string }
interface DiffEntry {
  status: string; vuln_id: string; title: string
  affected_host?: string; hostname?: string; severity: string; cvss_score?: number; cve_id?: string
}

export function ScanDiff() {
  const { data: scans = [] } = useQuery({ queryKey: ['scans'], queryFn: () => api.get<Scan[]>('/scans') })
  const [oldId, setOldId] = useState('')
  const [newId, setNewId] = useState('')
  const [filter, setFilter] = useState<string>('')

  const sortedScans = useMemo(() =>
    [...scans].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()),
    [scans])

  // Default to the two most recent scans of the same type
  useEffect(() => {
    if (sortedScans.length >= 2 && !oldId && !newId) {
      const newest = sortedScans[0]
      const prevSameType = sortedScans.find(s => s.id !== newest.id && s.scan_type === newest.scan_type)
      if (prevSameType) {
        setNewId(newest.id)
        setOldId(prevSameType.id)
      }
    }
  }, [sortedScans, oldId, newId])

  // Determine selected scans and their types
  const newScan = scans.find(s => s.id === newId)
  // Filter old scan options: same type as selected new scan, and older
  const oldScanOptions = useMemo(() => {
    if (!newScan) return sortedScans
    return sortedScans.filter(s =>
      s.scan_type === newScan.scan_type &&
      new Date(s.created_at).getTime() < new Date(newScan.created_at).getTime()
    )
  }, [sortedScans, newScan])

  // When new scan changes, auto-select the next older scan of same type
  useEffect(() => {
    if (!newScan) return
    const nextOlder = sortedScans.find(s =>
      s.id !== newScan.id &&
      s.scan_type === newScan.scan_type &&
      new Date(s.created_at).getTime() < new Date(newScan.created_at).getTime()
    )
    setOldId(nextOlder?.id || '')
  }, [newId])

  const { data: diff, isFetching } = useQuery({
    queryKey: ['scan-diff', oldId, newId],
    queryFn: () => api.get<DiffEntry[]>(`/scans/diff?old=${oldId}&new=${newId}`),
    enabled: !!oldId && !!newId && oldId !== newId,
  })

  const filtered = useMemo(() => {
    if (!diff) return []
    if (!filter) return diff
    return diff.filter(d => d.status === filter)
  }, [diff, filter])

  const counts = useMemo(() => {
    if (!diff) return { new: 0, pending_fix: 0, fixed: 0, risk_accepted: 0, unchanged: 0 }
    return {
      new: diff.filter(d => d.status === 'new').length,
      pending_fix: diff.filter(d => d.status === 'pending_fix').length,
      fixed: diff.filter(d => d.status === 'fixed').length,
      risk_accepted: diff.filter(d => d.status === 'risk_accepted').length,
      unchanged: diff.filter(d => d.status === 'unchanged').length,
    }
  }, [diff])

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Scan Comparison</h1>

      <div className="flex gap-4 mb-6">
        <div className="flex-1">
          <label className="text-xs text-slate-500 mb-1 block">Current Scan</label>
          <select value={newId} onChange={e => setNewId(e.target.value)}
            className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-blue-500">
            <option value="">Select...</option>
            {sortedScans.map(s => (
              <option key={s.id} value={s.id}>
                {s.name} ({new Date(s.created_at).toLocaleDateString()})
              </option>
            ))}
          </select>
          {newScan && (
            <span className={`inline-block mt-1 px-2 py-0.5 rounded text-xs font-medium ${SCAN_TYPE_COLORS[newScan.scan_type] || 'bg-slate-700 text-slate-300'}`}>
              {newScan.scan_type.toUpperCase()}
            </span>
          )}
        </div>
        <div className="flex-1">
          <label className="text-xs text-slate-500 mb-1 block">Previous Scan</label>
          <select value={oldId} onChange={e => setOldId(e.target.value)}
            className="w-full bg-slate-800 border border-slate-700 rounded px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-blue-500">
            <option value="">Select...</option>
            {oldScanOptions.map(s => (
              <option key={s.id} value={s.id}>
                {s.name} ({new Date(s.created_at).toLocaleDateString()})
              </option>
            ))}
          </select>
          {!newId && <p className="text-xs text-slate-500 mt-1">Select current scan first</p>}
          {newId && oldScanOptions.length === 0 && <p className="text-xs text-amber-400 mt-1">No older {newScan?.scan_type.toUpperCase()} scans available</p>}
        </div>
      </div>

      {diff && (
        <>
          <div className="flex gap-3 mb-4">
            <button onClick={() => setFilter('')} className={`px-3 py-1.5 rounded text-sm ${!filter ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
              All ({diff.length})
            </button>
            <button onClick={() => setFilter('new')} className={`px-3 py-1.5 rounded text-sm ${filter === 'new' ? 'bg-red-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
              New ({counts.new})
            </button>
            <button onClick={() => setFilter('pending_fix')} className={`px-3 py-1.5 rounded text-sm ${filter === 'pending_fix' ? 'bg-amber-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
              Pending Fix ({counts.pending_fix})
            </button>
            <button onClick={() => setFilter('fixed')} className={`px-3 py-1.5 rounded text-sm ${filter === 'fixed' ? 'bg-green-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
              Fixed ({counts.fixed})
            </button>
            <button onClick={() => setFilter('risk_accepted')} className={`px-3 py-1.5 rounded text-sm ${filter === 'risk_accepted' ? 'bg-sky-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
              Risk Accepted ({counts.risk_accepted})
            </button>
            <button onClick={() => setFilter('unchanged')} className={`px-3 py-1.5 rounded text-sm ${filter === 'unchanged' ? 'bg-slate-600 text-white' : 'bg-slate-800 text-slate-400'}`}>
              Unchanged ({counts.unchanged})
            </button>
          </div>

          <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
            <table className="w-full text-sm">
              <thead><tr className="border-b border-slate-800">
                <th className="text-left p-3 text-slate-400 w-24">Status</th>
                <th className="text-left p-3 text-slate-400">Title</th>
                <th className="text-left p-3 text-slate-400">Host</th>
                <th className="text-left p-3 text-slate-400 w-20">Severity</th>
                <th className="text-left p-3 text-slate-400 w-16">CVSS</th>
                <th className="text-left p-3 text-slate-400 w-32">CVE</th>
              </tr></thead>
              <tbody>
                {filtered.map(d => (
                  <tr key={d.vuln_id} className="border-b border-slate-800/50 hover:bg-slate-800/30">
                    <td className="p-3"><span className={`px-2 py-1 rounded text-xs ${BADGE[d.status] ?? BADGE.unchanged}`}>{LABEL[d.status] ?? d.status}</span></td>
                    <td className="p-3">{d.title}</td>
                    <td className="p-3 text-slate-400">
                      <span className="font-mono">{d.affected_host}</span>
                      {d.hostname && <span className="text-slate-500 text-xs ml-1">({d.hostname})</span>}
                    </td>
                    <td className="p-3"><span className={`px-2 py-1 rounded text-xs text-white ${SEV_COLORS[d.severity] || 'bg-gray-600'}`}>{d.severity}</span></td>
                    <td className="p-3">{d.cvss_score?.toFixed(1) ?? '—'}</td>
                    <td className="p-3 text-slate-400">{d.cve_id ? <a href={`https://nvd.nist.gov/vuln/detail/${d.cve_id}`} target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline">{d.cve_id}</a> : '—'}</td>
                  </tr>
                ))}
                {filtered.length === 0 && !isFetching && (
                  <tr><td colSpan={6} className="p-6 text-center text-slate-500">No differences found</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!oldId || !newId ? (
        <p className="text-slate-500 text-sm mt-4">Select two scans to compare.</p>
      ) : isFetching ? (
        <p className="text-slate-400 text-sm mt-4">Loading diff...</p>
      ) : null}
    </div>
  )
}
