import { useState, useMemo, useCallback } from 'react'
import { useSearchParams } from 'react-router'

// Split out of components/TableFilter.tsx so that file only exports components
// (react-refresh/only-export-components — mixed exports break Vite fast refresh).

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
