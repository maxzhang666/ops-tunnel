import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Open a URL in the system browser (Wails desktop) or new tab (web). */
export function openExternal(url: string) {
  const rt = (window as unknown as Record<string, unknown>).runtime as
    | { BrowserOpenURL?: (url: string) => void }
    | undefined
  if (rt?.BrowserOpenURL) {
    rt.BrowserOpenURL(url)
  } else {
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}
