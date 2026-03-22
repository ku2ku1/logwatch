import { useState, useEffect } from 'react'

const api = (url, opts = {}) => {
  const token = localStorage.getItem('token')
  return fetch(url, {
    ...opts,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...opts.headers }
  }).then(r => r.json())
}

export default function Security2FA() {
  const [status, setStatus] = useState(null)
  const [setup, setSetup] = useState(null)
  const [code, setCode] = useState('')
  const [msg, setMsg] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api('/api/auth/totp/status').then(setStatus)
  }, [])

  const startSetup = async () => {
    setLoading(true)
    const data = await api('/api/auth/totp/setup', { method: 'POST' })
    setSetup(data)
    setLoading(false)
  }

  const enable2FA = async () => {
    if (code.length !== 6) { setMsg('6 digit code daalo'); return }
    setLoading(true)
    const data = await api('/api/auth/totp/enable', {
      method: 'POST',
      body: JSON.stringify({ code })
    })
    setMsg(data.status || data.error || 'Error')
    if (data.status) { setStatus({ enabled: true }); setSetup(null); setCode('') }
    setLoading(false)
  }

  const disable2FA = async () => {
    if (code.length !== 6) { setMsg('6 digit code daalo'); return }
    setLoading(true)
    const data = await api('/api/auth/totp/disable', {
      method: 'POST',
      body: JSON.stringify({ code })
    })
    setMsg(data.status || data.error || 'Error')
    if (data.status) { setStatus({ enabled: false }); setCode('') }
    setLoading(false)
  }

  const inp = { width: '100%', background: '#0f172a', border: '1px solid #2d3748', borderRadius: 8, padding: '10px 14px', color: '#e2e8f0', fontSize: 14, outline: 'none', boxSizing: 'border-box', letterSpacing: 4, textAlign: 'center', fontSize: 22, fontWeight: 700 }

  return (
    <div style={{ maxWidth: 480, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 20 }}>
      <div style={{ color: '#f1f5f9', fontWeight: 700, fontSize: 18 }}>Two-Factor Authentication (2FA)</div>

      {/* Status */}
      <div style={{ background: '#1e293b', border: `1px solid ${status?.enabled ? 'rgba(34,197,94,0.3)' : '#2d3748'}`, borderRadius: 12, padding: 20, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <div style={{ color: '#f1f5f9', fontWeight: 600 }}>TOTP Authenticator</div>
          <div style={{ color: '#64748b', fontSize: 13, marginTop: 4 }}>Google Authenticator / Authy</div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div style={{ width: 8, height: 8, borderRadius: '50%', background: status?.enabled ? '#22c55e' : '#ef4444' }}/>
          <span style={{ color: status?.enabled ? '#22c55e' : '#ef4444', fontSize: 13, fontWeight: 600 }}>
            {status?.enabled ? 'Enabled' : 'Disabled'}
          </span>
        </div>
      </div>

      {msg && (
        <div style={{ background: msg.includes('enabled') || msg.includes('disabled') ? 'rgba(34,197,94,0.1)' : 'rgba(239,68,68,0.1)', border: `1px solid ${msg.includes('enabled') || msg.includes('disabled') ? 'rgba(34,197,94,0.3)' : 'rgba(239,68,68,0.3)'}`, color: msg.includes('enabled') || msg.includes('disabled') ? '#22c55e' : '#f87171', borderRadius: 8, padding: '10px 14px', fontSize: 13 }}>
          {msg}
        </div>
      )}

      {/* Setup flow */}
      {!status?.enabled && !setup && (
        <button onClick={startSetup} disabled={loading} style={{ background: '#6366f1', border: 'none', borderRadius: 8, padding: '12px', color: 'white', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>
          {loading ? 'Loading...' : 'Setup 2FA'}
        </button>
      )}

      {setup && (
        <div style={{ background: '#1e293b', border: '1px solid #2d3748', borderRadius: 12, padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ color: '#f1f5f9', fontWeight: 600 }}>Step 1 — QR Code scan karo</div>
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <img src={`data:image/png;base64,${setup.qr_code}`} alt="QR Code" style={{ width: 180, height: 180, borderRadius: 8, border: '4px solid white' }}/>
          </div>
          <div style={{ background: '#0f172a', borderRadius: 8, padding: 12, textAlign: 'center' }}>
            <div style={{ color: '#64748b', fontSize: 11, marginBottom: 4 }}>Manual key</div>
            <code style={{ color: '#a5b4fc', fontSize: 13, letterSpacing: 2 }}>{setup.secret}</code>
          </div>
          <div style={{ color: '#f1f5f9', fontWeight: 600 }}>Step 2 — Code verify karo</div>
          <input value={code} onChange={e => setCode(e.target.value.replace(/\D/g,'').slice(0,6))} placeholder="000000" style={inp} maxLength={6}/>
          <button onClick={enable2FA} disabled={loading || code.length !== 6} style={{ background: code.length === 6 ? '#22c55e' : '#1e293b', border: '1px solid #2d3748', borderRadius: 8, padding: 12, color: 'white', fontSize: 14, fontWeight: 600, cursor: code.length === 6 ? 'pointer' : 'not-allowed' }}>
            {loading ? 'Verifying...' : 'Enable 2FA'}
          </button>
        </div>
      )}

      {status?.enabled && (
        <div style={{ background: '#1e293b', border: '1px solid #2d3748', borderRadius: 12, padding: 24, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ color: '#f1f5f9', fontWeight: 600 }}>2FA Disable karna chahte ho?</div>
          <div style={{ color: '#64748b', fontSize: 13 }}>Authenticator app se current code daalo</div>
          <input value={code} onChange={e => setCode(e.target.value.replace(/\D/g,'').slice(0,6))} placeholder="000000" style={inp} maxLength={6}/>
          <button onClick={disable2FA} disabled={loading || code.length !== 6} style={{ background: 'rgba(239,68,68,0.15)', border: '1px solid rgba(239,68,68,0.3)', borderRadius: 8, padding: 12, color: '#f87171', fontSize: 14, fontWeight: 600, cursor: 'pointer' }}>
            Disable 2FA
          </button>
        </div>
      )}

      <div style={{ background: '#1e293b', border: '1px solid #2d3748', borderRadius: 12, padding: 16 }}>
        <div style={{ color: '#64748b', fontSize: 12, lineHeight: 1.6 }}>
          <div style={{ color: '#94a3b8', fontWeight: 600, marginBottom: 8 }}>Rate Limiting Active</div>
          API: 100 req/min per IP<br/>
          Login: 5 attempts/min → 5 min block<br/>
          Headers: X-RateLimit-Remaining
        </div>
      </div>
    </div>
  )
}
