import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { api } from '@/api/client'
import { Copy, Check, Eye, EyeOff } from 'lucide-react'

interface SetupInfo {
  api_key: string
  api_key_masked: string
  server_port: number
  webhook_url: string
  curl_example: string
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  const copy = () => {
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
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

function SetupGuide() {
  const { data: setup } = useQuery({ queryKey: ['setup'], queryFn: () => api.get<SetupInfo>('/settings/setup') })
  const [showKey, setShowKey] = useState(false)

  if (!setup) return <p className="text-slate-500 text-sm">Loading setup info...</p>

  return (
    <div className="space-y-6">
      {/* API Key */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
        <h2 className="text-lg font-semibold mb-4">API Key</h2>
        <div className="flex items-center gap-2 mb-2">
          <code className="bg-slate-950 border border-slate-800 rounded px-3 py-2 font-mono text-sm text-slate-300 flex-1 break-all">
            {showKey ? setup.api_key : setup.api_key_masked}
          </code>
          <button onClick={() => setShowKey(!showKey)} className="text-slate-500 hover:text-white p-1">
            {showKey ? <EyeOff size={16} /> : <Eye size={16} />}
          </button>
          <CopyButton text={setup.api_key} />
        </div>
        <p className="text-xs text-slate-500">Configured via <code>OT_IMPORT_APIKEY</code> environment variable.</p>
      </div>

      {/* OpenVAS Alert Setup */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
        <h2 className="text-lg font-semibold mb-4">OpenVAS Alert Setup</h2>
        <p className="text-sm text-slate-400 mb-4">
          Configure OpenVAS to automatically send scan results to this tracker when a scan completes.
        </p>

        <div className="space-y-4">
          <div>
            <h3 className="text-sm font-medium text-slate-300 mb-2">Step 1: Create Alert in GSA</h3>
            <ol className="text-sm text-slate-400 space-y-1 list-decimal list-inside">
              <li>Open <strong>Greenbone Security Assistant</strong> web UI</li>
              <li>Go to <strong>Configuration &rarr; Alerts &rarr; New Alert</strong></li>
              <li>Set <strong>Event</strong> to &ldquo;Task run status changed &rarr; Done&rdquo;</li>
              <li>Set <strong>Method</strong> to &ldquo;HTTP Get&rdquo;</li>
              <li>Set the URL below as <strong>HTTP Get URL</strong>:</li>
            </ol>
          </div>

          <CodeBlock
            label="Alert URL (contains API key as parameter)"
            value={`http://<tracker-host>:${setup.server_port}${setup.webhook_url}`}
          />

          <div>
            <h3 className="text-sm font-medium text-slate-300 mb-2">Step 2: Attach Alert to Scan Task</h3>
            <ol className="text-sm text-slate-400 space-y-1 list-decimal list-inside">
              <li>Go to <strong>Scans &rarr; Tasks</strong></li>
              <li>Edit your scan task (or create a new one)</li>
              <li>Under <strong>Alerts</strong>, select the alert you just created</li>
              <li>Save &mdash; scan results will now be imported automatically</li>
            </ol>
          </div>
        </div>
      </div>

      {/* Manual Import */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
        <h2 className="text-lg font-semibold mb-4">Manual Import via curl</h2>
        <p className="text-sm text-slate-400 mb-4">
          You can also import OpenVAS XML reports manually:
        </p>
        <CodeBlock label="curl command" value={setup.curl_example} />
        <div className="text-sm text-slate-400 mt-2">
          <p className="mb-1"><strong>Export from GSA:</strong></p>
          <ol className="list-decimal list-inside space-y-1">
            <li>Go to <strong>Scans &rarr; Reports</strong></li>
            <li>Select a completed report</li>
            <li>Click the download icon &rarr; choose <strong>XML</strong> format</li>
            <li>Use the curl command above to import the file</li>
          </ol>
        </div>
      </div>

      {/* Cron Fallback */}
      <div className="bg-slate-900 rounded-lg border border-slate-800 p-6">
        <h2 className="text-lg font-semibold mb-4">Alternative: Cron-based Import</h2>
        <p className="text-sm text-slate-400 mb-4">
          If your GVM version doesn&apos;t support HTTP alerts with report body, use SCP export + cron:
        </p>
        <CodeBlock
          label="/opt/openvas-tracker/push-report.sh"
          value={`#!/bin/bash
REPORT_DIR="/var/lib/gvm/reports"
API_URL="http://localhost:${setup.server_port}/api/import/openvas"
API_KEY="${setup.api_key}"

for f in "$REPORT_DIR"/*.xml; do
  curl -s -X POST "$API_URL" \\
    -H "X-API-Key: $API_KEY" \\
    -H "Content-Type: application/xml" \\
    --data-binary "@$f" && rm "$f"
done`}
        />
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
          <div><span className="text-slate-400">Email:</span> <span>{user?.email}</span></div>
          <div><span className="text-slate-400">Username:</span> <span>{user?.username}</span></div>
          <div><span className="text-slate-400">Role:</span> <span className="capitalize">{user?.role}</span></div>
        </div>
      </div>

      {/* Setup Guide (admin only) */}
      {user?.role === 'admin' && <SetupGuide />}
    </div>
  )
}
