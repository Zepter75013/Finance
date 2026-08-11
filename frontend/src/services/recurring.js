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
    } else {
      const text = (await response.text()).trim()
      if (text) message = text
    }
  } catch {
    // on garde le message par défaut
  }

  return message
}

export async function fetchRecurring(accountId) {
  const response = await fetch(`${API_BASE_URL}/recurring?account_id=${accountId}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les transactions récurrentes.')
  }

  return parseJson(response)
}

export async function createRecurring(payload) {
  const response = await fetch(`${API_BASE_URL}/recurring`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de créer la récurrence.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateRecurring(id, payload) {
  const response = await fetch(`${API_BASE_URL}/recurring/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de modifier la récurrence.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function executeRecurringNow(id) {
  const response = await fetch(`${API_BASE_URL}/recurring/${id}/execute`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’exécuter cette occurrence.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteRecurring(id) {
  const response = await fetch(`${API_BASE_URL}/recurring/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer la récurrence.')
    throw new Error(message)
  }
}
