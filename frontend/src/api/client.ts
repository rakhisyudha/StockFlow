import type {
  ApiErrorBody,
  CreateOrderInput,
  MovementList,
  Order,
  Paged,
  Product,
  ProductInput,
  ProductQuery,
  StockInput,
  StockResult,
} from './types'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details: string[]
  readonly available?: number
  readonly sku?: string

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.name = 'ApiError'
    this.status = status
    this.code = body.code
    this.details = body.details ?? []
    this.available = body.available
    this.sku = body.sku
  }

  get isInsufficientStock(): boolean {
    return this.code === 'INSUFFICIENT_STOCK'
  }

  get isValidation(): boolean {
    return this.status === 400
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return typeof value === 'object' && value !== null
    ? (value as Record<string, unknown>)
    : null
}

function parseErrorBody(payload: unknown): ApiErrorBody {
  const fallback: ApiErrorBody = { code: 'UNKNOWN_ERROR', message: 'Request failed' }
  const envelope = asRecord(payload)
  const raw = envelope?.error
  const body = asRecord(raw)
  if (!body) return fallback

  const details = Array.isArray(body.details)
    ? body.details.filter((item): item is string => typeof item === 'string')
    : undefined

  return {
    code: typeof body.code === 'string' ? body.code : fallback.code,
    message: typeof body.message === 'string' ? body.message : fallback.message,
    ...(details && details.length > 0 ? { details } : {}),
    ...(typeof body.available === 'number' ? { available: body.available } : {}),
    ...(typeof body.sku === 'string' ? { sku: body.sku } : {}),
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined) headers.set('Content-Type', 'application/json')

  let response: Response
  try {
    response = await fetch(`/api${path}`, { ...init, headers })
  } catch {
    throw new ApiError(0, { code: 'NETWORK_ERROR', message: 'Cannot reach the server' })
  }

  if (response.status === 204) return undefined as T

  const payload: unknown = await response.json().catch(() => null)
  if (!response.ok) throw new ApiError(response.status, parseErrorBody(payload))
  return payload as T
}

function productQuery(params: ProductQuery): string {
  const search = new URLSearchParams()
  if (params.search) search.set('search', params.search)
  if (params.page !== undefined) search.set('page', String(params.page))
  if (params.limit !== undefined) search.set('limit', String(params.limit))
  const encoded = search.toString()
  return encoded ? `?${encoded}` : ''
}

function jsonBody(value: unknown): string {
  return JSON.stringify(value)
}

export const api = {
  listProducts: (params: ProductQuery = {}): Promise<Paged<Product>> =>
    request(`/products${productQuery(params)}`),

  createProduct: (input: ProductInput): Promise<Product> =>
    request('/products', { method: 'POST', body: jsonBody(input) }),

  updateProduct: (id: number, input: ProductInput): Promise<Product> =>
    request(`/products/${id}`, { method: 'PUT', body: jsonBody(input) }),

  deleteProduct: (id: number): Promise<void> =>
    request(`/products/${id}`, { method: 'DELETE' }),

  adjustStock: (id: number, input: StockInput): Promise<StockResult> =>
    request(`/products/${id}/stock`, { method: 'POST', body: jsonBody(input) }),

  listMovements: (id: number): Promise<MovementList> =>
    request(`/products/${id}/movements`),

  createOrder: (input: CreateOrderInput): Promise<Order> =>
    request('/orders', { method: 'POST', body: jsonBody(input) }),

  getOrder: (id: number): Promise<Order> =>
    request(`/orders/${id}`),

  cancelOrder: (id: number): Promise<Order> =>
    request(`/orders/${id}/cancel`, { method: 'POST' }),
}
