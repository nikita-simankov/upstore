const TOKEN_KEY = 'upstore_token'

export function useAuth() {
  const token = localStorage.getItem(TOKEN_KEY)

  function setToken(t: string) {
    localStorage.setItem(TOKEN_KEY, t)
  }

  function clearToken() {
    localStorage.removeItem(TOKEN_KEY)
  }

  return { token, setToken, clearToken }
}
