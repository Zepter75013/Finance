import { API_BASE_URL } from '../config'

async function parseJson(response) {
  const contentType = response.headers.get('content-type') || ''

  if (!contentType.includes('application/json')) {
    throw new Error('Réponse API invalide.')
  }

  return response.json()
}

async function extractErrorMessage(response, fallbackMessage) {
  let message = fallbackMessage

  try {
    const contentType = response.headers.get('content-type') || ''

    if (contentType.includes('application/json')) {
      const errorData = await response.json()
      if (errorData?.message) {
        message = errorData.message
      }
    }
  } catch {
    // on garde le message par défaut
  }

  return message
}

export async function fetchUsers() {
  const response = await fetch(`${API_BASE_URL}/users`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les utilisateurs.')
  }

  return parseJson(response)
}

export async function createUser(payload) {
  const response = await fetch(`${API_BASE_URL}/users`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de créer l’utilisateur.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateUser(id, payload) {
  const response = await fetch(`${API_BASE_URL}/users/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de modifier l’utilisateur.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteUser(id) {
  const response = await fetch(`${API_BASE_URL}/users/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer l’utilisateur.')
    throw new Error(message)
  }
}
