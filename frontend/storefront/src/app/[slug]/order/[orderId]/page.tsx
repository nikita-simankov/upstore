import { notFound } from 'next/navigation'

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

async function getOrder(orderId: string) {
  const res = await fetch(`${API}/api/v1/public/orders/${orderId}`, { cache: 'no-store' })
  return res.ok ? res.json() : null
}

const statusLabels: Record<string, string> = {
  pending: 'Ожидает оплаты',
  paid: 'Оплачен',
  shipped: 'Отправлен',
  delivered: 'Доставлен',
  cancelled: 'Отменён',
}

export default async function OrderPage({ params }: { params: { slug: string; orderId: string } }) {
  const order = await getOrder(params.orderId)
  if (!order) notFound()

  const isPaid = order.status === 'paid' || order.status === 'shipped' || order.status === 'delivered'

  return (
    <main className="max-w-md mx-auto px-6 py-20 text-center">
      <div className={`w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-6 text-2xl ${
        isPaid ? 'bg-green-100' : 'bg-gray-100'
      }`}>
        {isPaid ? '✓' : '⧖'}
      </div>
      <h1 className="text-2xl font-bold text-gray-900 mb-2">
        {isPaid ? 'Заказ принят!' : 'Заказ оформлен'}
      </h1>
      <p className="text-gray-500 text-sm mb-2">Статус: {statusLabels[order.status] ?? order.status}</p>
      <p className="text-gray-400 text-xs mb-8">Заказ #{order.id.slice(0, 8).toUpperCase()}</p>
      <p className="text-sm text-gray-500">Подтверждение будет отправлено на {order.customer_email}</p>
      <a href={`/${params.slug}`} className="mt-8 inline-block text-sm text-gray-900 underline">Вернуться в магазин</a>
    </main>
  )
}
