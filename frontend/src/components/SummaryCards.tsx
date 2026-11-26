import type { Product } from '../api/types'
import { formatPrice } from '../lib/format'

interface SummaryCardsProps {
  products: Product[]
}

export function SummaryCards({ products }: SummaryCardsProps) {
  const total = products.length
  const active = products.filter((product) => product.is_active).length
  const value = products.reduce((sum, product) => sum + product.price * product.stock, 0)
  const low = products.filter((product) => product.stock <= 10).length

  return (
    <div className="summary-grid">
      <div className="summary-card">
        <span>Total products</span>
        <strong>{total}</strong>
      </div>
      <div className="summary-card">
        <span>Active products</span>
        <strong>{active}</strong>
      </div>
      <div className="summary-card">
        <span>Stock value</span>
        <strong>{formatPrice(value)}</strong>
      </div>
      <div className="summary-card">
        <span>Low stock</span>
        <strong>{low}</strong>
      </div>
    </div>
  )
}
