import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts'

const COLORS: Record<string, string> = {
  critical: '#dc2626', high: '#ea580c', medium: '#d97706', low: '#2563eb', info: '#6b7280',
}

interface DashboardData {
  vulns_by_severity: { severity: string; count: number }[]
  my_tickets: number
  unassigned_tickets: number
  open_tickets_total: number
  resolved_tickets: number
}

export function Dashboard() {
  const { data } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api.get<DashboardData>('/dashboard'),
  })

  const chartData = data?.vulns_by_severity?.map(v => ({
    name: v.severity, value: v.count, fill: COLORS[v.severity] || '#6b7280',
  })) || []

  const totalVulns = chartData.reduce((sum, d) => sum + d.value, 0)

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
        {/* Ticket stats */}
        <Link to="/tickets" className="bg-slate-900 rounded-lg border border-slate-800 p-6 hover:border-slate-600 transition-colors">
          <h2 className="text-lg font-semibold mb-4">My Tickets</h2>
          <p className="text-4xl font-bold text-blue-400">{data?.my_tickets ?? 0}</p>
          <p className="text-slate-400 text-sm mt-1">assigned to me</p>
        </Link>
        <Link to="/tickets" className="bg-slate-900 rounded-lg border border-slate-800 p-6 hover:border-slate-600 transition-colors">
          <h2 className="text-lg font-semibold mb-4">Unassigned</h2>
          <p className="text-4xl font-bold text-yellow-400">{data?.unassigned_tickets ?? 0}</p>
          <p className="text-slate-400 text-sm mt-1">need attention</p>
        </Link>
        <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
          <h2 className="text-lg font-semibold mb-4">Tickets Overview</h2>
          <div className="space-y-2">
            <div className="flex justify-between"><span className="text-slate-400">Open</span><span className="font-bold">{data?.open_tickets_total ?? 0}</span></div>
            <div className="flex justify-between"><span className="text-slate-400">Resolved</span><span className="font-bold text-green-400">{data?.resolved_tickets ?? 0}</span></div>
          </div>
        </div>
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
          <h2 className="text-lg font-semibold mb-4">Vulnerabilities</h2>
          <p className="text-4xl font-bold">{totalVulns}</p>
          <p className="text-slate-400">Total open vulnerabilities</p>
        </div>
      </div>
    </div>
  )
}
