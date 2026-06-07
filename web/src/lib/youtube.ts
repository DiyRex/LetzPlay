// Mirror of the server's YouTube id extraction, for client-side duplicate detection.
const BARE_ID = /^[A-Za-z0-9_-]{11}$/
const PATTERNS = [
  /youtu\.be\/([A-Za-z0-9_-]{11})/,
  /[?&]v=([A-Za-z0-9_-]{11})/,
  /\/embed\/([A-Za-z0-9_-]{11})/,
  /\/shorts\/([A-Za-z0-9_-]{11})/,
  /\/live\/([A-Za-z0-9_-]{11})/,
]

/** Extracts the 11-char video id from a link or bare id, or null. */
export function extractVideoId(input: string): string | null {
  const s = input.trim()
  if (!s) return null
  if (BARE_ID.test(s)) return s
  for (const re of PATTERNS) {
    const m = re.exec(s)
    if (m) return m[1]
  }
  return null
}
