import { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Search, X, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react'

// --- Filter ---

interface SelectOption {
  value: string
  label: string
}

interface FilterOption {
  label: string
  key: string
  options?: string[] | SelectOption[]
  searchable?: boolean
}

function Combobox({ placeholder, options, value, onChange }: {
  placeholder: string
  options: SelectOption[]
  value: string
  onChange: (v: string) => void
}) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const selectedLabel = value ? options.find(o => o.value === value)?.label || value : ''

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const filtered = useMemo(() => {
    if (!query) return options
    const q = query.toLowerCase()
    return options.filter(o => o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q))
  }, [options, query])

  return (
    <div ref={ref} className="relative">
      <input
        type="text"
        placeholder={placeholder}
        value={open ? query : selectedLabel}
        onChange={e => { setQuery(e.target.value); setOpen(true); if (!e.target.value) onChange('') }}
        onFocus={() => { setOpen(true); setQuery('') }}
        className="bg-slate-800 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-300 placeholder-slate-500 focus:outline-none focus:border-blue-500 w-52"
      />
      {value && !open && (
        <button onClick={() => { onChange(''); setQuery('') }} className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-white">
          <X size={14} />
        </button>
      )}
      {open && filtered.length > 0 && (
        <div className="absolute z-50 mt-1 w-full max-h-60 overflow-auto bg-slate-800 border border-slate-700 rounded shadow-lg">
          {filtered.map(o => (
            <button
              key={o.value}
              onClick={() => { onChange(o.value); setOpen(false); setQuery('') }}
              className={`block w-full text-left px-3 py-1.5 text-sm hover:bg-slate-700 ${o.value === value ? 'text-blue-400' : 'text-slate-300'}`}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

interface FilterProps {
  filters: FilterOption[]
  values: Record<string, string>
  onChange: (values: Record<string, string>) => void
}

export function TableFilter({ filters, values, onChange }: FilterProps) {
  const set = (key: string, val: string) => onChange({ ...values, [key]: val })
  const hasActive = Object.values(values).some(v => v !== '')

  return (
    <div className="flex items-center gap-3 mb-4 flex-wrap">
      <Search size={16} className="text-slate-500" />
      {filters.map(f => f.searchable && f.options ? (
        <Combobox
          key={f.key}
          placeholder={f.label}
          options={f.options.map(o => typeof o === 'string' ? { value: o, label: o } : o)}
          value={values[f.key] || ''}
          onChange={v => set(f.key, v)}
        />
      ) : f.options ? (
        <select
          key={f.key}
          value={values[f.key] || ''}
          onChange={e => set(f.key, e.target.value)}
          className="bg-slate-800 border border-slate-700 rounded px-2 py-1.5 text-sm text-slate-300 focus:outline-none focus:border-blue-500"
        >
          <option value="">{f.label}</option>
          {f.options.map(o => {
            const val = typeof o === 'string' ? o : o.value
            const lbl = typeof o === 'string' ? o : o.label
            return <option key={val} value={val}>{lbl}</option>
          })}
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

export function useTableFilter(keys: string[], defaults?: Record<string, string>) {
  const [searchParams, setSearchParams] = useSearchParams()

  // Fresh navigation (no filter keys in URL) → apply defaults
  const hasAnyFilterParam = keys.some(k => searchParams.has(k))

  const values = useMemo(() =>
    Object.fromEntries(keys.map(k => {
      if (searchParams.has(k)) return [k, searchParams.get(k)!]
      if (!hasAnyFilterParam && defaults?.[k]) return [k, defaults[k]]
      return [k, '']
    })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [searchParams]
  )

  const setValues = useCallback((next: Record<string, string>) => {
    setSearchParams(prev => {
      const updated = new URLSearchParams(prev)
      for (const k of keys) {
        const v = next[k]
        if (v) {
          updated.set(k, v)
        } else {
          updated.delete(k)
        }
      }
      return updated
    }, { replace: true })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [setSearchParams])

  return { values, setValues }
}

// --- Sorting ---

export type SortDir = 'asc' | 'desc' | null

export interface SortState {
  key: string
  dir: SortDir
}

export function useSortable() {
  const [sort, setSort] = useState<SortState>({ key: '', dir: null })

  const toggle = useCallback((key: string) => {
    setSort(prev => {
      if (prev.key !== key) return { key, dir: 'asc' }
      if (prev.dir === 'asc') return { key, dir: 'desc' }
      return { key: '', dir: null }
    })
  }, [])

  return { sort, toggle }
}

interface SortHeaderProps {
  label: string
  sortKey: string
  sort: SortState
  onToggle: (key: string) => void
  className?: string
}

export function SortHeader({ label, sortKey, sort, onToggle, className = '' }: SortHeaderProps) {
  const active = sort.key === sortKey
  return (
    <th
      className={`text-left p-3 text-slate-400 cursor-pointer hover:text-white select-none ${className}`}
      onClick={() => onToggle(sortKey)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {active && sort.dir === 'asc' && <ArrowUp size={14} />}
        {active && sort.dir === 'desc' && <ArrowDown size={14} />}
        {!active && <ArrowUpDown size={14} className="opacity-30" />}
      </span>
    </th>
  )
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useSorted<T extends Record<string, any>>(items: T[], sort: SortState): T[] {
  return useMemo(() => {
    if (!sort.key || !sort.dir) return items
    const k = sort.key
    const dir = sort.dir === 'asc' ? 1 : -1
    return [...items].sort((a, b) => {
      const av = a[k], bv = b[k]
      if (av == null && bv == null) return 0
      if (av == null) return 1
      if (bv == null) return -1
      if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * dir
      return String(av).localeCompare(String(bv)) * dir
    })
  }, [items, sort])
}
