import { Link, useLocation } from 'react-router-dom'
import { LayoutDashboard, Scan, GitCompare, Ticket, User, ShieldCheck, Settings } from 'lucide-react'

const links = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/tickets?assigned=me', icon: User, label: 'My Tickets' },
  { to: '/tickets', icon: Ticket, label: 'All Tickets' },
  { to: '/scans', icon: Scan, label: 'Scans' },
  { to: '/scans/diff', icon: GitCompare, label: 'Scan Diff' },
  { to: '/risk-rules', icon: ShieldCheck, label: 'Auto-Accept Rules' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export function Sidebar() {
  const location = useLocation()
  const current = location.pathname + location.search

  return (
    <aside className="w-64 bg-slate-900 border-r border-slate-800 min-h-screen p-4">
      <div className="text-xl font-bold text-white mb-8 px-2">OpenVAS-Tracker</div>
      <nav className="space-y-1">
        {links.map(({ to, icon: Icon, label }) => {
          let active: boolean
          if (to.includes('?')) {
            // Query-param link: exact match only
            active = current === to
          } else if (to === '/') {
            active = location.pathname === '/'
          } else {
            // Path-only link: match pathname but NOT if a query-param sibling matches
            active = location.pathname === to && !location.search
          }

          return (
            <Link key={to} to={to}
              className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm ${
                active ? 'bg-blue-600 text-white' : 'text-slate-400 hover:bg-slate-800 hover:text-white'
              }`}>
              <Icon size={18} />
              {label}
            </Link>
          )
        })}
      </nav>
    </aside>
  )
}
