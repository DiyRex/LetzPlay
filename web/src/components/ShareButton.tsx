import { useState } from "react"
import { createPortal } from "react-dom"
import QRCode from "qrcode"
import { Check, Copy, Share2, X } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"

/**
 * Copies text, falling back to a hidden-textarea + execCommand for insecure contexts — the
 * Clipboard API is unavailable over plain http (LAN IPs), which is exactly how this app is served.
 */
async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    /* fall through to legacy path */
  }
  try {
    const ta = document.createElement("textarea")
    ta.value = text
    ta.style.position = "fixed"
    ta.style.top = "0"
    ta.style.opacity = "0"
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand("copy")
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}

/** Header button that shows a QR + link so others can join the jukebox from their phone. */
export function ShareButton() {
  const [open, setOpen] = useState(false)
  const [qr, setQr] = useState<string>("")
  const [copied, setCopied] = useState(false)
  const url = typeof window !== "undefined" ? window.location.origin : ""

  const show = async () => {
    setOpen(true)
    try {
      setQr(await QRCode.toDataURL(url, { width: 240, margin: 1 }))
    } catch {
      /* ignore */
    }
  }

  const copy = async () => {
    const ok = await copyText(url)
    if (ok) {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } else {
      toast.error("Couldn't copy — long-press the link to copy it")
    }
  }

  const nativeShare = async () => {
    try {
      await navigator.share({ title: "LetzPlay Musix", text: "Join the jukebox", url })
    } catch {
      /* user cancelled / unsupported */
    }
  }

  return (
    <>
      <Button variant="ghost" size="icon" onClick={show} aria-label="Share / invite">
        <Share2 className="size-4" />
      </Button>

      {open &&
        createPortal(
          <div
            className="fixed inset-0 z-[100] flex items-center justify-center overflow-y-auto bg-black/70 p-4"
            onClick={() => setOpen(false)}
          >
            <div
              className="my-auto flex max-h-[90dvh] w-full max-w-xs flex-col items-center gap-4 overflow-y-auto rounded-xl border bg-card p-5"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex w-full items-center justify-between">
                <h2 className="text-sm font-semibold">Invite to the jukebox</h2>
                <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setOpen(false)} aria-label="Close">
                  <X className="size-4" />
                </Button>
              </div>
              {qr && (
                <img
                  src={qr}
                  alt="QR code to join"
                  className="h-auto w-full max-w-[15rem] shrink-0 rounded-lg bg-white p-2"
                />
              )}
              <p className="w-full break-all text-center text-xs text-muted-foreground">{url}</p>
              <div className="flex w-full gap-2">
                <Button variant="outline" size="sm" className="flex-1" onClick={copy}>
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                  {copied ? "Copied" : "Copy link"}
                </Button>
                {typeof navigator !== "undefined" && "share" in navigator && (
                  <Button size="sm" className="flex-1" onClick={nativeShare}>
                    <Share2 className="size-4" />
                    Share
                  </Button>
                )}
              </div>
            </div>
          </div>,
          document.body,
        )}
    </>
  )
}
