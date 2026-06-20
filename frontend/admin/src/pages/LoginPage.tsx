import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../hooks/useAuth'

export default function LoginPage() {
  const navigate = useNavigate()
  const { setToken } = useAuth()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res =
        mode === 'login'
          ? await api.login(email, password)
          : await api.register(email, password, name)
      setToken(res.token)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Что-то пошло не так')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-gray-900 mb-1">Upstore</h1>
        <p className="text-sm text-gray-500 mb-8">Панель управления магазином</p>

        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <div className="flex rounded-lg bg-gray-100 p-1 mb-6">
            <button
              onClick={() => setMode('login')}
              className={`flex-1 py-1.5 text-sm font-medium rounded-md transition-colors ${
                mode === 'login' ? 'bg-white shadow-sm text-gray-900' : 'text-gray-500'
              }`}
            >
              Войти
            </button>
            <button
              onClick={() => setMode('register')}
              className={`flex-1 py-1.5 text-sm font-medium rounded-md transition-colors ${
                mode === 'register' ? 'bg-white shadow-sm text-gray-900' : 'text-gray-500'
              }`}
            >
              Регистрация
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            {mode === 'register' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Ваше имя</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  placeholder="Иван Иванов"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
                />
              </div>
            )}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                placeholder="you@example.com"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Пароль</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                placeholder="••••••••"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
              />
            </div>

            {error && (
              <p className="text-sm text-red-600 bg-red-50 px-3 py-2 rounded-lg">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-gray-900 text-white py-2 rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition-colors"
            >
              {loading ? 'Загрузка...' : mode === 'login' ? 'Войти' : 'Создать аккаунт'}
            </button>
          </form>

          {mode === 'register' && (
            <p className="text-xs text-gray-400 text-center mt-4">30 дней бесплатно — без карты</p>
          )}
        </div>
      </div>
    </div>
  )
}
