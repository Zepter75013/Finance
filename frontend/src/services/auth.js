import { API_BASE_URL } from '../config'

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

export async function login(username, password) {
  const response = await fetch(`${API_BASE_URL}/auth/login`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ username, password }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Identifiants invalides.')
    throw new Error(message)
  }

  return response.json()
}

export async function logout() {
  await fetch(`${API_BASE_URL}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })
}

export async function changePassword(currentPassword, newPassword) {
  const response = await fetch(`${API_BASE_URL}/auth/change-password`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ currentPassword, newPassword }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de modifier le mot de passe.')
    throw new Error(message)
  }
}

export async function requestResetCode(username) {
  const response = await fetch(`${API_BASE_URL}/auth/request-reset-code`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ username }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’envoyer le code.')
    throw new Error(message)
  }
}

export async function resetPasswordWithCode(username, code, newPassword) {
  const response = await fetch(`${API_BASE_URL}/auth/reset-password-with-code`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ username, code, newPassword }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de modifier le mot de passe.')
    throw new Error(message)
  }
}

export async function fetchCurrentUser() {
  const response = await fetch(`${API_BASE_URL}/auth/me`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    return null
  }

  return response.json()
}
