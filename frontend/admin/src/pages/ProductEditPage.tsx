import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api, type Product } from '../api/client'

export default function ProductEditPage() {
  const { storeId, productId } = useParams<{ storeId: string; productId?: string }>()
  const navigate = useNavigate()
  const isEdit = !!productId
  const fileRef = useRef<HTMLInputElement>(null)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [price, setPrice] = useState('')
  const [stock, setStock] = useState('0')
  const [imageUrl, setImageUrl] = useState('')
  const [published, setPublished] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)

  useEffect(() => {
    if (!isEdit) return
    api.products.get(storeId!, productId!).then((p: Product) => {
      setName(p.name)
      setDescription(p.description ?? '')
      setPrice(String(p.price / 100))
      setStock(String(p.stock))
      setImageUrl(p.image_url ?? '')
      setPublished(p.published)
    }).catch(() => navigate(`/stores/${storeId}/products`))
  }, [productId])

  async function handleImageUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const ext = file.name.split('.').pop() ?? 'jpg'
      const { upload_url, public_url } = await api.upload.presign(ext)
      await fetch(upload_url, { method: 'PUT', body: file, headers: { 'Content-Type': file.type } })
      setImageUrl(public_url)
    } catch {
      setError('Ошибка загрузки изображения')
    } finally {
      setUploading(false)
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    const data = {
      name,
      description: description || undefined,
      price: Math.round(parseFloat(price) * 100),
      stock: parseInt(stock),
      image_url: imageUrl || undefined,
      published,
    }
    try {
      if (isEdit) {
        await api.products.update(storeId!, productId!, data)
      } else {
        await api.products.create(storeId!, data)
      }
      navigate(`/stores/${storeId}/products`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка сохранения')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200 px-6 py-4">
        <button onClick={() => navigate(`/stores/${storeId}/products`)} className="text-sm text-gray-500 hover:text-gray-900">
          ← Назад к товарам
        </button>
      </header>

      <main className="max-w-2xl mx-auto px-6 py-10">
        <h1 className="text-xl font-semibold text-gray-900 mb-8">
          {isEdit ? 'Редактировать товар' : 'Добавить товар'}
        </h1>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <label className="block text-sm font-medium text-gray-700 mb-3">Фото товара</label>
            <div className="flex items-start gap-4">
              <div
                className="w-24 h-24 rounded-xl bg-gray-100 overflow-hidden cursor-pointer flex items-center justify-center shrink-0"
                onClick={() => fileRef.current?.click()}
              >
                {imageUrl
                  ? <img src={imageUrl} alt="preview" className="w-full h-full object-cover" />
                  : <span className="text-gray-400 text-xs text-center px-2">{uploading ? 'Загрузка...' : 'Нажмите'}</span>
                }
              </div>
              <div>
                <button
                  type="button"
                  onClick={() => fileRef.current?.click()}
                  disabled={uploading}
                  className="text-sm text-gray-700 border border-gray-300 px-3 py-1.5 rounded-lg hover:bg-gray-50 disabled:opacity-50"
                >
                  {uploading ? 'Загрузка...' : 'Выбрать файл'}
                </button>
                <p className="text-xs text-gray-400 mt-2">JPG, PNG или WebP</p>
              </div>
            </div>
            <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={handleImageUpload} />
          </div>

          <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Название *</label>
              <input
                type="text" value={name} onChange={(e) => setName(e.target.value)} required
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Описание</label>
              <textarea
                value={description} onChange={(e) => setDescription(e.target.value)} rows={4}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent resize-none"
              />
            </div>
          </div>

          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Цена (₽) *</label>
                <input
                  type="number" min="0" step="0.01" value={price} onChange={(e) => setPrice(e.target.value)} required
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Остаток</label>
                <input
                  type="number" min="0" value={stock} onChange={(e) => setStock(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
                />
              </div>
            </div>
          </div>

          <div className="bg-white rounded-xl border border-gray-200 p-5 flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-700">Опубликовать товар</p>
              <p className="text-xs text-gray-400 mt-0.5">Товар будет виден покупателям</p>
            </div>
            <button
              type="button" onClick={() => setPublished(!published)}
              className={`relative w-11 h-6 rounded-full transition-colors ${published ? 'bg-gray-900' : 'bg-gray-200'}`}
            >
              <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform ${published ? 'translate-x-5' : ''}`} />
            </button>
          </div>

          {error && <p className="text-sm text-red-600 bg-red-50 px-3 py-2 rounded-lg">{error}</p>}

          <div className="flex gap-3">
            <button
              type="submit" disabled={loading}
              className="flex-1 bg-gray-900 text-white py-2 rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition-colors"
            >
              {loading ? 'Сохранение...' : isEdit ? 'Сохранить изменения' : 'Добавить товар'}
            </button>
            <button
              type="button" onClick={() => navigate(`/stores/${storeId}/products`)}
              className="px-4 py-2 border border-gray-300 rounded-lg text-sm text-gray-700 hover:bg-gray-50 transition-colors"
            >
              Отмена
            </button>
          </div>
        </form>
      </main>
    </div>
  )
}
