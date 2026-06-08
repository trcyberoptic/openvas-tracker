import { api } from '@/api/client'

export interface FeedStatus {
  feed_type: string
  feed_name: string
  version: string
  version_date: string | null
  last_seen: string
  last_changed: string
}

export function getFeeds() {
  return api.get<FeedStatus[]>('/feeds')
}

export type FreshnessTier = 'fresh' | 'aging' | 'stale' | 'unknown'

// Thresholds (by version date age): <=3d fresh, <=10d aging, else stale.
export function feedFreshness(versionDate: string | null): { tier: FreshnessTier; label: string; color: string } {
  if (!versionDate) return { tier: 'unknown', label: 'unbekannt', color: '#6b7280' }
  const ageDays = (Date.now() - new Date(versionDate).getTime()) / 86400000
  if (ageDays <= 3) return { tier: 'fresh', label: 'aktuell', color: '#22c55e' }
  if (ageDays <= 10) return { tier: 'aging', label: 'etwas alt', color: '#d97706' }
  return { tier: 'stale', label: 'veraltet', color: '#dc2626' }
}

export function relativeAge(iso: string): string {
  const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000)
  if (days <= 0) return 'heute'
  if (days === 1) return 'gestern'
  return `vor ${days} Tagen`
}
