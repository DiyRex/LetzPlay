import React from "react"
import ReactDOM from "react-dom/client"
import App from "./App"
import "./index.css"

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)

// Register the service worker for installability + offline shell. Service workers require a secure
// context, so this only activates over HTTPS or on localhost (a plain-http LAN IP can't use it —
// iOS "Add to Home Screen" still works there via the manifest + apple meta tags).
if ("serviceWorker" in navigator && (location.protocol === "https:" || location.hostname === "localhost")) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("./sw.js").catch(() => undefined)
  })
}
