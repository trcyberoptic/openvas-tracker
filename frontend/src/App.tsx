import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Shell } from '@/components/layout/Shell'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'
import { Targets } from '@/pages/Targets'
import { Scans } from '@/pages/Scans'
import { ScanDetail } from '@/pages/ScanDetail'
import { ScanDiff } from '@/pages/ScanDiff'
import { Vulnerabilities } from '@/pages/Vulnerabilities'
import { Tickets } from '@/pages/Tickets'
import { TicketDetail } from '@/pages/TicketDetail'
import { Reports } from '@/pages/Reports'
import { Teams } from '@/pages/Teams'
import { Settings } from '@/pages/Settings'
import { RiskRules } from '@/pages/RiskRules'
import { NotFound } from '@/pages/NotFound'
import { useAuth } from '@/hooks/useAuth'

const queryClient = new QueryClient()

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuth()
  if (!token) return <Navigate to="/login" />
  return <>{children}</>
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<ProtectedRoute><Shell /></ProtectedRoute>}>
            <Route index element={<Dashboard />} />
            <Route path="hosts" element={<Targets />} />
            <Route path="scans" element={<Scans />} />
            <Route path="scans/:id" element={<ScanDetail />} />
            <Route path="scans/diff" element={<ScanDiff />} />
            <Route path="vulnerabilities" element={<Vulnerabilities />} />
            <Route path="tickets" element={<Tickets />} />
            <Route path="tickets/:id" element={<TicketDetail />} />
            <Route path="reports" element={<Reports />} />
            <Route path="teams" element={<Teams />} />
            <Route path="risk-rules" element={<RiskRules />} />
            <Route path="settings" element={<Settings />} />
          </Route>
          <Route path="*" element={<NotFound />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
