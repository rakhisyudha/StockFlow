import { useCallback, useEffect, useMemo, useState } from 'react'
import { ApiError, api } from '../api/client'
import type { Product } from '../api/types'
import { AdjustStockModal } from '../components/AdjustStockModal'
import { ProductTable } from '../components/ProductTable'
import { SummaryCards } from '../components/SummaryCards'
import { useDebouncedValue } from '../lib/useDebouncedValue'

type Filter = 'ALL' | 'ACTIVE' | 'INACTIVE' | 'LOW'
type LoadState = 'loading' | 'ready' | 'error'

const PAGE_SIZE = 100

export function ProductsPage() {
  const [products, setProducts] = useState<Product[]>([])
  const [state, setState] = useState<LoadState>('loading')
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<Filter>('ALL')
  const [adjusting, setAdjusting] = useState<Product | null>(null)
  const debouncedSearch = useDebouncedValue(search, 300)

  const load = useCallback(async () => {
    setState('loading')
    setError(null)
    try {
      const first = await api.listProducts({ page: 1, limit: PAGE_SIZE })
      const pageCount = Math.ceil(first.total / PAGE_SIZE)
      const pages = await Promise.all(
        Array.from({ length: Math.max(0, pageCount - 1) }, (_, index) =>
          api.listProducts({ page: index + 2, limit: PAGE_SIZE }),
        ),
      )
      setProducts([first, ...pages].flatMap((page) => page.data))
      setState('ready')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not load products')
      setState('error')
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const visible = useMemo(() => {
    const term = debouncedSearch.trim().toLowerCase()
    return products.filter((product) => {
      const matchesTerm =
        !term ||
        product.sku.toLowerCase().includes(term) ||
        product.name.toLowerCase().includes(term)
      if (!matchesTerm) return false
      if (filter === 'ACTIVE') return product.is_active
      if (filter === 'INACTIVE') return !product.is_active
      if (filter === 'LOW') return product.stock <= 10
      return true
    })
  }, [products, debouncedSearch, filter])

  return (
    <section className="page">
      <h1>Products</h1>

      <SummaryCards products={products} />

      <div className="toolbar">
        <input
          aria-label="Search products"
          placeholder="Search SKU or name"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
        <select
          aria-label="Filter products"
          value={filter}
          onChange={(event) => setFilter(event.target.value as Filter)}
        >
          <option value="ALL">All</option>
          <option value="ACTIVE">Active</option>
          <option value="INACTIVE">Inactive</option>
          <option value="LOW">Low stock (10 or less)</option>
        </select>
      </div>

      {state === 'loading' && <p className="muted">Loading products...</p>}

      {state === 'error' && (
        <div className="error">
          <p>{error}</p>
          <button type="button" onClick={() => void load()}>Retry</button>
        </div>
      )}

      {state === 'ready' && (
        visible.length === 0 ? (
          <p className="muted">No products found.</p>
        ) : (
          <ProductTable products={visible} onAdjust={setAdjusting} />
        )
      )}

      {adjusting && (
        <AdjustStockModal
          product={adjusting}
          onClose={() => setAdjusting(null)}
          onSaved={() => void load()}
        />
      )}
    </section>
  )
}
