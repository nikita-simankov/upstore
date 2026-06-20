'use client'
import { useState } from 'react'
import { useCart } from '../context/CartContext'
import CartDrawer from './CartDrawer'

export default function CartButton({ slug }: { slug: string }) {
  const { count } = useCart()
  const [open, setOpen] = useState(false)
  return (
    <>
      <button onClick={() => setOpen(true)} className="relative text-sm text-gray-700 hover:text-gray-900">
        Корзина
        {count > 0 && (
          <span className="ml-1.5 bg-gray-900 text-white text-xs rounded-full px-1.5 py-0.5">{count}</span>
        )}
      </button>
      {open && <CartDrawer slug={slug} onClose={() => setOpen(false)} />}
    </>
  )
}
