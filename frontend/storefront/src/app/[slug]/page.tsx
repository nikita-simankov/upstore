import { notFound } from 'next/navigation'

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

async function getStore(slug: string) {
  const res = await fetch(`${API}/api/v1/public/stores/${slug}`, { next: { revalidate: 60 } })
  if (!res.ok) return null
  return res.json()
}

async function getProducts(slug: string) {
  const res = await fetch(`${API}/api/v1/public/stores/${slug}/products`, { next: { revalidate: 60 } })
  if (!res.ok) return []
  return res.json()
}

function formatPrice(kopecks: number) {
  return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(kopecks / 100)
}

export default async function StorePage({ params }: { params: { slug: string } }) {
  const [store, products] = await Promise.all([getStore(params.slug), getProducts(params.slug)])
  if (!store) notFound()

  return (
    <div className="min-h-screen bg-white">
      <header className="border-b border-gray-200 px-6 py-5">
        <div className="max-w-5xl mx-auto flex items-center gap-3">
          {store.logo_url && (
            <img src={store.logo_url} alt={store.name} className="w-8 h-8 rounded-full object-cover" />
          )}
          <h1 className="text-lg font-semibold text-gray-900">{store.name}</h1>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        {products.length === 0 ? (
          <p className="text-gray-400 text-sm text-center py-20">Товары скоро появятся</p>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
            {products.map((p: any) => (
              <a key={p.id} href={`/${params.slug}/products/${p.id}`} className="group block">
                <div className="aspect-square bg-gray-100 rounded-xl overflow-hidden mb-3">
                  {p.image_url ? (
                    <img
                      src={p.image_url}
                      alt={p.name}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                    />
                  ) : (
                    <div className="w-full h-full flex items-center justify-center text-gray-300 text-xs">Нет фото</div>
                  )}
                </div>
                <p className="text-sm font-medium text-gray-900 truncate">{p.name}</p>
                <p className="text-sm text-gray-500 mt-0.5">{formatPrice(p.price)}</p>
              </a>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}
