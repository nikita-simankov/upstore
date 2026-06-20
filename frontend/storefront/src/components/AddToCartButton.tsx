'use client'
import { useCart } from '../context/CartContext'

interface Props {
  productId: string
  name: string
  price: number
  imageUrl: string | null
  stock: number
}

export default function AddToCartButton({ productId, name, price, imageUrl, stock }: Props) {
  const { addItem, items } = useCart()
  const inCart = items.find((i) => i.productId === productId)

  if (stock <= 0) {
    return (
      <button disabled className="w-full bg-gray-200 text-gray-400 py-3 rounded-xl font-medium cursor-not-allowed">
        Нет в наличии
      </button>
    )
  }

  return (
    <button
      onClick={() => addItem({ productId, name, price, imageUrl })}
      className="w-full bg-gray-900 text-white py-3 rounded-xl font-medium hover:bg-gray-800 transition-colors"
    >
      {inCart ? `Добавить ещё (${inCart.quantity} в корзине)` : 'В корзину'}
    </button>
  )
}
