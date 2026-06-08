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

// Whole calendar days between the given date and today, in the viewer's local
// timezone. Using calendar days (not elapsed hours) keeps the freshness colour
// consistent with the displayed date: two feeds shown on the same day always
// get the same colour, regardless of their time-of-day.
function calendarDaysAgo(iso: string): number {
  const d = new Date(iso)
  const then = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  return Math.round((today - then) / 86400000)
}

// Thresholds (by version-date age in calendar days): <=3 fresh, <=10 aging, else stale.
export function feedFreshness(versionDate: string | null): { tier: FreshnessTier; label: string; color: string } {
  if (!versionDate) return { tier: 'unknown', label: 'unbekannt', color: '#6b7280' }
  const ageDays = calendarDaysAgo(versionDate)
  if (ageDays <= 3) return { tier: 'fresh', label: 'aktuell', color: '#22c55e' }
  if (ageDays <= 10) return { tier: 'aging', label: 'etwas alt', color: '#d97706' }
  return { tier: 'stale', label: 'veraltet', color: '#dc2626' }
}

export function relativeAge(iso: string): string {
  const days = calendarDaysAgo(iso)
  if (days <= 0) return 'heute'
  if (days === 1) return 'gestern'
  return `vor ${days} Tagen`
}
