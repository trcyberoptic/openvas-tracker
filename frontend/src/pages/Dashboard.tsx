import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, LineChart, Line, XAxis, YAxis, CartesianGrid, Legend } from 'recharts'

const COLORS: Record<string, string> = {
  critical: '#dc2626', high: '#ea580c', medium: '#d97706', low: '#2563eb', info: '#6b7280',
}

const SOURCE_COLORS: Record<string, string> = {
  openvas: '#22c55e', zap: '#3b82f6', unknown: '#6b7280',
}

interface DashboardData {
  vulns_by_severity: { severity: string; count: number }[]
  tickets_by_scan_type: { scan_type: string; count: number }[] | null
  my_tickets: number
  unassigned_tickets: number
  open_tickets_total: number
  pending_resolution_total: number
  resolved_tickets: number
}

interface TrendPoint {
  scan_id: string
  scan_name: string
  scan_date: string
  total: number
  critical: number
  high: number
  medium: number
  low: number
  pending_resolution: number
}

export function Dashboard() {
  const { data } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<DashboardData>('/dashboard'),
  })

  const { data: trend = [] } = useQuery({
    queryKey: ['dashboard-trend'],
    queryFn: () => api.get<TrendPoint[]>('/dashboard/trend'),
  })

  const chartData = data?.vulns_by_severity?.map(v => ({
    name: v.severity, value: v.count, fill: COLORS[v.severity] || '#6b7280',
  })) || []

  const totalVulns = chartData.reduce((sum, d) => sum + d.value, 0)

  const trendData = trend.map(t => ({
    ...t,
    date: new Date(t.scan_date).toLocaleDateString(),
  }))

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>

      {/* Vulnerability severity cards */}
      <div className="grid grid-cols-5 gap-4 mb-8">
        {['critical', 'high', 'medium', 'low', 'info'].map(sev => {
          const count = chartData.find(d => d.name === sev)?.value || 0
          return (
            <div key={sev} className="rounded-lg p-4 text-center text-white" style={{ background: COLORS[sev] }}>
              <div className="text-3xl font-bold">{count}</div>
              <div className="text-sm opacity-90 capitalize">{sev}</div>
            </div>
          )
        })}
      </div>

      <div className="grid grid-cols-3 gap-6 mb-8">
        <Link to="/tickets?assigned=me" className="bg-slate-900 rounded-lg border border-slate-800 p-6 hover:border-slate-600 transition-colors">
          <h2 className="text-lg font-semibold mb-4">My Tickets</h2>
          <p className="text-4xl font-bold text-blue-400">{data?.my_tickets ?? 0}</p>
          <p className="text-slate-400 text-sm mt-1">assigned to me</p>
        </Link>
        <Link to="/tickets?assigned=unassigned" className="bg-slate-900 rounded-lg border border-slate-800 p-6 hover:border-slate-600 transition-colors">
          <h2 className="text-lg font-semibold mb-4">Unassigned</h2>
          <p className="text-4xl font-bold text-yellow-400">{data?.unassigned_tickets ?? 0}</p>
          <p className="text-slate-400 text-sm mt-1">need attention</p>
        </Link>
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
          <h2 className="text-lg font-semibold mb-4">Tickets Overview</h2>
          <div className="space-y-2">
            <div className="flex justify-between"><span className="text-slate-400">Open</span><span className="font-bold">{data?.open_tickets_total ?? 0}</span></div>
            <div className="flex justify-between"><span className="text-amber-400">Pending Resolution</span><span className="font-bold text-amber-400">{data?.pending_resolution_total ?? 0}</span></div>
            <div className="flex justify-between"><span className="text-slate-400">Resolved</span><span className="font-bold text-green-400">{data?.resolved_tickets ?? 0}</span></div>
          </div>
        </div>
      </div>

      {/* Trend Chart */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6 mb-8">
        <h2 className="text-lg font-semibold mb-4">Vulnerability Trend</h2>
        {trendData.length > 1 ? (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={trendData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
              <XAxis dataKey="date" stroke="#94a3b8" tick={{ fontSize: 12 }} />
              <YAxis stroke="#94a3b8" tick={{ fontSize: 12 }} />
              <Tooltip
                contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: 8 }}
                labelStyle={{ color: '#e2e8f0' }}
              />
              <Legend />
              <Line type="monotone" dataKey="critical" stroke="#dc2626" strokeWidth={2} dot={{ r: 3 }} />
              <Line type="monotone" dataKey="high" stroke="#ea580c" strokeWidth={2} dot={{ r: 3 }} />
              <Line type="monotone" dataKey="medium" stroke="#d97706" strokeWidth={2} dot={{ r: 3 }} />
              <Line type="monotone" dataKey="low" stroke="#2563eb" strokeWidth={2} dot={{ r: 3 }} />
              <Line type="monotone" dataKey="pending_resolution" name="Pending Resolution" stroke="#d97706" strokeWidth={2} dot={{ r: 3 }} />
              <Line type="monotone" dataKey="total" stroke="#e2e8f0" strokeWidth={2} strokeDasharray="5 5" dot={{ r: 3 }} />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <p className="text-slate-500">{trendData.length === 1 ? 'Need at least 2 scans to show trend' : 'No scan data yet'}</p>
        )}
      </div>

      <div className="grid grid-cols-2 gap-6">
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
          <h2 className="text-lg font-semibold mb-4">Severity Distribution</h2>
          {totalVulns > 0 ? (
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie data={chartData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={90} label>
                  {chartData.map((entry, i) => <Cell key={i} fill={entry.fill} />)}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          ) : <p className="text-slate-500">No vulnerabilities found</p>}
        </div>
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
          <h2 className="text-lg font-semibold mb-4">Open Tickets by Source</h2>
          {(data?.tickets_by_scan_type?.length ?? 0) > 0 ? (
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie data={data!.tickets_by_scan_type!.map(s => ({ name: s.scan_type.toUpperCase(), value: s.count, fill: SOURCE_COLORS[s.scan_type] || '#6b7280' }))} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={90} label>
                  {data!.tickets_by_scan_type!.map((s, i) => <Cell key={i} fill={SOURCE_COLORS[s.scan_type] || '#6b7280'} />)}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          ) : <p className="text-slate-500">No open tickets</p>}
        </div>
      </div>
    </div>
  )
}
