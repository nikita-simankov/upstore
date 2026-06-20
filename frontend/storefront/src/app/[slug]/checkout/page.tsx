'use client'
import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useCart } from '../../../context/CartContext'

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

function fmt(k: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(k / 100)
}

export default function CheckoutPage() {
  const params = useParams<{ slug: string }>()
  const router = useRouter()
  const { items, total, clearCart } = useCart()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [address, setAddress] = useState('')
  const [provider, setProvider] = useState<'yukassa' | 'sbp'>('yukassa')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  if (items.length === 0) {
    return (
      <main className="max-w-md mx-auto px-6 py-20 text-center">
        <p className="text-gray-400 text-sm">Корзина пуста</p>
        <a href={`/${params.slug}`} className="mt-4 inline-block text-sm text-gray-900 underline">Вернуться в магазин</a>
      </main>
    )
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch(`${API}/api/v1/public/stores/${params.slug}/checkout`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          customer_name: name,
          customer_email: email,
          customer_phone: phone || undefined,
          shipping_address: address,
          payment_provider: provider,
          items: items.map((i) => ({ product_id: i.productId, quantity: i.quantity })),
        }),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      clearCart()
      if (data.payment_url) {
        window.location.href = data.payment_url
      } else {
        router.push(`/${params.slug}/order/${data.order_id}`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка оформления заказа')
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="max-w-lg mx-auto px-6 py-10">
      <a href={`/${params.slug}`} className="text-sm text-gray-500 hover:text-gray-900 block mb-8">← Назад</a>
      <h1 className="text-2xl font-bold text-gray-900 mb-8">Оформление заказа</h1>

      <div className="bg-gray-50 rounded-xl p-4 mb-6 space-y-2">
        {items.map((item) => (
          <div key={item.productId} className="flex items-center justify-between text-sm">
            <span className="text-gray-700">{item.name} × {item.quantity}</span>
            <span className="text-gray-900 font-medium">{fmt(item.price * item.quantity)}</span>
          </div>
        ))}
        <div className="border-t border-gray-200 pt-2 flex items-center justify-between text-sm font-semibold">
          <span>Итого</span>
          <span>{fmt(total)}</span>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Имя *</label>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} required
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Email *</label>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Телефон</label>
          <input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Адрес доставки *</label>
          <textarea value={address} onChange={(e) => setAddress(e.target.value)} required rows={3}
            placeholder="г. Москва, ул. Ленина, д. 1, кв. 1"
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 resize-none" />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Способ оплаты</label>
          <div className="grid grid-cols-2 gap-2">
            {(['yukassa', 'sbp'] as const).map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setProvider(p)}
                className={`py-2.5 rounded-lg text-sm border transition-colors ${
                  provider === p ? 'border-gray-900 bg-gray-900 text-white' : 'border-gray-300 text-gray-700 hover:bg-gray-50'
                }`}
              >
                {p === 'yukassa' ? 'ЮКасса' : 'СБП'}
              </button>
            ))}
          </div>
        </div>

        {error && <p className="text-sm text-red-600 bg-red-50 px-3 py-2 rounded-lg">{error}</p>}

        <button type="submit" disabled={loading}
          className="w-full bg-gray-900 text-white py-3 rounded-xl font-medium hover:bg-gray-800 disabled:opacity-50 transition-colors">
          {loading ? 'Оформление...' : `Оплатить ${fmt(total)}`}
        </button>
      </form>
    </main>
  )
}
