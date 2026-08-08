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
      // Le backend renvoie ses erreurs en texte brut (http.Error) plutôt qu'en
      // JSON — on récupère quand même le texte pour ne pas perdre le détail
      // (ex: "une sous-catégorie avec ce nom existe déjà").
      const text = (await response.text()).trim()
      if (text) message = text
    }
  } catch {
    // on garde le message par défaut
  }

  return message
}

export async function fetchSubCategories() {
  const response = await fetch(`${API_BASE_URL}/subcategories`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les sous-catégories.')
  }

  return parseJson(response)
}

export async function createSubCategory(payload) {
  const response = await fetch(`${API_BASE_URL}/subcategories`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de créer la sous-catégorie.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function updateSubCategory(id, name) {
  const response = await fetch(`${API_BASE_URL}/subcategories/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ name }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de modifier la sous-catégorie.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteSubCategory(id) {
  const response = await fetch(`${API_BASE_URL}/subcategories/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer la sous-catégorie.')
    throw new Error(message)
  }
}
