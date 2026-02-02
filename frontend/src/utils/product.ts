export function extractChip(name: string): string | null {
  const match = name.match(/(M[1-4])(?:\s*(Max|Pro|Ultra))?/i)
  if (match) {
    return match[1] + (match[2] || '')
  }
  return null
}

export function extractStorage(name: string): string | null {
  const match = name.match(/(\d+)\s*(GB|TB)/i)
  if (match) {
    return match[1] + match[2]
  }
  return null
}

export function parseSpecs(specsDetail: Record<string, string | number | boolean> | string | undefined | null) {
  if (!specsDetail) return null

  if (typeof specsDetail === 'string') {
    try {
      return JSON.parse(specsDetail)
    } catch {
      return null
    }
  }

  return specsDetail
}
