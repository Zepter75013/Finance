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
    const errorData = await parseJson(response)
    if (errorData?.message) {
      message = errorData.message
    }
  } catch {
    // on garde le message par défaut
  }

  return message
}

export async function fetchPurchases(accountId) {
  const response = await fetch(`${API_BASE_URL}/purchases?account_id=${accountId}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les achats.')
  }

  return parseJson(response)
}

export async function fetchCategories(accountId) {
  const response = await fetch(`${API_BASE_URL}/categories?account_id=${accountId}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les catégories.')
  }

  return parseJson(response)
}

export async function fetchHealth() {
  const response = await fetch(`${API_BASE_URL}/health`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de joindre le backend.')
  }

  return parseJson(response)
}

export async function createPurchase(payload) {
  const response = await fetch(`${API_BASE_URL}/purchases`, {
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
      'Impossible de créer l’achat.'
    )

    throw new Error(message)
  }

  return parseJson(response)
}

export async function updatePurchase(id, payload) {
  const response = await fetch(`${API_BASE_URL}/purchases/${id}`, {
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
      'Impossible de modifier l’achat.'
    )

    throw new Error(message)
  }

  return parseJson(response)
}

export async function deletePurchase(id) {
  const response = await fetch(`${API_BASE_URL}/purchases/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    let message = 'Impossible de supprimer l’achat.'

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

    throw new Error(message)
  }
}

export async function createCategory(payload) {
  const response = await fetch(`${API_BASE_URL}/categories`, {
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
      'Impossible de créer la catégorie.'
    )

    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateCategory(id, payload) {
  const response = await fetch(`${API_BASE_URL}/categories/${id}`, {
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
      'Impossible de modifier la catégorie.'
    )

    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteCategory(id) {
  const response = await fetch(`${API_BASE_URL}/categories/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(
      response,
      'Impossible de supprimer la catégorie.'
    )

    throw new Error(message)
  }
}
