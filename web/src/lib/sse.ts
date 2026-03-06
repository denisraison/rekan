// onResume registers a visibilitychange listener that fires the callback when
// the page becomes visible again (e.g. after lock screen or tab switch).
// Returns a cleanup function to remove the listener.
export function onResume(callback: () => void): () => void {
  function handler() {
    if (document.visibilityState === "visible") callback()
  }
  document.addEventListener("visibilitychange", handler)
  return () => document.removeEventListener("visibilitychange", handler)
}

// readSSE reads a Server-Sent Events stream and calls onData for each parsed
// JSON payload. Returns when the stream ends or an error occurs. AbortError is
// swallowed so callers can abort without special-casing.
export async function readSSE(
  body: ReadableStream<Uint8Array>,
  onData: (data: unknown) => void | Promise<void>,
): Promise<void> {
  const reader = body.getReader()
  const decoder = new TextDecoder()
  let buf = ""
  try {
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      const lines = buf.split("\n")
      buf = lines.pop() ?? ""
      for (const line of lines) {
        if (!line.startsWith("data: ")) continue
        try {
          await onData(JSON.parse(line.slice(6)))
        } catch {}
      }
    }
  } finally {
    reader.releaseLock()
  }
}
