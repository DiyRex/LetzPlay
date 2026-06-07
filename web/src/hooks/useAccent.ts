import { useCallback, useEffect, useState } from "react"

/** Accent presets (HSL values plugged into the --primary / --ring CSS variables). */
export const ACCENTS: Record<string, string> = {
  violet: "252 100% 68%",
  blue: "212 100% 60%",
  green: "152 70% 45%",
  pink: "330 90% 62%",
  orange: "28 100% 58%",
}

const STORAGE_KEY = "letzplay-accent"

/** Persists and applies a chosen accent color (client-side, per device). */
export function useAccent() {
  const [accent, setAccent] = useState<string>(() => localStorage.getItem(STORAGE_KEY) || "violet")

  useEffect(() => {
    const hsl = ACCENTS[accent] ?? ACCENTS.violet
    const root = document.documentElement
    root.style.setProperty("--primary", hsl)
    root.style.setProperty("--ring", hsl)
    localStorage.setItem(STORAGE_KEY, accent)
  }, [accent])

  const cycle = useCallback(() => {
    const keys = Object.keys(ACCENTS)
    setAccent((cur) => keys[(keys.indexOf(cur) + 1) % keys.length])
  }, [])

  return { accent, setAccent, cycle }
}
