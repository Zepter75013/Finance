const API_BASE_URL = 'http://localhost:8080'

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

export async function fetchIncomes(accountId) {
  const response = await fetch(`${API_BASE_URL}/incomes?account_id=${accountId}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les revenus.')
  }

  return parseJson(response)
}

export async function createIncome(payload) {
  const response = await fetch(`${API_BASE_URL}/incomes`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(
      response,
      'Impossible de créer le revenu.'
    )
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateIncome(id, payload) {
  const response = await fetch(`${API_BASE_URL}/incomes/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(
      response,
      'Impossible de modifier le revenu.'
    )
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteIncome(id) {
  const response = await fetch(`${API_BASE_URL}/incomes/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(
      response,
      'Impossible de supprimer le revenu.'
    )
    throw new Error(message)
  }
}
