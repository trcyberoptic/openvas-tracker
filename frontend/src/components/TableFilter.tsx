import { useState, useMemo, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Search, X, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react'

// --- Filter ---

interface FilterOption {
  label: string
  key: string
  options?: string[]
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
