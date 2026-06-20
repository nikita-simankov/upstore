import { CartProvider } from '../../context/CartContext'
import CartButton from '../../components/CartButton'

export default function StoreLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: { slug: string }
}) {
  return (
    <CartProvider>
      <div className="min-h-screen bg-white">
        <nav className="border-b border-gray-100 px-6 py-3 flex justify-end">
          <CartButton slug={params.slug} />
        </nav>
        {children}
      </div>
    </CartProvider>
  )
}
