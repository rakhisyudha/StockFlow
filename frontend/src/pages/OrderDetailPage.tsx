import { useParams } from 'react-router-dom'

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>()
  return (
    <section className="page">
      <h1>Order {id}</h1>
      <p className="muted">Order detail and cancel arrive in part 9.</p>
    </section>
  )
}
