// frontend/src/components/layout/Sidebar.tsx
import { NavLink } from 'react-router-dom'
import { LayoutDashboard, Scan, GitCompare, Ticket, User, Settings } from 'lucide-react'

const links = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/tickets?assigned=me', icon: User, label: 'My Tickets' },
  { to: '/tickets', icon: Ticket, label: 'All Tickets' },
  { to: '/scans', icon: Scan, label: 'Scans' },
  { to: '/scans/diff', icon: GitCompare, label: 'Scan Diff' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export function Sidebar() {
  return (
    <aside className="w-64 bg-slate-900 border-r border-slate-800 min-h-screen p-4">
      <div className="text-xl font-bold text-white mb-8 px-2">OpenVAS-Tracker</div>
      <nav className="space-y-1">
        {links.map(({ to, icon: Icon, label }) => (
          <NavLink key={to} to={to} end={to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2 rounded-lg text-sm ${
                isActive ? 'bg-blue-600 text-white' : 'text-slate-400 hover:bg-slate-800 hover:text-white'
              }`
            }>
            <Icon size={18} />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
