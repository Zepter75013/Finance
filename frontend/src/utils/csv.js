function stripBom(text) {
  return text.charCodeAt(0) === 0xfeff ? text.slice(1) : text
}

function detectDelimiter(headerLine) {
  const candidates = [',', ';', '\t']
  let best = ','
  let bestCount = -1

  for (const candidate of candidates) {
    const count = headerLine.split(candidate).length - 1
    if (count > bestCount) {
      bestCount = count
      best = candidate
    }
  }

  return best
}

function parseLine(line, delimiter) {
  const fields = []
  let current = ''
  let inQuotes = false

  for (let i = 0; i < line.length; i += 1) {
    const char = line[i]

    if (inQuotes) {
      if (char === '"') {
        if (line[i + 1] === '"') {
          current += '"'
          i += 1
        } else {
          inQuotes = false
        }
      } else {
        current += char
      }
      continue
    }

    if (char === '"') {
      inQuotes = true
      continue
    }

    if (char === delimiter) {
      fields.push(current)
      current = ''
      continue
    }

    current += char
  }

  fields.push(current)

  return fields.map((field) => field.trim())
}

export function parseCsv(text) {
  const normalized = stripBom(String(text ?? '')).replace(/\r\n/g, '\n').replace(/\r/g, '\n')
  const lines = normalized.split('\n').filter((line) => line.trim() !== '')

  if (!lines.length) {
    return { headers: [], rows: [] }
  }

  const delimiter = detectDelimiter(lines[0])
  const headers = parseLine(lines[0], delimiter)
  const rows = lines.slice(1).map((line) => parseLine(line, delimiter))

  return { headers, rows }
}

export function normalizeAmount(raw) {
  if (raw == null) return NaN

  let value = String(raw)
    .replace(/ /g, ' ')
    .replace(/[€$]/g, '')
    .trim()

  if (value === '') return NaN

  const hasComma = value.includes(',')
  const hasDot = value.includes('.')

  if (hasComma && hasDot) {
    value = value.replace(/\./g, '').replace(',', '.')
  } else if (hasComma) {
    value = value.replace(/\s/g, '').replace(',', '.')
  } else {
    value = value.replace(/\s/g, '')
  }

  return Number.parseFloat(value)
}

export function normalizeDate(raw) {
  const value = String(raw ?? '').trim()
  if (!value) return ''

  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return value
  }

  const slashOrDash = value.match(/^(\d{1,2})[/-](\d{1,2})[/-](\d{4})$/)
  if (slashOrDash) {
    const [, day, month, year] = slashOrDash
    return `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`
  }

  const parsed = new Date(value)
  if (!Number.isNaN(parsed.getTime())) {
    return parsed.toISOString().slice(0, 10)
  }

  return ''
}
