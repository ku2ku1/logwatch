import { useState, useEffect, useRef } from 'react'

export function useWebSocket() {
  const [data, setData] = useState({})
  const [connected, setConnected] = useState(false)
  const ws = useRef(null)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) return

    // Auto-detect WebSocket URL from current page
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const url = `${proto}//${host}/api/v1/ws?token=${token}`

    ws.current = new WebSocket(url)
    ws.current.onopen = () => setConnected(true)
    ws.current.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        setData(prev => ({ ...prev, [msg.type]: msg.data }))
      } catch {}
    }
    ws.current.onclose = () => setConnected(false)
    ws.current.onerror = (e) => console.error('WS error:', e)

    return () => ws.current?.close()
  }, [])

  return { data, connected }
}
