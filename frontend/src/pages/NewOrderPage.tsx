import type { FormEvent } from 'react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError, api } from '../api/client'
import type { Product } from '../api/types'

interface DraftLine {
  key: string
  sku: string
  qty: string
}

interface FormError {
  message: string
  details: string[]
  sku?: string
  available?: number
}

export function NewOrderPage() {
  const navigate = useNavigate()
  const [customer, setCustomer] = useState('')
  const [lines, setLines] = useState<DraftLine[]>([{ key: '0', sku: '', qty: '1' }])
  const [products, setProducts] = useState<Product[]>([])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<FormError | null>(null)

  useEffect(() => {
    api.listProducts({ page: 1, limit: 100 })
      .then((page) => setProducts(page.data))
      .catch(() => setProducts([]))
  }, [])

  function updateLine(key: string, patch: Partial<DraftLine>) {
    setLines((current) => current.map((line) => (line.key === key ? { ...line, ...patch } : line)))
  }

  function addLine() {
    setLines((current) => [...current, { key: String(Date.now()), sku: '', qty: '1' }])
  }

  function removeLine(key: string) {
    setLines((current) => current.filter((line) => line.key !== key))
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (saving) return

    setSaving(true)
    setError(null)
    try {
      const order = await api.createOrder({
        customer,
        items: lines.map((line) => ({ sku: line.sku.trim(), qty: Number(line.qty) })),
      })
      navigate(`/orders/${order.id}`)
    } catch (err) {
      if (err instanceof ApiError) {
        setError({
          message: err.message,
          details: err.details,
          sku: err.isInsufficientStock ? err.sku : undefined,
          available: err.available,
        })
      } else {
        setError({ message: 'Could not create order', details: [] })
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="page">
      <h1>Create order</h1>
      <datalist id="sku-options">
        {products.map((product) => (
          <option key={product.id} value={product.sku}>{product.name}</option>
        ))}
      </datalist>

      <form className="form-grid" onSubmit={(event) => void submit(event)}>
        <label>
          Customer
          <input value={customer} onChange={(event) => setCustomer(event.target.value)} />
        </label>

        <div className="form-lines">
          {lines.map((line) => {
            const failed = error?.sku !== undefined && error.sku === line.sku.trim()
            return (
              <div key={line.key} className={failed ? 'form-row row-error' : 'form-row'}>
                <label>
                  SKU
                  <input
                    list="sku-options"
                    value={line.sku}
                    onChange={(event) => updateLine(line.key, { sku: event.target.value })}
                  />
                </label>
                <label>
                  Qty
                  <input
                    inputMode="numeric"
                    value={line.qty}
                    onChange={(event) => updateLine(line.key, { qty: event.target.value })}
                  />
                </label>
                <button
                  type="button"
                  onClick={() => removeLine(line.key)}
                  disabled={lines.length === 1 || saving}
                >
                  Remove
                </button>
                {failed && (
                  <p className="inline-error">Insufficient stock. Available: {error?.available ?? 0}</p>
                )}
              </div>
            )
          })}
        </div>

        <div className="list-actions">
          <button type="button" onClick={addLine} disabled={saving}>Add item</button>
        </div>

        {error && (
          <div className="error">
            <p>{error.message}</p>
            {error.details.length > 0 && (
              <ul>
                {error.details.map((detail, index) => <li key={index}>{detail}</li>)}
              </ul>
            )}
          </div>
        )}

        <div className="list-actions">
          <button type="submit" disabled={saving}>
            {saving ? 'Saving...' : 'Create order'}
          </button>
        </div>
      </form>
    </section>
  )
}
