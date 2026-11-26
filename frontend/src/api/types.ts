export interface Product {
  id: number
  sku: string
  name: string
  price: number
  stock: number
  is_active: boolean
  created_at: string
  last_movement_at: string | null
}

export interface Paged<T> {
  data: T[]
  page: number
  limit: number
  total: number
}

export interface Movement {
  id: number
  product_id: number
  type: 'in' | 'out'
  qty: number
  note: string | null
  created_at: string
}

export interface MovementList {
  data: Movement[]
}

export interface ProductInput {
  sku: string
  name: string
  price: number
  is_active?: boolean
}

export interface StockInput {
  type: 'in' | 'out'
  qty: number
  note?: string
}

export interface StockResult {
  product: Product
  movement: Movement
}

export interface OrderItem {
  id: number
  sku: string
  qty: number
  unit_price: number
}

export interface Order {
  id: number
  customer: string
  status: 'CREATED' | 'CANCELLED'
  total: number
  created_at: string
  cancelled_at: string | null
  items: OrderItem[]
}

export interface OrderLineInput {
  sku: string
  qty: number
}

export interface CreateOrderInput {
  customer: string
  items: OrderLineInput[]
}

export interface ProductQuery {
  search?: string
  page?: number
  limit?: number
}

export interface ApiErrorBody {
  code: string
  message: string
  details?: string[]
  available?: number
  sku?: string
}
