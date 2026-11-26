import type { FormEvent } from 'react'
import { useState } from 'react'
import { ApiError, api } from '../api/client'
import type { Product, StockInput } from '../api/types'

interface AdjustStockModalProps {
  product: Product
  onClose: () => void
  onSaved: () => void
}

export function AdjustStockModal({ product, onClose, onSaved }: AdjustStockModalProps) {
  const [type, setType] = useState<StockInput['type']>('in')
  const [qty, setQty] = useState('')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const parsedQty = Number(qty)
  const qtyValid = qty.trim() !== '' && Number.isInteger(parsedQty) && parsedQty > 0

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!qtyValid || saving) return

    setSaving(true)
    setError(null)
    try {
      await api.adjustStock(product.id, {
        type,
        qty: parsedQty,
        note: note.trim() || undefined,
      })
      onSaved()
      onClose()
    } catch (err) {
      if (err instanceof ApiError && err.isInsufficientStock) {
        setError(`Insufficient stock. Available: ${err.available ?? 0}`)
      } else if (err instanceof ApiError) {
        setError([err.message, ...err.details].join(' '))
      } else {
        setError('Could not save the adjustment')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-backdrop">
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="adjust-title">
        <h2 id="adjust-title">Adjust stock: {product.sku}</h2>
        <form onSubmit={(event) => void submit(event)}>
          <label>
            Direction
            <select value={type} onChange={(event) => setType(event.target.value as StockInput['type'])}>
              <option value="in">In</option>
              <option value="out">Out</option>
            </select>
          </label>
          <label>
            Quantity
            <input value={qty} onChange={(event) => setQty(event.target.value)} inputMode="numeric" />
          </label>
          <label>
            Note
            <input value={note} onChange={(event) => setNote(event.target.value)} />
          </label>
          {error && <p className="error">{error}</p>}
          <div className="modal-actions">
            <button type="button" onClick={onClose} disabled={saving}>Cancel</button>
            <button type="submit" disabled={saving || !qtyValid}>
              {saving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
