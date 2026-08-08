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
    } else {
      const text = (await response.text()).trim()
      if (text) message = text
    }
  } catch {
    // on garde le message par défaut
  }

  return message
}

export async function fetchDatabaseSettings() {
  const response = await fetch(`${API_BASE_URL}/settings/database`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger le réglage de base de données.')
  }

  return parseJson(response)
}

export async function updateDatabaseSettings(driver) {
  const response = await fetch(`${API_BASE_URL}/settings/database`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ driver }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de changer le réglage de base de données.')
    throw new Error(message)
  }

  return parseJson(response)
}
