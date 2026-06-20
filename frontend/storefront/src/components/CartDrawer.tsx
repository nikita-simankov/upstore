'use client'
import { useRouter } from 'next/navigation'
import { useCart } from '../context/CartContext'

function formatPrice(k: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(k / 100)
}

export default function CartDrawer({ slug, onClose }: { slug: string; onClose: () => void }) {
  const { items, removeItem, updateQty, total } = useCart()
  const router = useRouter()

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative bg-white w-full max-w-sm flex flex-col shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-200">
          <h2 className="font-semibold text-gray-900">Корзина</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
        </div>

        {items.length === 0 ? (
          <div className="flex-1 flex items-center justify-center">
            <p className="text-gray-400 text-sm">Корзина пуста</p>
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto px-5 py-4 space-y-4">
            {items.map((item) => (
              <div key={item.productId} className="flex items-center gap-3">
                <div className="w-14 h-14 rounded-lg bg-gray-100 overflow-hidden shrink-0">
                  {item.imageUrl && <img src={item.imageUrl} alt={item.name} className="w-full h-full object-cover" />}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">{item.name}</p>
                  <p className="text-sm text-gray-500">{formatPrice(item.price)}</p>
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={() => updateQty(item.productId, item.quantity - 1)} className="w-7 h-7 rounded-full border border-gray-300 text-sm flex items-center justify-center hover:bg-gray-50">-</button>
                  <span className="text-sm w-4 text-center">{item.quantity}</span>
                  <button onClick={() => updateQty(item.productId, item.quantity + 1)} className="w-7 h-7 rounded-full border border-gray-300 text-sm flex items-center justify-center hover:bg-gray-50">+</button>
                </div>
              </div>
            ))}
          </div>
        )}

        {items.length > 0 && (
          <div className="px-5 py-4 border-t border-gray-200">
            <div className="flex items-center justify-between mb-4">
              <span className="text-sm text-gray-500">Итого</span>
              <span className="font-semibold text-gray-900">{formatPrice(total)}</span>
            </div>
            <button
              onClick={() => { onClose(); router.push(`/${slug}/checkout`) }}
              className="w-full bg-gray-900 text-white py-3 rounded-xl font-medium hover:bg-gray-800 transition-colors"
            >
              Оформить заказ
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
