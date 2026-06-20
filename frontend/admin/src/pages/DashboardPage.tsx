import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, type Merchant } from '../api/client'
import { useAuth } from '../hooks/useAuth'

export default function DashboardPage() {
  const navigate = useNavigate()
  const { clearToken } = useAuth()
  const [merchant, setMerchant] = useState<Merchant | null>(null)

  useEffect(() => {
    api.me().then(setMerchant).catch(() => {
      clearToken()
      navigate('/login')
    })
  }, [])

  function handleLogout() {
    clearToken()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
        <span className="font-bold text-gray-900">Upstore</span>
        <div className="flex items-center gap-4">
          <span className="text-sm text-gray-500">{merchant?.email}</span>
          <button
            onClick={handleLogout}
            className="text-sm text-gray-500 hover:text-gray-900 transition-colors"
          >
            Выйти
          </button>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <h2 className="text-xl font-semibold text-gray-900 mb-2">
          Добро пожаловать{merchant?.name ? `, ${merchant.name}` : ''}
        </h2>
        <p className="text-gray-500 text-sm mb-8">Ваш магазин ещё не создан. Начните с настройки магазина.</p>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          {[
            { label: 'Товары', value: '0', hint: 'Добавьте первый товар' },
            { label: 'Заказы', value: '0', hint: 'Заказов пока нет' },
            { label: 'Выручка', value: '₽0', hint: 'За всё время' },
          ].map((card) => (
            <div key={card.label} className="bg-white rounded-xl border border-gray-200 p-5">
              <p className="text-sm text-gray-500">{card.label}</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">{card.value}</p>
              <p className="text-xs text-gray-400 mt-1">{card.hint}</p>
            </div>
          ))}
        </div>

        <div className="mt-8 bg-white rounded-xl border border-gray-200 flex flex-col items-center justify-center py-16 text-center">
          <p className="text-gray-400 text-sm">Создайте магазин, чтобы начать продавать</p>
          <button
            disabled
            className="mt-4 bg-gray-900 text-white px-5 py-2 rounded-lg text-sm font-medium opacity-50 cursor-not-allowed"
          >
            Создать магазин — скоро
          </button>
        </div>
      </main>
    </div>
  )
}
