import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, type Store, type StoreStats } from '../api/client'
import { useAuth } from '../hooks/useAuth'

function fmt(k: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(k / 100)
}

export default function AnalyticsPage() {
  const { storeId } = useParams<{ storeId: string }>()
  const navigate = useNavigate()
  const { clearToken } = useAuth()
  const [store, setStore] = useState<Store | null>(null)
  const [stats, setStats] = useState<StoreStats | null>(null)

  useEffect(() => {
    Promise.all([api.stores.get(storeId!), api.analytics.stats(storeId!)])
      .then(([s, st]) => { setStore(s); setStats(st) })
      .catch(() => { clearToken(); navigate('/login') })
  }, [storeId])

  const planName: Record<string, string> = { start: 'Старт', growth: 'Рост', pro: 'Про' }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link to="/" className="text-sm text-gray-500 hover:text-gray-900">← Магазины</Link>
          <span className="text-gray-300">/</span>
          <Link to={`/stores/${storeId}/products`} className="text-sm text-gray-500 hover:text-gray-900">{store?.name}</Link>
          <span className="text-gray-300">/</span>
          <span className="text-sm font-medium text-gray-900">Аналитика</span>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <h2 className="text-xl font-semibold text-gray-900 mb-8">Аналитика</h2>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
          {[
            { label: 'Выручка', value: stats ? fmt(stats.revenue) : '—' },
            { label: 'Заказы', value: stats?.order_count ?? '—' },
            { label: 'Оплачено', value: stats?.paid_count ?? '—' },
          ].map((card) => (
            <div key={card.label} className="bg-white rounded-xl border border-gray-200 p-5">
              <p className="text-sm text-gray-500">{card.label}</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">{card.value}</p>
            </div>
          ))}
        </div>

        {store && (
          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <p className="text-sm font-medium text-gray-700 mb-3">Тарифный план</p>
            <div className="flex items-center justify-between">
              <div>
                <p className="font-semibold text-gray-900">{planName[store.plan] ?? store.plan}</p>
                <p className="text-xs text-gray-400 mt-0.5">
                  Пробный период до {new Date(store.trial_ends_at).toLocaleDateString('ru-RU')}
                </p>
              </div>
              <button disabled className="text-sm border border-gray-300 text-gray-500 px-4 py-2 rounded-lg opacity-50 cursor-not-allowed">
                Сменить план — скоро
              </button>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
