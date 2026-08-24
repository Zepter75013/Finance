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

export async function fetchAccounts() {
  const response = await fetch(`${API_BASE_URL}/accounts`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les comptes.')
  }

  return parseJson(response)
}

export async function createAccount(payload) {
  const response = await fetch(`${API_BASE_URL}/accounts`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de créer le compte.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateAccount(id, payload) {
  const response = await fetch(`${API_BASE_URL}/accounts/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de modifier le compte.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateOpeningBalance(id, amount, date) {
  const response = await fetch(`${API_BASE_URL}/accounts/${id}/opening-balance`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ amount, date }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de mettre à jour le solde initial.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function clearOpeningBalance(id) {
  const response = await fetch(`${API_BASE_URL}/accounts/${id}/opening-balance`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’effacer le solde initial.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateHasStatements(id, hasStatements) {
  const response = await fetch(`${API_BASE_URL}/accounts/${id}/has-statements`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ has_statements: hasStatements }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de mettre à jour ce réglage.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteAccount(id) {
  const response = await fetch(`${API_BASE_URL}/accounts/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer le compte.')
    throw new Error(message)
  }
}
