export function formatPrice(value: number): string {
  return `Rp ${value.toLocaleString('id-ID')}`
}

export function formatRelativeTime(iso: string | null): string {
  if (!iso) return '—'
  const elapsed = Date.now() - new Date(iso).getTime()
  if (!Number.isFinite(elapsed) || elapsed < 0) return '—'

  const minutes = Math.floor(elapsed / 60_000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`

  return `${Math.floor(hours / 24)}d ago`
}
