import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiError, api } from '../api/client'
import type { Order } from '../api/types'
import { formatPrice, formatRelativeTime } from '../lib/format'

type LoadState = 'loading' | 'ready' | 'error'

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  const orderId = Number(id)
  const [order, setOrder] = useState<Order | null>(null)
  const [state, setState] = useState<LoadState>('loading')
  const [message, setMessage] = useState<string | null>(null)
  const [cancelling, setCancelling] = useState(false)

  const load = useCallback(async () => {
    setState('loading')
    setMessage(null)
    if (!Number.isInteger(orderId) || orderId < 1) {
      setMessage('Invalid order id')
      setState('error')
      return
    }
    try {
      setOrder(await api.getOrder(orderId))
      setState('ready')
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : 'Could not load order')
      setState('error')
    }
  }, [orderId])

  useEffect(() => {
    void load()
  }, [load])

  async function cancel() {
    if (!order || cancelling) return
    if (!window.confirm('Cancel this order? Stock will be restored.')) return

    setCancelling(true)
    setMessage(null)
    try {
      setOrder(await api.cancelOrder(order.id))
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : 'Could not cancel order')
    } finally {
      setCancelling(false)
    }
  }

  if (state === 'loading') return <p className="muted">Loading order...</p>

  if (state === 'error') {
    return (
      <div className="error">
        <p>{message}</p>
        <button type="button" onClick={() => void load()}>Retry</button>
      </div>
    )
  }

  if (!order) return <p className="muted">No order to show.</p>

  return (
    <section className="page">
      <h1>Order {order.id}</h1>

      <div className="order-meta">
        <div><span>Customer</span><strong>{order.customer}</strong></div>
        <div>
          <span>Status</span>
          <strong className={order.status === 'CANCELLED' ? 'status-cancelled badge' : 'badge badge-active'}>
            {order.status}
          </strong>
        </div>
        <div><span>Total</span><strong>{formatPrice(order.total)}</strong></div>
        <div><span>Created</span><strong>{formatRelativeTime(order.created_at)}</strong></div>
      </div>

      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>SKU</th>
              <th>Qty</th>
              <th>Unit price</th>
              <th>Line total</th>
            </tr>
          </thead>
          <tbody>
            {order.items.map((item) => (
              <tr key={item.id}>
                <td>{item.sku}</td>
                <td>{item.qty}</td>
                <td>{formatPrice(item.unit_price)}</td>
                <td>{formatPrice(item.unit_price * item.qty)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {message && <div className="error"><p>{message}</p></div>}

      <div className="detail-actions">
        {order.status === 'CREATED' && (
          <button type="button" onClick={() => void cancel()} disabled={cancelling}>
            {cancelling ? 'Cancelling...' : 'Cancel order'}
          </button>
        )}
        {order.status === 'CANCELLED' && <Link to="/">Back to products</Link>}
      </div>
    </section>
  )
}
