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

export async function fetchLatestReportMetadata() {
  const response = await fetch(`${API_BASE_URL}/reports/latest`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (response.status === 204) {
    return null
  }

  if (!response.ok) {
    throw new Error('Impossible de charger le dernier état généré.')
  }

  return parseJson(response)
}

// Sans timeout, un problème réseau ou serveur (proxy bloqué, disque lent...)
// laisserait le bouton "Génération..." tourner indéfiniment sans jamais
// afficher d'erreur — 60s est largement suffisant pour l'upload d'un PDF.
const UPLOAD_TIMEOUT_MS = 60000

export async function uploadGeneratedReport(pdfBlob, criteria) {
  const formData = new FormData()
  formData.append('file', pdfBlob, 'rapport-finance-ui.pdf')
  formData.append('criteria', criteria)

  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), UPLOAD_TIMEOUT_MS)

  let response
  try {
    response = await fetch(`${API_BASE_URL}/reports`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
      },
      body: formData,
      signal: controller.signal,
    })
  } catch (err) {
    if (err instanceof Error && err.name === 'AbortError') {
      throw new Error('L’enregistrement du rapport a pris trop de temps (délai dépassé).')
    }
    throw err
  } finally {
    clearTimeout(timeoutId)
  }

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’enregistrer le rapport généré.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function viewLatestReportPdf() {
  const response = await fetch(`${API_BASE_URL}/reports/latest/pdf`, {
    method: 'GET',
    credentials: 'include',
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’ouvrir le rapport.')
    throw new Error(message)
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  window.open(url, '_blank', 'noopener')
  setTimeout(() => URL.revokeObjectURL(url), 60000)
}
