import { notFound } from 'next/navigation'

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

async function getStore(slug: string) {
  const res = await fetch(`${API}/api/v1/public/stores/${slug}`, { next: { revalidate: 60 } })
  if (!res.ok) return null
  return res.json()
}

async function getProduct(slug: string, id: string) {
  const res = await fetch(`${API}/api/v1/public/stores/${slug}/products/${id}`, { next: { revalidate: 60 } })
  if (!res.ok) return null
  return res.json()
}

function formatPrice(kopecks: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(kopecks / 100)
}

export default async function ProductPage({ params }: { params: { slug: string; id: string } }) {
  const [store, product] = await Promise.all([getStore(params.slug), getProduct(params.slug, params.id)])
  if (!store || !product) notFound()

  return (
    <div className="min-h-screen bg-white">
      <header className="border-b border-gray-200 px-6 py-5">
        <div className="max-w-5xl mx-auto">
          <a href={`/${params.slug}`} className="text-sm text-gray-500 hover:text-gray-900">← {store.name}</a>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-10">
          <div className="aspect-square bg-gray-100 rounded-2xl overflow-hidden">
            {product.image_url ? (
              <img src={product.image_url} alt={product.name} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-gray-300 text-sm">Нет фото</div>
            )}
          </div>

          <div className="flex flex-col">
            <h1 className="text-2xl font-bold text-gray-900">{product.name}</h1>
            <p className="text-2xl font-bold text-gray-900 mt-3">{formatPrice(product.price)}</p>
            {product.description && (
              <p className="text-gray-500 text-sm mt-4 leading-relaxed">{product.description}</p>
            )}
            <div className="mt-auto pt-8">
              {product.stock > 0 ? (
                <button className="w-full bg-gray-900 text-white py-3 rounded-xl font-medium hover:bg-gray-800 transition-colors">
                  В корзину
                </button>
              ) : (
                <button disabled className="w-full bg-gray-200 text-gray-400 py-3 rounded-xl font-medium cursor-not-allowed">
                  Нет в наличии
                </button>
              )}
              {product.stock > 0 && (
                <p className="text-xs text-gray-400 text-center mt-3">В наличии: {product.stock} шт.</p>
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
