const MAX_SOURCE_BYTES = 8 * 1024 * 1024

// Resizes/crops an uploaded image file to a square JPEG data URL so avatars
// stay small regardless of the original photo's resolution.
export function resizeAvatarFile(file, size = 256, quality = 0.85) {
  return new Promise((resolve, reject) => {
    if (!file.type.startsWith('image/')) {
      reject(new Error('Le fichier sélectionné n’est pas une image.'))
      return
    }

    if (file.size > MAX_SOURCE_BYTES) {
      reject(new Error('L’image est trop volumineuse (8 Mo maximum).'))
      return
    }

    const reader = new FileReader()

    reader.onerror = () => reject(new Error('Impossible de lire l’image.'))

    reader.onload = () => {
      const image = new Image()

      image.onerror = () => reject(new Error('Impossible de lire l’image.'))

      image.onload = () => {
        const side = Math.min(image.width, image.height)
        const sx = (image.width - side) / 2
        const sy = (image.height - side) / 2

        const canvas = document.createElement('canvas')
        canvas.width = size
        canvas.height = size

        const ctx = canvas.getContext('2d')
        ctx.drawImage(image, sx, sy, side, side, 0, 0, size, size)

        resolve(canvas.toDataURL('image/jpeg', quality))
      }

      image.src = reader.result
    }

    reader.readAsDataURL(file)
  })
}
