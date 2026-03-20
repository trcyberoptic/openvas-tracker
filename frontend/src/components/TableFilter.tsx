import { useState } from 'react'
import { Search, X } from 'lucide-react'

interface FilterOption {
  label: string
  key: string
  options?: string[]  // dropdown options; if omitted, treated as text search
}

interface Props {
  filters: FilterOption[]
  values: Record<string, string>
  onChange: (values: Record<string, string>) => void
}

export function TableFilter({ filters, values, onChange }: Props) {
  const set = (key: string, val: string) => onChange({ ...values, [key]: val })
  const hasActive = Object.values(values).some(v => v !== '')

  return (
    <div className="flex items-center gap-3 mb-4 flex-wrap">
      <Search size={16} className="text-slate-500" />
      {filters.map(f => f.options ? (
        <select
          key={f.key}
          value={values[f.key] || ''}
          onChange={e => set(f.key, e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-2 py-1.5 text-sm text-slate-300 focus:outline-none focus:border-blue-500"
        >
          <option value="">{f.label}</option>
          {f.options.map(o => <option key={o} value={o}>{o}</option>)}
        </select>
      ) : (
        <input
          key={f.key}
          type="text"
          placeholder={f.label}
          value={values[f.key] || ''}
          onChange={e => set(f.key, e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-300 placeholder-slate-500 focus:outline-none focus:border-blue-500 w-48"
        />
      ))}
      {hasActive && (
        <button onClick={() => onChange(Object.fromEntries(filters.map(f => [f.key, ''])))} className="text-slate-500 hover:text-white">
          <X size={16} />
        </button>
      )}
    </div>
  )
}

export function useTableFilter(keys: string[]) {
  const [values, setValues] = useState<Record<string, string>>(
    Object.fromEntries(keys.map(k => [k, '']))
  )
  return { values, setValues }
}
