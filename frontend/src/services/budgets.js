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

export async function fetchBudgets(accountId, monthKey) {
  const response = await fetch(`${API_BASE_URL}/budgets?account_id=${accountId}&month=${monthKey}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les budgets.')
  }

  return parseJson(response)
}

export async function upsertBudget(categoryId, monthKey, amount) {
  const response = await fetch(`${API_BASE_URL}/budgets`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ category_id: categoryId, month: monthKey, amount }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de mettre à jour le budget.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteBudget(categoryId, monthKey) {
  const response = await fetch(
    `${API_BASE_URL}/budgets?category_id=${categoryId}&month=${monthKey}`,
    {
      method: 'DELETE',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
    }
  )

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer ce budget.')
    throw new Error(message)
  }
}
