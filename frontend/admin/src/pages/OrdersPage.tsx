import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, type Order, type Store } from '../api/client'
import { useAuth } from '../hooks/useAuth'

const statusLabels: Record<string, string> = {
  pending: 'Ожидает оплаты',
  paid: 'Оплачен',
  shipped: 'Отправлен',
  delivered: 'Доставлен',
  cancelled: 'Отменён',
}

const statusColors: Record<string, string> = {
  pending: 'bg-yellow-50 text-yellow-700',
  paid: 'bg-green-50 text-green-700',
  shipped: 'bg-blue-50 text-blue-700',
  delivered: 'bg-gray-100 text-gray-700',
  cancelled: 'bg-red-50 text-red-600',
}

function formatPrice(k: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(k / 100)
}

export default function OrdersPage() {
  const { storeId } = useParams<{ storeId: string }>()
  const navigate = useNavigate()
  const { clearToken } = useAuth()
  const [store, setStore] = useState<Store | null>(null)
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([api.stores.get(storeId!), api.orders.list(storeId!)])
      .then(([s, o]) => { setStore(s); setOrders(o) })
      .catch(() => { clearToken(); navigate('/login') })
      .finally(() => setLoading(false))
  }, [storeId])

  async function handleStatusChange(id: string, status: string) {
    await api.orders.updateStatus(storeId!, id, status)
    setOrders((prev) => prev.map((o) => o.id === id ? { ...o, status } : o))
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/" className="text-sm text-gray-500 hover:text-gray-900">← Магазины</Link>
          <span className="text-gray-300">/</span>
          <Link to={`/stores/${storeId}/products`} className="text-sm text-gray-500 hover:text-gray-900">{store?.name}</Link>
          <span className="text-gray-300">/</span>
          <span className="text-sm font-medium text-gray-900">Заказы</span>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <h2 className="text-xl font-semibold text-gray-900 mb-8">Заказы</h2>

        {loading ? (
          <p className="text-gray-400 text-sm text-center py-20">Загрузка...</p>
        ) : orders.length === 0 ? (
          <div className="bg-white rounded-xl border border-gray-200 flex items-center justify-center py-20">
            <p className="text-gray-400 text-sm">Заказов пока нет</p>
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-100">
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Заказ</th>
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Покупатель</th>
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Сумма</th>
                  <th className="text-left text-xs font-medium text-gray-500 px-5 py-3">Статус</th>
                  <th className="px-5 py-3" />
                </tr>
              </thead>
              <tbody>
                {orders.map((o) => (
                  <tr key={o.id} className="border-b border-gray-50 last:border-0">
                    <td className="px-5 py-4">
                      <p className="text-sm font-mono text-gray-600">#{o.id.slice(0, 8).toUpperCase()}</p>
                      <p className="text-xs text-gray-400">{new Date(o.created_at).toLocaleDateString('ru-RU')}</p>
                    </td>
                    <td className="px-5 py-4">
                      <p className="text-sm text-gray-900">{o.customer_name}</p>
                      <p className="text-xs text-gray-400">{o.customer_email}</p>
                    </td>
                    <td className="px-5 py-4 text-sm font-medium text-gray-900">{formatPrice(o.total)}</td>
                    <td className="px-5 py-4">
                      <span className={`text-xs px-2 py-1 rounded-full ${statusColors[o.status] ?? 'bg-gray-100 text-gray-500'}`}>
                        {statusLabels[o.status] ?? o.status}
                      </span>
                    </td>
                    <td className="px-5 py-4">
                      <select
                        value={o.status}
                        onChange={(e) => handleStatusChange(o.id, e.target.value)}
                        className="text-xs border border-gray-300 rounded px-2 py-1 text-gray-600"
                      >
                        {Object.entries(statusLabels).map(([v, l]) => (
                          <option key={v} value={v}>{l}</option>
                        ))}
                      </select>
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
