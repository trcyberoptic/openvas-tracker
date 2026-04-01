import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { api } from '@/api/client'
import { Copy, Check, RefreshCw } from 'lucide-react'

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  const copy = () => { navigator.clipboard.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 2000) }
  return (
    <button onClick={copy} className="text-slate-500 hover:text-white ml-2 inline-flex items-center">
      {copied ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}
    </button>
  )
}

function CodeBlock({ label, value }: { label: string; value: string }) {
  return (
    <div className="mb-4">
      <div className="text-xs text-slate-500 mb-1">{label}</div>
      <div className="bg-slate-950 border border-slate-800 rounded p-3 font-mono text-sm text-slate-300 flex items-start justify-between">
        <pre className="whitespace-pre-wrap break-all">{value}</pre>
        <CopyButton text={value} />
      </div>
    </div>
  )
}

interface SetupInfo { api_key_masked: string; server_port: number; webhook_url: string; curl_example: string; ldap_enabled: boolean }

function SetupGuide() {
  const { data: setup } = useQuery({ queryKey: ['setup'], queryFn: () => api.get<SetupInfo>('/settings/setup') })
  if (!setup) return <p className="text-slate-500 text-sm">Loading setup info...</p>

  return (
    <div className="bg-slate-900 rounded-lg border border-slate-800 p-6 mb-6">
      <h2 className="text-lg font-semibold mb-4">OpenVAS Alert Setup</h2>
      <p className="text-sm text-slate-400 mb-4">Configure OpenVAS to send scan results automatically.</p>
      <div className="space-y-3">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-slate-400">API Key:</span>
          <code className="bg-slate-950 px-2 py-1 rounded text-xs">{setup.api_key_masked}</code>
        </div>
        <CodeBlock label="Alert URL" value={`http://<tracker-host>:${setup.server_port}${setup.webhook_url}`} />
        <CodeBlock label="Manual Import" value={setup.curl_example} />
        <div className="flex items-center gap-2 text-sm">
          <span className="text-slate-400">LDAP:</span>
          <span className={setup.ldap_enabled ? 'text-green-400' : 'text-slate-500'}>{setup.ldap_enabled ? 'Enabled' : 'Not configured'}</span>
        </div>
      </div>
    </div>
  )
}

const ENV_FIELDS = [
  { key: 'OT_SERVER_PORT', label: 'Server Port', type: 'text' },
  { key: 'OT_DATABASE_DSN', label: 'Database DSN', type: 'password' },
  { key: 'OT_JWT_SECRET', label: 'JWT Secret', type: 'password' },
  { key: 'OT_JWT_EXPIREHOURS', label: 'JWT Expire (hours)', type: 'text' },
  { key: 'OT_IMPORT_APIKEY', label: 'Import API Key', type: 'password' },
  { key: 'OT_ADMIN_PASSWORD', label: 'Admin Password', type: 'password' },
  { key: 'OT_AUTORESOLVE_THRESHOLD', label: 'Auto-Resolve Threshold', type: 'text', placeholder: '3 (consecutive scans without finding before auto-resolve)' },
  { key: 'OT_BUGREPORT_URL', label: 'Bug Report Widget URL', type: 'text', placeholder: 'URL zum Bug-Report Service (leer = deaktiviert)' },
  { key: 'OT_LDAP_URL', label: 'LDAP URL', type: 'text', placeholder: 'ldaps://dc01.example.com:636' },
  { key: 'OT_LDAP_BASE_DN', label: 'LDAP Base DN', type: 'text', placeholder: 'DC=example,DC=com' },
  { key: 'OT_LDAP_BIND_DN', label: 'LDAP Bind DN', type: 'text', placeholder: 'CN=svc-openvas,OU=Service,DC=example,DC=com' },
  { key: 'OT_LDAP_BIND_PASSWORD', label: 'LDAP Bind Password', type: 'password' },
  { key: 'OT_LDAP_GROUP_DN', label: 'LDAP Group DN', type: 'text', placeholder: 'CN=SEC-VulnMgmt,OU=Groups,DC=example,DC=com' },
  { key: 'OT_LDAP_USER_FILTER', label: 'LDAP User Filter', type: 'text', placeholder: '(sAMAccountName=%s)' },
]

function EnvConfig() {
  const qc = useQueryClient()
  const { data: envData } = useQuery({ queryKey: ['env-config'], queryFn: () => api.get<Record<string, string>>('/settings/env') })
  const [values, setValues] = useState<Record<string, string>>({})
  const [dirty, setDirty] = useState<Set<string>>(new Set())
  const [showFields, setShowFields] = useState<Set<string>>(new Set())

  useEffect(() => { if (envData) setValues(envData) }, [envData])

  const saveMut = useMutation({
    mutationFn: (pairs: Record<string, string>) => api.put('/settings/env/batch', { values: pairs }),
    onSuccess: () => { setDirty(new Set()); qc.invalidateQueries({ queryKey: ['env-config'] }) },
  })

  const testLdapMut = useMutation({
    mutationFn: () => api.post<{ status: string; error?: string; group_members?: number }>('/settings/ldap/test', {}),
  })

  const onChange = (key: string, val: string) => {
    setValues(prev => ({ ...prev, [key]: val }))
    setDirty(prev => new Set(prev).add(key))
  }

  const save = () => {
    const changed: Record<string, string> = {}
    dirty.forEach(k => { changed[k] = values[k] || '' })
    saveMut.mutate(changed)
  }

  const toggleShow = (key: string) => {
    setShowFields(prev => { const n = new Set(prev); n.has(key) ? n.delete(key) : n.add(key); return n })
  }

  return (
    <div className="bg-slate-900 rounded-lg border border-slate-800 p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold">Configuration (.env)</h2>
        {dirty.size > 0 && (
          <button onClick={save} disabled={saveMut.isPending}
            className="px-4 py-1.5 rounded text-sm bg-blue-600 text-white hover:bg-blue-500 disabled:opacity-50">
            {saveMut.isPending ? 'Saving...' : `Save ${dirty.size} change${dirty.size > 1 ? 's' : ''}`}
          </button>
        )}
      </div>

      {saveMut.isSuccess && <p className="text-green-400 text-xs mb-3">Saved. Restart the service for changes to take effect.</p>}

      <div className="space-y-3">
        {ENV_FIELDS.map(f => {
          const isPassword = f.type === 'password'
          const shown = showFields.has(f.key)
          const isDirty = dirty.has(f.key)
          return (
            <div key={f.key}>
              <label className="text-xs text-slate-500 mb-1 block">{f.label} <code className="text-slate-600">{f.key}</code></label>
              <div className="flex gap-2">
                <input
                  type={isPassword && !shown ? 'password' : 'text'}
                  value={values[f.key] || ''}
                  onChange={e => onChange(f.key, e.target.value)}
                  placeholder={f.placeholder || ''}
                  className={`flex-1 bg-slate-800 border rounded px-3 py-1.5 text-sm text-slate-300 focus:outline-none focus:border-blue-500 ${isDirty ? 'border-blue-500' : 'border-slate-700'}`}
                />
                {isPassword && (
                  <button onClick={() => toggleShow(f.key)} className="text-slate-500 hover:text-white px-2 text-xs">
                    {shown ? 'Hide' : 'Show'}
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* LDAP Test */}
      <div className="mt-6 pt-4 border-t border-slate-800">
        <div className="flex items-center gap-3">
          <button onClick={() => testLdapMut.mutate()}
            disabled={testLdapMut.isPending}
            className="px-4 py-1.5 rounded text-sm bg-slate-700 text-slate-300 hover:bg-slate-600 disabled:opacity-50 inline-flex items-center gap-2">
            <RefreshCw size={14} className={testLdapMut.isPending ? 'animate-spin' : ''} />
            Test LDAP Connection
          </button>
          {testLdapMut.data && (
            <span className={`text-sm ${testLdapMut.data.status === 'ok' ? 'text-green-400' : testLdapMut.data.status === 'not_configured' ? 'text-slate-500' : 'text-red-400'}`}>
              {testLdapMut.data.status === 'ok' && `Connected — ${testLdapMut.data.group_members} group members`}
              {testLdapMut.data.status === 'not_configured' && 'LDAP not configured'}
              {testLdapMut.data.status === 'error' && `Error: ${testLdapMut.data.error}`}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

export function Settings() {
  const { user } = useAuth()

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold mb-6">Settings</h1>

      {/* Profile */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Profile</h2>
        <div className="space-y-3 text-sm">
          <div><span className="text-slate-400">Username:</span> <span>{user?.username}</span></div>
          <div><span className="text-slate-400">Email:</span> <span>{user?.email}</span></div>
        </div>
      </div>

      <SetupGuide />
      <EnvConfig />
    </div>
  )
}
