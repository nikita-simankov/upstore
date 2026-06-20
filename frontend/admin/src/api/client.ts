const BASE = '/api/v1'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('upstore_token')
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  register: (email: string, password: string, name: string) =>
    request<{ token: string; merchant: Merchant }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name }),
    }),
  login: (email: string, password: string) =>
    request<{ token: string; merchant: Merchant }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),
  me: () => request<Merchant>('/me'),

  stores: {
    create: (data: { name: string; slug: string; logo_url?: string }) =>
      request<Store>('/stores', { method: 'POST', body: JSON.stringify(data) }),
    list: () => request<Store[]>('/stores'),
    get: (id: string) => request<Store>(`/stores/${id}`),
    update: (id: string, data: Partial<Pick<Store, 'name' | 'slug' | 'logo_url'>>) =>
      request<Store>(`/stores/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  },

  products: {
    create: (storeId: string, data: ProductInput) =>
      request<Product>(`/stores/${storeId}/products`, { method: 'POST', body: JSON.stringify(data) }),
    list: (storeId: string) => request<Product[]>(`/stores/${storeId}/products`),
    get: (storeId: string, id: string) => request<Product>(`/stores/${storeId}/products/${id}`),
    update: (storeId: string, id: string, data: Partial<ProductInput>) =>
      request<Product>(`/stores/${storeId}/products/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (storeId: string, id: string) =>
      request<void>(`/stores/${storeId}/products/${id}`, { method: 'DELETE' }),
  },

  upload: {
    presign: (ext = 'jpg') =>
      request<{ upload_url: string; public_url: string; key: string }>(`/upload/presign?ext=${ext}`),
  },
}

export interface Merchant {
  id: string
  email: string
  name: string
  created_at: string
}

export interface Store {
  id: string
  merchant_id: string
  name: string
  slug: string
  logo_url: string | null
  plan: string
  trial_ends_at: string
  created_at: string
}

export interface Product {
  id: string
  store_id: string
  name: string
  description: string | null
  price: number
  stock: number
  image_url: string | null
  published: boolean
  created_at: string
}

export interface ProductInput {
  name: string
  description?: string
  price: number
  stock: number
  image_url?: string
  published: boolean
}
