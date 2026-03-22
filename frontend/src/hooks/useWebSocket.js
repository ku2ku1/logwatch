import { useState, useEffect, useRef } from 'react'

export function useWebSocket(url) {
  const [data, setData] = useState({})
  const [connected, setConnected] = useState(false)
  const ws = useRef(null)

  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) return

    ws.current = new WebSocket(`${url}?token=${token}`)

    ws.current.onopen = () => {
      setConnected(true)
    }

    ws.current.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        setData(prev => ({ ...prev, [msg.type]: msg.data }))
      } catch (err) {
        console.error('WS parse error:', err)
      }
    }

    ws.current.onclose = () => {
      setConnected(false)
    }

    ws.current.onerror = (error) => {
      console.error('WS error:', error)
    }

    return () => {
      if (ws.current) {
        ws.current.close()
      }
    }
  }, [url])

  return { data, connected }
}