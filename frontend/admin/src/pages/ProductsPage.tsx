import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, type Product, type Store } from '../api/client'
import { useAuth } from '../hooks/useAuth'

function formatPrice(kopecks: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(kopecks / 100)
}

export default function ProductsPage() {
  const { storeId } = useParams<{ storeId: string }>()
  const navigate = useNavigate()
  const { clearToken } = useAuth()
  const [store, setStore] = useState<Store | null>(null)
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.stores.get(storeId!), api.products.list(storeId!)])
      .then(([s, p]) => { setStore(s); setProducts(p) })
      .catch(() => { clearToken(); navigate('/login') })
      .finally(() => setLoading(false))
  }, [storeId])

  async function handleDelete(id: string) {
    if (!confirm('Удалить товар?')) return
    await api.products.delete(storeId!, id)
    setProducts((prev) => prev.filter((p) => p.id !== id))
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/" className="text-sm text-gray-500 hover:text-gray-900">← Магазины</Link>
          <span className="text-gray-300">/</span>
          <span className="text-sm font-medium text-gray-900">{store?.name}</span>
        </div>
        <div className="flex items-center gap-3">
          <Link to={`/stores/${storeId}/orders`} className="text-sm text-gray-500 hover:text-gray-900">Заказы</Link>
          <Link to={`/stores/${storeId}/analytics`} className="text-sm text-gray-500 hover:text-gray-900">Аналитика</Link>
          <button onClick={() => { clearToken(); navigate('/login') }} className="text-sm text-gray-500 hover:text-gray-900">Выйти</button>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <div className="flex items-center justify-between mb-8">
          <h2 className="text-xl font-semibold text-gray-900">Товары</h2>
          <Link to={`/stores/${storeId}/products/new`} className="bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition-colors">
            + Добавить товар
          </Link>
        </div>

        {loading ? (
          <p className="text-gray-400 text-sm text-center py-20">Загрузка...</p>
        ) : products.length === 0 ? (
          <div className="bg-white rounded-xl border border-gray-200 flex flex-col items-center justify-center py-20 text-center">
            <p className="text-gray-400 text-sm">Товаров пока нет</p>
            <Link to={`/stores/${storeId}/products/new`} className="mt-4 bg-gray-900 text-white px-5 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition-colors">
              Добавить первый товар
            </Link>
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100">
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Товар</th>
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Цена</th>
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Остаток</th>
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Статус</th>
                  <th className="px-5 py-3" />
                </tr>
              </thead>
              <tbody>
                {products.map((p) => (
                  <tr key={p.id} className="border-b border-gray-50 last:border-0">
                    <td className="px-5 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-lg bg-gray-100 overflow-hidden shrink-0">
                          {p.image_url && <img src={p.image_url} alt={p.name} className="w-full h-full object-cover" />}
                        </div>
                        <span className="text-sm font-medium text-gray-900">{p.name}</span>
                      </div>
                    </td>
                    <td className="px-5 py-4 text-sm text-gray-600">{formatPrice(p.price)}</td>
                    <td className="px-5 py-4 text-sm text-gray-600">{p.stock}</td>
                    <td className="px-5 py-4">
                      <span className={`text-xs px-2 py-1 rounded-full ${p.published ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                        {p.published ? 'Опубликован' : 'Черновик'}
                      </span>
                    </td>
                    <td className="px-5 py-4 text-right">
                      <div className="flex items-center justify-end gap-3">
                        <Link to={`/stores/${storeId}/products/${p.id}/edit`} className="text-xs text-gray-500 hover:text-gray-900">Изменить</Link>
                        <button onClick={() => handleDelete(p.id)} className="text-xs text-red-500 hover:text-red-700">Удалить</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  )
}
