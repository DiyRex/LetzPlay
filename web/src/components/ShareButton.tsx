import { useState } from "react"
import QRCode from "qrcode"
import { Check, Copy, Share2, X } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"

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
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.error("Couldn't copy")
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

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6"
          onClick={() => setOpen(false)}
        >
          <div
            className="flex w-full max-w-xs flex-col items-center gap-4 rounded-xl border bg-card p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex w-full items-center justify-between">
              <h2 className="text-sm font-semibold">Invite to the jukebox</h2>
              <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setOpen(false)} aria-label="Close">
                <X className="size-4" />
              </Button>
            </div>
            {qr && <img src={qr} alt="QR code to join" className="rounded-lg" />}
            <p className="break-all text-center text-xs text-muted-foreground">{url}</p>
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
        </div>
      )}
    </>
  )
}
