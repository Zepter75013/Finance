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

export async function fetchBackupSettings() {
  const response = await fetch(`${API_BASE_URL}/backups/settings`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger le dossier de sauvegarde.')
  }

  return parseJson(response)
}

export async function updateBackupSettings(directory) {
  const response = await fetch(`${API_BASE_URL}/backups/settings`, {
    method: 'PUT',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({ directory }),
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de changer le dossier de sauvegarde.')
    throw new Error(message)
  }

  return parseJson(response)
}

// Ouvre le sélecteur de dossier natif (Finder) sur la machine qui fait
// tourner le serveur. Retourne le chemin choisi, ou null si l'utilisateur a
// annulé la sélection.
export async function pickBackupDirectory() {
  const response = await fetch(`${API_BASE_URL}/backups/settings/pick-directory`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (response.status === 204) {
    return null
  }

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible d’ouvrir le sélecteur de dossier.')
    throw new Error(message)
  }

  const settings = await parseJson(response)
  return settings.directory
}

export async function fetchBackups() {
  const response = await fetch(`${API_BASE_URL}/backups`, {
    method: 'GET',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    throw new Error('Impossible de charger les sauvegardes.')
  }

  return parseJson(response)
}

export async function createBackup() {
  const response = await fetch(`${API_BASE_URL}/backups`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de créer la sauvegarde.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function downloadBackup(name) {
  const response = await fetch(`${API_BASE_URL}/backups/${encodeURIComponent(name)}/download`, {
    method: 'GET',
    credentials: 'include',
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de télécharger la sauvegarde.')
    throw new Error(message)
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = name
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export async function restoreBackup(name) {
  const response = await fetch(`${API_BASE_URL}/backups/${encodeURIComponent(name)}/restore`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de restaurer la sauvegarde.')
    throw new Error(message)
  }
}

export async function restoreUploadedBackup(file) {
  const formData = new FormData()
  formData.append('file', file)

  const response = await fetch(`${API_BASE_URL}/backups/restore-upload`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
    body: formData,
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de restaurer ce fichier.')
    throw new Error(message)
  }

  return parseJson(response)
}

export async function deleteBackup(name) {
  const response = await fetch(`${API_BASE_URL}/backups/${encodeURIComponent(name)}`, {
    method: 'DELETE',
    credentials: 'include',
    headers: {
      Accept: 'application/json',
    },
  })

  if (!response.ok) {
    const message = await extractErrorMessage(response, 'Impossible de supprimer la sauvegarde.')
    throw new Error(message)
  }
}
