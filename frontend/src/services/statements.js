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
      // Le backend renvoie ses erreurs en texte brut (http.Error) plutôt qu'en
      // JSON — on récupère quand même le texte pour ne pas perdre le détail
      // (ex: "ce relevé est verrouillé").
      const text = (await response.text()).trim()
      if (text) message = text
    }
  } catch {
    // on garde le message par défaut
  }

  return message
}

export async function fetchStatements(accountId) {
  const response = await fetch(`${API_BASE_URL}/statements?account_id=${accountId}`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les relevés.')
  }

  return parseJson(response)
}

export async function saveStatement(payload) {
  const response = await fetch(`${API_BASE_URL}/statements`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’enregistrer le relevé.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function setStatementLock(id, isLocked) {
  const response = await fetch(`${API_BASE_URL}/statements/${id}`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ is_locked: isLocked }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(
      response,
      isLocked ? 'Impossible de verrouiller le relevé.' : 'Impossible de déverrouiller le relevé.'
    )
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteStatement(id) {
  const response = await fetch(`${API_BASE_URL}/statements/${id}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer le relevé.')
    throw new Error(message)
  }
}

export async function fetchStatementPdfs(statementId) {
  const response = await fetch(`${API_BASE_URL}/statements/${statementId}/pdfs`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les PDF de ce relevé.')
  }

  return parseJson(response)
}

export async function uploadStatementPdf(statementId, file) {
  const formData = new FormData()
  formData.append('file', file)

  const response = await fetch(`${API_BASE_URL}/statements/${statementId}/pdfs`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
    body: formData,
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’enregistrer le PDF.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function viewStatementPdf(statementId, pdfId) {
  const response = await fetch(`${API_BASE_URL}/statements/${statementId}/pdfs/${pdfId}`, {
    method: 'GET',
    credentials: 'include',
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’ouvrir le PDF.')
    throw new Error(message)
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  window.open(url, '_blank', 'noopener')
  // Laisse le temps au nouvel onglet de charger le blob avant de le révoquer.
  setTimeout(() => URL.revokeObjectURL(url), 60000)
}

export async function deleteStatementPdf(statementId, pdfId) {
  const response = await fetch(`${API_BASE_URL}/statements/${statementId}/pdfs/${pdfId}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer le PDF.')
    throw new Error(message)
  }
}
