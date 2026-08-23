import { API_BASE_URL } from '../config'

export async function fetchAuditLog() {
  const response = await fetch(`${API_BASE_URL}/audit-log`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger le journal d’audit.')
  }

  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('application/json')) {
    throw new Error('Réponse API invalide.')
  }

  return response.json()
}
