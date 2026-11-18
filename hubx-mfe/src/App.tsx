import { useEffect, useState } from 'react'

// Contrato del endpoint de salud expuesto por el gateway (/api/health).
interface HealthResponse {
  status: string
  service: string
}

// Estados de interfaz explícitos: carga, éxito y fallo.
type HealthState =
  | { kind: 'loading' }
  | { kind: 'ok'; status: string; service: string }
  | { kind: 'error'; httpStatus: number }

function App() {
  const [state, setState] = useState<HealthState>({ kind: 'loading' })

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        const res = await fetch('/api/health')
        if (!res.ok) {
          if (!cancelled) setState({ kind: 'error', httpStatus: res.status })
          return
        }
        const data = (await res.json()) as HealthResponse
        if (!cancelled) setState({ kind: 'ok', status: data.status, service: data.service })
      } catch {
        // Sin respuesta HTTP: gateway caído o red ausente.
        if (!cancelled) setState({ kind: 'error', httpStatus: 0 })
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <main>
      {state.kind === 'loading' && <p>Cargando…</p>}
      {state.kind === 'ok' && (
        <p>
          API: {state.status} ({state.service})
        </p>
      )}
      {state.kind === 'error' && <p>Error: HTTP {state.httpStatus}</p>}
    </main>
  )
}

export default App
