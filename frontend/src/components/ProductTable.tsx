import type { Product } from '../api/types'
import { formatPrice, formatRelativeTime } from '../lib/format'

interface ProductTableProps {
  products: Product[]
  onAdjust: (product: Product) => void
}

export function ProductTable({ products, onAdjust }: ProductTableProps) {
  return (
    <div className="table-wrap">
      <table className="data-table">
        <thead>
          <tr>
            <th>SKU</th>
            <th>Name</th>
            <th>Price</th>
            <th>Stock</th>
            <th>Status</th>
            <th>Last movement</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {products.map((product) => (
            <tr key={product.id}>
              <td>{product.sku}</td>
              <td>{product.name}</td>
              <td>{formatPrice(product.price)}</td>
              <td className={product.stock <= 10 ? 'stock-low' : undefined}>{product.stock}</td>
              <td>
                <span className={product.is_active ? 'badge badge-active' : 'badge badge-inactive'}>
                  {product.is_active ? 'Active' : 'Inactive'}
                </span>
              </td>
              <td>{formatRelativeTime(product.last_movement_at)}</td>
              <td>
                <button type="button" onClick={() => onAdjust(product)}>Adjust</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
