/**
 * Send a native notification via Tauri (no-op in browser).
 * Uses dynamic import so the Tauri plugin is only loaded in the desktop app.
 */
export async function sendNotification(title: string, body: string) {
  try {
    // Only available in Tauri context
    if (!('__TAURI__' in window)) return

    const { sendNotification: tauriNotify } = await import('@tauri-apps/plugin-notification')
    await tauriNotify({ title, body })
  } catch {
    // Silently ignore — not in Tauri or plugin not available
  }
}
