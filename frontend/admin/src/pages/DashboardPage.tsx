import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, type Merchant, type Store } from '../api/client'
import { useAuth } from '../hooks/useAuth'

export default function DashboardPage() {
  const navigate = useNavigate()
  const { clearToken } = useAuth()
  const [merchant, setMerchant] = useState<Merchant | null>(null)
  const [stores, setStores] = useState<Store[]>([])

  useEffect(() => {
    api.me().then(setMerchant).catch(() => { clearToken(); navigate('/login') })
    api.stores.list().then(setStores).catch(() => {})
  }, [])

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <span className="font-bold text-gray-900">Upstore</span>
        <div className="flex items-center gap-4">
          <span className="text-sm text-gray-500">{merchant?.email}</span>
          <button onClick={() => { clearToken(); navigate('/login') }} className="text-sm text-gray-500 hover:text-gray-900">
            Выйти
          </button>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <div className="flex items-center justify-between mb-8">
          <h2 className="text-xl font-semibold text-gray-900">
            Добро пожаловать{merchant?.name ? `, ${merchant.name}` : ''}
          </h2>
          <Link
            to="/stores/new"
            className="bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition-colors"
          >
            + Создать магазин
          </Link>
        </div>

        {stores.length === 0 ? (
          <div className="bg-white rounded-xl border border-gray-200 flex flex-col items-center justify-center py-20 text-center">
            <p className="text-gray-400 text-sm">У вас пока нет магазинов</p>
            <Link
              to="/stores/new"
              className="mt-4 bg-gray-900 text-white px-5 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition-colors"
            >
              Создать первый магазин
            </Link>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {stores.map((store) => (
              <Link
                key={store.id}
                to={`/stores/${store.id}/products`}
                className="bg-white rounded-xl border border-gray-200 p-5 hover:border-gray-400 transition-colors"
              >
                <div className="flex items-center gap-3 mb-4">
                  {store.logo_url ? (
                    <img src={store.logo_url} alt={store.name} className="w-10 h-10 rounded-full object-cover" />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-gray-100 flex items-center justify-center text-gray-500 text-sm font-bold">
                      {store.name[0].toUpperCase()}
                    </div>
                  )}
                  <div>
                    <p className="font-medium text-gray-900">{store.name}</p>
                    <p className="text-xs text-gray-400">{store.slug}</p>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-xs bg-gray-100 text-gray-600 px-2 py-1 rounded-full capitalize">{store.plan}</span>
                  <span className="text-xs text-gray-400">
                    До {new Date(store.trial_ends_at).toLocaleDateString('ru-RU')}
                  </span>
                </div>
              </Link>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
