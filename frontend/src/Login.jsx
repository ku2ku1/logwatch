import { useState } from 'react'
import { login } from './api'

export default function Login({ onLogin }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!username || !password) { setError('Username aur password dono zaroori hain'); return }
    setLoading(true)
    setError('')
    try {
      const data = await login(username, password)
      localStorage.setItem('token', data.token)
      localStorage.setItem('user', JSON.stringify(data.user))
      onLogin(data.user)
    } catch {
      setError('Invalid username ya password')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ minHeight:'100vh', background:'#0f1117', display:'flex', alignItems:'center', justifyContent:'center' }}>
      <div style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:16, padding:'40px 36px', width:360 }}>
        <div style={{ display:'flex', alignItems:'center', gap:12, marginBottom:32 }}>
          <div style={{ width:40, height:40, background:'#6366f1', borderRadius:10, display:'flex', alignItems:'center', justifyContent:'center', color:'white', fontWeight:800, fontSize:15 }}>LW</div>
          <div>
            <div style={{ color:'#f1f5f9', fontWeight:700, fontSize:18 }}>Logvance</div>
            <div style={{ color:'#64748b', fontSize:12 }}>Sign in to your account</div>
          </div>
        </div>

        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom:16 }}>
            <label style={{ color:'#94a3b8', fontSize:12, fontWeight:600, textTransform:'uppercase', letterSpacing:1, display:'block', marginBottom:6 }}>Username</label>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder="admin"
              autoFocus
              style={{ width:'100%', background:'#0f172a', border:'1px solid #2d3748', borderRadius:8, padding:'10px 14px', color:'#e2e8f0', fontSize:14, outline:'none', boxSizing:'border-box' }}
            />
          </div>
          <div style={{ marginBottom:24 }}>
            <label style={{ color:'#94a3b8', fontSize:12, fontWeight:600, textTransform:'uppercase', letterSpacing:1, display:'block', marginBottom:6 }}>Password</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="••••••••"
              style={{ width:'100%', background:'#0f172a', border:'1px solid #2d3748', borderRadius:8, padding:'10px 14px', color:'#e2e8f0', fontSize:14, outline:'none', boxSizing:'border-box' }}
            />
          </div>

          {error && (
            <div style={{ background:'rgba(239,68,68,0.1)', border:'1px solid rgba(239,68,68,0.3)', color:'#f87171', borderRadius:8, padding:'10px 14px', fontSize:13, marginBottom:16 }}>
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            style={{ width:'100%', background: loading ? '#4f46e5' : '#6366f1', border:'none', borderRadius:8, padding:'11px', color:'white', fontSize:14, fontWeight:600, cursor: loading ? 'not-allowed' : 'pointer', transition:'background 0.2s' }}
          >
            {loading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>

        <div style={{ marginTop:24, padding:'12px 14px', background:'#0f172a', borderRadius:8, border:'1px solid #1e293b' }}>
          <div style={{ color:'#475569', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:6 }}>Default credentials</div>
          <div style={{ color:'#64748b', fontSize:12 }}>Username: <span style={{ color:'#a5b4fc' }}>admin</span></div>
          <div style={{ color:'#64748b', fontSize:12 }}>Password: <span style={{ color:'#a5b4fc' }}>admin123!</span></div>
        </div>
      </div>
    </div>
  )
}
