import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '@/api/client'
import { TableFilter, useTableFilter, SortHeader, useSortable, useSorted } from '@/components/TableFilter'

interface Scan { id: string; name: string; scan_type: string; status: string; created_at: string }

export function Scans() {
  const navigate = useNavigate()
  const { data: scans = [] } = useQuery({ queryKey: ['scans'], queryFn: () => api.get<Scan[]>('/scans') })
  const { values, setValues } = useTableFilter(['search', 'status'])
  const { sort, toggle } = useSortable()

  const filtered = useMemo(() => {
    let result = scans
    if (values.status) result = result.filter(s => s.status === values.status)
    if (values.search) { const q = values.search.toLowerCase(); result = result.filter(s => s.name.toLowerCase().includes(q) || s.scan_type.toLowerCase().includes(q) || s.status.toLowerCase().includes(q) || new Date(s.created_at).toLocaleString().toLowerCase().includes(q)) }
    return result
  }, [scans, values])

  const sorted = useSorted(filtered, sort)

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Scans (Imports)</h1>
      <TableFilter filters={[{ key: 'search', label: 'Search scans...' }, { key: 'status', label: 'Status', options: ['completed', 'running', 'failed', 'pending'] }]} values={values} onChange={setValues} />
      <div className="bg-slate-900 rounded-lg border border-slate-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-slate-800">
            <SortHeader label="Name" sortKey="name" sort={sort} onToggle={toggle} />
            <SortHeader label="Type" sortKey="scan_type" sort={sort} onToggle={toggle} />
            <SortHeader label="Status" sortKey="status" sort={sort} onToggle={toggle} />
            <SortHeader label="Date" sortKey="created_at" sort={sort} onToggle={toggle} />
          </tr></thead>
          <tbody>
            {sorted.map(s => (
              <tr key={s.id} onClick={() => navigate(`/scans/${s.id}`)} className="border-b border-slate-800/50 hover:bg-slate-800/30 cursor-pointer">
                <td className="p-3 text-blue-400">{s.name}</td>
                <td className="p-3 text-slate-400">{s.scan_type}</td>
                <td className="p-3"><span className={`px-2 py-1 rounded text-xs ${s.status === 'completed' ? 'bg-green-900 text-green-300' : s.status === 'failed' ? 'bg-red-900 text-red-300' : 'bg-slate-700 text-slate-300'}`}>{s.status}</span></td>
                <td className="p-3 text-slate-400">{new Date(s.created_at).toLocaleString()}</td>
              </tr>
            ))}
            {sorted.length === 0 && (<tr><td colSpan={4} className="p-6 text-center text-slate-500">{scans.length > 0 ? 'No matches' : 'No scans found'}</td></tr>)}
          </tbody>
        </table>
      </div>
      <p className="text-slate-500 text-xs mt-2">{filtered.length} of {scans.length} scans</p>
    </div>
  )
}
