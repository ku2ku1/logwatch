import { useState, useEffect, useCallback } from 'react'
import Login from './Login'
import Reports from './Reports'
import WorldMap from './WorldMap'
import { useWebSocket } from './hooks/useWebSocket'
import {
  getStats, getTopPaths, getTopIPs, getStatusCodes,
  getSecurityStats, getRecentThreats, getTopAttackers,
  logout, listUsers, createUser, deleteUser
} from './api'
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, LineChart, Line, CartesianGrid
} from 'recharts'

const COLORS = ['#6366f1','#22d3ee','#f59e0b','#ef4444','#10b981','#f472b6','#a78bfa','#34d399']
const SEVERITY_COLORS = {
  critical: 'background:rgba(239,68,68,0.15);color:#f87171;border:1px solid rgba(239,68,68,0.3)',
  high:     'background:rgba(249,115,22,0.15);color:#fb923c;border:1px solid rgba(249,115,22,0.3)',
  medium:   'background:rgba(234,179,8,0.15);color:#facc15;border:1px solid rgba(234,179,8,0.3)',
  low:      'background:rgba(99,102,241,0.15);color:#a5b4fc;border:1px solid rgba(99,102,241,0.3)',
}
const THREAT_ICONS = { sql_injection:'💉', xss:'🔥', path_traversal:'📂', scanner_bot:'🤖', brute_force:'🔨' }

function StatCard({ title, value, sub, color='indigo' }) {
  const colors = { indigo:'#6366f1', green:'#22c55e', amber:'#f59e0b', red:'#ef4444' }
  const c = colors[color] || colors.indigo
  return (
    <div style={{ background:'#1e293b', borderRadius:12, padding:'20px', borderLeft:`4px solid ${c}` }}>
      <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1 }}>{title}</div>
      <div style={{ color:c, fontSize:28, fontWeight:800, margin:'4px 0 2px' }}>{value}</div>
      {sub && <div style={{ color:'#475569', fontSize:12 }}>{sub}</div>}
    </div>
  )
}

function Dashboard({ stats, paths, ips, codes, history }) {
  const fmt = n => n>=1e9?(n/1e9).toFixed(1)+'GB':n>=1e6?(n/1e6).toFixed(1)+'MB':n>=1e3?(n/1e3).toFixed(1)+'KB':n+'B'
  return (
    <div style={{ display:'flex', flexDirection:'column', gap:24 }}>
      {stats && (
        <div style={{ display:'grid', gridTemplateColumns:'repeat(4,1fr)', gap:16 }}>
          <StatCard title="Total Requests" value={stats.TotalRequests.toLocaleString()} sub="Last 24 hours" color="indigo"/>
          <StatCard title="Unique IPs" value={stats.UniqueIPs.toLocaleString()} sub="Distinct visitors" color="green"/>
          <StatCard title="Bandwidth" value={fmt(stats.TotalBytes)} sub="Data transferred" color="amber"/>
          <StatCard title="Error Rate" value={stats.ErrorRate.toFixed(1)+'%'} sub="4xx + 5xx" color={stats.ErrorRate>10?'red':'green'}/>
        </div>
      )}
      {history.length > 1 && (
        <div style={{ background:'#1e293b', borderRadius:12, padding:20, border:'1px solid #2d3748' }}>
          <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:12 }}>Request history (live)</div>
          <ResponsiveContainer width="100%" height={120}>
            <LineChart data={history}>
              <CartesianGrid strokeDasharray="3 3" stroke="#374151"/>
              <XAxis dataKey="time" tick={{ fill:'#6b7280', fontSize:10 }} interval="preserveStartEnd"/>
              <YAxis tick={{ fill:'#6b7280', fontSize:10 }} width={40}/>
              <Tooltip contentStyle={{ background:'#1f2937', border:'1px solid #374151', borderRadius:8 }}/>
              <Line type="monotone" dataKey="requests" stroke="#6366f1" strokeWidth={2} dot={false}/>
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
      <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:16 }}>
        <div style={{ background:'#1e293b', borderRadius:12, padding:20, border:'1px solid #2d3748' }}>
          <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:12 }}>Top paths</div>
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={paths} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" stroke="#374151" horizontal={false}/>
              <XAxis type="number" tick={{ fill:'#6b7280', fontSize:11 }}/>
              <YAxis type="category" dataKey="Key" tick={{ fill:'#d1d5db', fontSize:11 }} width={100}/>
              <Tooltip contentStyle={{ background:'#1f2937', border:'1px solid #374151', borderRadius:8 }}/>
              <Bar dataKey="Count" fill="#6366f1" radius={[0,4,4,0]}/>
            </BarChart>
          </ResponsiveContainer>
        </div>
        <div style={{ background:'#1e293b', borderRadius:12, padding:20, border:'1px solid #2d3748' }}>
          <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:12 }}>Status codes</div>
          <ResponsiveContainer width="100%" height={220}>
            <PieChart>
              <Pie data={codes} dataKey="Count" nameKey="Key" cx="50%" cy="50%" outerRadius={80}
                label={({ Key, percent }) => `${Key} (${(percent*100).toFixed(0)}%)`}>
                {codes.map((_, i) => <Cell key={i} fill={COLORS[i%COLORS.length]}/>)}
              </Pie>
              <Tooltip contentStyle={{ background:'#1f2937', border:'1px solid #374151', borderRadius:8 }}/>
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>
      <div style={{ background:'#1e293b', borderRadius:12, padding:20, border:'1px solid #2d3748' }}>
        <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:12 }}>Top IPs</div>
        <table style={{ width:'100%', borderCollapse:'collapse', fontSize:13 }}>
          <thead>
            <tr style={{ color:'#64748b', fontSize:11, textTransform:'uppercase', borderBottom:'1px solid #2d3748' }}>
              <th style={{ textAlign:'left', padding:'8px 12px' }}>#</th>
              <th style={{ textAlign:'left', padding:'8px 12px' }}>IP</th>
              <th style={{ textAlign:'right', padding:'8px 12px' }}>Requests</th>
              <th style={{ textAlign:'right', padding:'8px 12px' }}>Share</th>
            </tr>
          </thead>
          <tbody>
            {ips.map((ip, i) => {
              const total = ips.reduce((s,x) => s+x.Count, 0)
              const pct = ((ip.Count/total)*100).toFixed(1)
              return (
                <tr key={i} style={{ borderBottom:'1px solid #1e293b' }}>
                  <td style={{ padding:'10px 12px', color:'#64748b' }}>{i+1}</td>
                  <td style={{ padding:'10px 12px', fontFamily:'monospace', color:'#a5b4fc' }}>{ip.Key}</td>
                  <td style={{ padding:'10px 12px', textAlign:'right', fontWeight:600, color:'#f1f5f9' }}>{ip.Count.toLocaleString()}</td>
                  <td style={{ padding:'10px 12px', textAlign:'right' }}>
                    <div style={{ display:'flex', alignItems:'center', justifyContent:'flex-end', gap:8 }}>
                      <div style={{ width:64, background:'#1e293b', borderRadius:99, height:6 }}>
                        <div style={{ width:pct+'%', background:'#6366f1', height:6, borderRadius:99 }}/>
                      </div>
                      <span style={{ color:'#64748b', fontSize:11, width:36, textAlign:'right' }}>{pct}%</span>
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function SecurityPage({ secStats, threats, attackers }) {
  return (
    <div style={{ display:'flex', flexDirection:'column', gap:24 }}>
      <div style={{ display:'grid', gridTemplateColumns:'repeat(4,1fr)', gap:16 }}>
        <StatCard title="Total Threats" value={secStats?.TotalEvents??0} sub="Last 24 hours" color="red"/>
        <StatCard title="Critical" value={secStats?.CriticalEvents??0} sub="High severity" color="red"/>
        <StatCard title="Attackers" value={secStats?.UniqueAttackers??0} sub="Unique IPs" color="amber"/>
        <StatCard title="Top Threat" value={secStats?.TopThreatType||'—'} sub="Most common" color="indigo"/>
      </div>
      <div style={{ background:'#1e293b', borderRadius:12, padding:20, border:'1px solid #2d3748' }}>
        <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:16 }}>Recent threats (live)</div>
        {!threats?.length ? (
          <div style={{ textAlign:'center', padding:'40px 0' }}>
            <div style={{ fontSize:40, marginBottom:8 }}>🛡️</div>
            <div style={{ color:'#64748b' }}>No threats detected yet</div>
          </div>
        ) : (
          <div style={{ display:'flex', flexDirection:'column', gap:8, maxHeight:400, overflowY:'auto' }}>
            {threats.map((t, i) => (
              <div key={i} style={{ display:'flex', alignItems:'flex-start', gap:12, padding:12, borderRadius:8, ...Object.fromEntries((SEVERITY_COLORS[t.Severity]||SEVERITY_COLORS.low).split(';').map(s => { const [k,v]=s.split(':'); return [k?.trim().replace(/-([a-z])/g,(_,c)=>c.toUpperCase()),v?.trim()] }).filter(([k])=>k)) }}>
                <span style={{ fontSize:20 }}>{THREAT_ICONS[t.ThreatType]||'⚠️'}</span>
                <div style={{ flex:1 }}>
                  <div style={{ display:'flex', alignItems:'center', gap:8, flexWrap:'wrap' }}>
                    <span style={{ fontFamily:'monospace', fontWeight:600, fontSize:13 }}>{t.IP}</span>
                    <span style={{ fontSize:11, padding:'2px 8px', borderRadius:99, background:'rgba(0,0,0,0.2)', border:'1px solid currentColor' }}>{t.ThreatType?.replace('_',' ')}</span>
                    <span style={{ fontSize:11, padding:'2px 8px', borderRadius:99, background:'rgba(0,0,0,0lid currentColor', textTransform:'uppercase' }}>{t.Severity}</span>
                    <span style={{ fontSize:11, opacity:0.6, marginLeft:'auto' }}>score: {t.Score}</span>
                  </div>
                  <div style={{ fontSize:12, opacity:0.7, marginTop:4, fontFamily:'monospace' }}>{t.Path}</div>
                  <div style={{ fontSize:12, opacity:0.5, marginTop:2 }}>{t.Description}</div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      {attackers?.length > 0 && (
        <div style={{ background:'#1e293b', borderRadius:12, padding:20, border:'1px solid #2d3748' }}>
          <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:12 }}>Top attackers</div>
          {attackers.map((a, i) => (
            <div key={i} style={{ display:'flex', alignItems:'center', gap:12, marginBottom:10 }}>
              <span style={{ color:'#64748b', fontSize:13, width:20 }}>{i+1}</span>
              <span style={{ fontFamily:'monospace', color:'#f87171', fontSize:13, flex:1 }}>{a.Key}</span>
              <div style={{ width:96, background:'#0f172a', borderRadius:99, height:6 }}>
                <div style={{ width:Math.min((a.Count/(attackers[0]?.Count||1))*100,100)+'%', background:'#ef4444', height:6, borderRadius:99 }}/>
              </div>
              <span style={{ color:'#f87171', fontWeight:700, fontSize:13, width:48, textAlign:'right' }}>{a.Count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function UsersPage({ currentUser }) {
  const [users, setUsers] = useState([])
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ username:'', password:'', role:'viewer' })
  const [msg, setMsg] = useState('')

  useEffect(() => { loadUsers() }, [])

  const loadUsers = async () => {
    try { const data = await listUsers(); setUsers(data||[]) } catch {}
  }

  const handleCreate = async (e) => {
    e.preventDefault()
    try {
      await createUser(form.username, form.password, form.role)
      setForm({ username:'', password:'', role:'viewer' })
      setShowForm(false)
      setMsg('User created!')
      loadUsers()
      setTimeout(() => setMsg(''), 3000)
    } catch { setMsg('Error creating user') }
  }

  const handleDelete = async (id, uname) => {
    if (uname === currentUser?.username) { setMsg("Aap khud ko delete nahi kar sakte!"); setTimeout(()=>setMsg(''),3000); return }
    if (!confirm(`Delete user "${uname}"?`)) return
    try { await deleteUser(id); loadUsers() } catch {}
  }

  return (
    <div style={{ display:'flex', flexDirection:'column', gap:24 }}>
      <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div style={{ color:'#f1f5f9', fontWeight:700, fontSize:18 }}>User Management</div>
        {currentUser?.role === 'admin' && (
          <button onClick={() => setShowForm(s => !s)}
            style={{ background:'#6366f1', border:'none', borderRadius:8, padding:'8px 16px', color:'white', fontSize:13, fontWeight:600, cursor:'pointer' }}>
            + Add User
          </button>
        )}
      </div>

      {msg && <div style={{ background:'rgba(99,102,241,0.15)', border:'1px solid rgba(99,102,241,0.3)', color:'#a5b4fc', borderRadius:8, padding:'10px 14px', fontSize:13 }}>{msg}</div>}

      {showForm && (
        <form onSubmit={handleCreate} style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:12, padding:24, display:'flex', flexDirection:'column', gap:16 }}>
          <div style={{ color:'#f1f5f9', fontWeight:600, fontSize:15, marginBottom:4 }}>Create new user</div>
          <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr 1fr', gap:12 }}>
            <div>
              <label style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', display:'block', marginBottom:6 }}>Username</label>
              <input value={form.username} onChange={e=>setForm(f=>({...f,username:e.target.value}))} required
                style={{ width:'100%', background:'#0f172a', border:'1px solid #2d3748', borderRadius:8, padding:'8px 12px', color:'#e2e8f0', fontSize:13, outline:'none', boxSizing:'border-box' }}/>
            </div>
            <div>
              <label style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', display:'block', marginBottom:6 }}>Password</label>
              <input type="password" value={form.password} onChange={e=>setForm(f=>({...f,password:e.target.value}))} required
                style={{ width:'100%', background:'#0f172a', border:'1px solid #2d3748', borderRadius:8, padding:'8px 12px', color:'#e2e8f0', fontSize:13, outline:'none', boxSizing:'border-box' }}/>
            </div>
            <div>
              <label style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', display:'block', marginBottom:6 }}>Role</label>
              <select value={form.role} onChange={e=>setForm(f=>({...f,role:e.target.value}))}
                style={{ width:'100%', background:'#0f172a', border:'1px solid #2d3748', borderRadius:8, padding:'8px 12px', color:'#e2e8f0', fontSize:13, outline:'none', boxSizing:'border-box' }}>
                <option value="viewer">Viewer</option>
                <option value="admin">Admin</option>
              </select>
            </div>
          </div>
          <div style={{ display:'flex', gap:10 }}>
            <button type="submit" style={{ background:'#6366f1', border:'none', borderRadius:8, padding:'8px 20px', color:'white', fontSize:13, fontWeight:600, cursor:'pointer' }}>Create</button>
            <button type="button" onClick={()=>setShowForm(false)} style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:8, padding:'8px 20px', color:'#94a3b8', fontSize:13, cursor:'pointer' }}>Cancel</button>
          </div>
        </form>
      )}

      <div style={{ background:'#1e293b', borderRadius:12, border:'1px solid #2d3748', overflow:'hidden' }}>
        <table style={{ width:'100%', borderCollapse:'collapse', fontSize:13 }}>
          <thead>
            <tr style={{ background:'#0f172a', color:'#64748b', fontSize:11, textTransform:'uppercase', letterSpacing:1 }}>
              <th style={{ textAlign:'left', padding:'12px 16px' }}>Username</th>
              <th style={{ textAlign:'left', padding:'12px 16px' }}>Role</th>
              <th style={{ textAlign:'left', padding:'12px 16px' }}>Created</th>
              <th style={{ textAlign:'right', padding:'12px 16px' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {(users||[]).map((u, i) => (
              <tr key={i} style={{ borderBottom:'1px solid #1e293b' }}>
                <td style={{ padding:'12px 16px', color:'#e2e8f0', fontWeight:500 }}>
                  {u.Username}
                  {u.Username === currentUser?.username && <span style={{ marginLeft:8, fontSize:10, background:'rgba(99,102,241,0.2)', color:'#a5b4fc', padding:'2px 6px', borderRadius:4 }}>you</span>}
                </td>
                <td style={{ padding:'12px 16px' }}>
                  <span style={{ fontSize:11, padding:'3px 10px', borderRadius:99, background: u.Role==='admin'?'rgba(239,68,68,0.15)':'rgba(99,102,241,0.15)', color: u.Role==='admin'?'#f87171':'#a5b4fc', border: u.Role==='admin'?'1px solid rgba(239,68,68,0.3)':'1px solid rgba(99,102,241,0.3)' }}>
                    {u.Role}
                  </span>
                </td>
                <td style={{ padding:'12px 16px', color:'#64748b', fontSize:12 }}>
                  {u.CreatedAt ? new Date(u.CreatedAt).toLocaleDateString() : '—'}
                </td>
                <td style={{ padding:'12px 16px', textAlign:'right' }}>
                  {currentUser?.role === 'admin' && u.Username !== currentUser?.username && (
                    <button onClick={() => handleDelete(u.ID, u.Username)}
                      style={{ background:'rgba(239,68,68,0.15)', border:'1px solid rgba(239,68,68,0.3)', borderRadius:6, padding:'4px 12px', color:'#f87171', fontSize:12, cursor:'pointer' }}>
                      Delete
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default function App() {
  const [user, setUser] = useState(() => {
    try { return JSON.parse(localStorage.getItem('user')) } catch { return null }
  })
  const [tab, setTab] = useState('dashboard')
  const [stats, setStats] = useState(null)
  const [paths, setPaths] = useState([])
  const [ips, setIPs] = useState([])
  const [codes, setCodes] = useState([])
  const [history, setHistory] = useState([])
  const [secStats, setSecStats] = useState(null)
  const [threats, setThreats] = useState([])
  const [attackers, setAttackers] = useState([])
  const [lastUpdate, setLastUpdate] = useState(null)
  const [loading, setLoading] = useState(true)

  const { data: wsData, connected } = useWebSocket('ws://127.0.0.1:8080/api/v1/ws')

  const fetchAll = useCallback(async () => {
    const [sr, pr, ir, cr, ssr, thr, atr] = await Promise.allSettled([
      getStats(), getTopPaths(), getTopIPs(), getStatusCodes(),
      getSecurityStats(), getRecentThreats(), getTopAttackers()
    ])
    if (sr.status==='fulfilled') { setStats(sr.value); setHistory(prev => [...prev, { time: new Date().toLocaleTimeString(), requests: sr.value?.TotalRequests??0 }].slice(-20)) }
    if (pr.status==='fulfilled') setPaths((pr.value||[]).slice(0,8))
    if (ir.status==='fulfilled') setIPs((ir.value||[]).slice(0,8))
    if (cr.status==='fulfilled') setCodes(cr.value||[])
    if (ssr.status==='fulfilled') setSecStats(ssr.value)
  if (thr.status==='fulfilled') setThreats(thr.value||[])
    if (atr.status==='fulfilled') setAttackers(atr.value||[])
    setLastUpdate(new Date().toLocaleTimeString())
    setLoading(false)
  }, [])

  useEffect(() => {
    if (!user) return
    fetchAll()
  }, [user, fetchAll])

  // Update from WebSocket
  useEffect(() => {
    if (wsData.stats) {
      setStats(wsData.stats)
      setHistory(prev => [...prev, { time: new Date().toLocaleTimeString(), requests: wsData.stats?.TotalRequests??0 }].slice(-20))
      setLastUpdate(new Date().toLocaleTimeString())
    }
    if (wsData.top_paths) setPaths((wsData.top_paths||[]).slice(0,8))
    if (wsData.top_ips) setIPs((wsData.top_ips||[]).slice(0,8))
    if (wsData.status_codes) setCodes(wsData.status_codes||[])
  }, [wsData])

  const handleLogin = (u) => { setUser(u); setLoading(true) }

  const handleLogout = async () => {
    try { await logout() } catch {}
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setUser(null)
  }

  if (!user) return <Login onLogin={handleLogin}/>

  if (loading) return (
    <div style={{ minHeight:'100vh', background:'#0f1117', display:'flex', alignItems:'center', justifyContent:'center' }}>
      <div style={{ textAlign:'center' }}>
        <div style={{ width:48, height:48, border:'4px solid #6366f1', borderTopColor:'transparent', borderRadius:'50%', animation:'spin 0.8s linear infinite', margin:'0 auto 16px' }}/>
        <p style={{ color:'#64748b' }}>Loading dashboard...</p>
      </div>
      <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
    </div>
  )

  const threatCount = threats?.length ?? 0

  return (
    <div style={{ minHeight:'100vh', background:'#0f1117', color:'#e2e8f0' }}>
      <header style={{ background:'#1e293b', borderBottom:'1px solid #2d3748', padding:'0 24px', display:'flex', alignItems:'center', justifyContent:'space-between', position:'sticky', top:0, zIndex:10, height:56 }}>
        <div style={{ display:'flex', alignItems:'center', gap:12 }}>
          <div style={{ width:32, height:32, background:'#6366f1', borderRadius:8, display:'flex', alignItems:'center', justifyContent:'center', color:'white', fontWeight:800, fontSize:13 }}>LW</div>
          <div>
            <div style={{ color:'white', fontWeight:700, fontSize:15, lineHeight:1 }}>LogWatch</div>
            <div style={{ color:'#64748b', fontSize:11 }}>Real-time VPS log analyzer</div>
          </div>
        </div>
        <div style={{ display:'flex', alignItems:'center', gap:16 }}>
          {lastUpdate && <div style={{ display:'flex', alignItems:'center', gap:6 }}>
            <div style={{ width:8, height:8, background:'#22c55e', borderRadius:'50%', animation:'pulse 2s infinite' }}/>
            <span style={{ color:'#64748b', fontSize:12 }}>Live · {lastUpdate}</span>
          </div>}
          <div style={{ display:'flex', alignItems:'center', gap:8 }}>
            <span style={{ color:'#94a3b8', fontSize:12 }}>{user?.username}</span>
            <span style={{ fontSize:11, padding:'2px 8px', borderRadius:99, background: user?.role==='admin'?'rgba(239,68,68,0.15)':'rgba(99,102,241,0.15)', color: user?.role==='admin'?'#f87171':'#a5b4fc', border: user?.role==='admin'?'1px solid rgba(239,68,68,0.3)':'1px solid rgba(99,102,241,0.3)' }}>{user?.role}</span>
            <button onClick={handleLogout} style={{ background:'transparent', border:'1px solid #2d3748', borderRadius:6, padding:'4px 10px', color:'#64748b', fontSize:12, cursor:'pointer' }}>Logout</button>
          </div>
        </div>
      </header>

      <div style={{ background:'#1e293b', borderBottom:'1px solid #2d3748', padding:'0 24px' }}>
        <nav style={{ display:'flex', gap:4 }}>
          {[
            { id:'dashboard', label:'Dashboard', icon:'📊' },
            { id:'security', label:'Security', icon:'🛡️', badge: threatCount>0?threatCount:null },
            { id:'users', label:'Users', icon:'👥', adminOnly:true },
            { id:'reports', label:'Reports', icon:'📄' },
            { id:'map', label:'World Map', icon:'🌍' },
          ].filter(t => !t.adminOnly || user?.role==='admin').map(t => (
            <button key={t.id} onClick={() => setTab(t.id)}
              style={{ display:'flex', alignItems:'center', gap:6, padding:'12px 16px', fontSize:13, fontWeight:500, border:'none', background:'transparent', cursor:'pointer', borderBottom: tab===t.id?'2px solid #6366f1':'2px solid transparent', color: tab===t.id?'#a5b4fc':'#64748b', transition:'color 0.2s', position:'relative' }}>
              <span>{t.icon}</span>
              {t.label}
              {t.badge && <span style={{ background:'#ef4444', color:'white', fontSize:10, borderRadius:'50%', width:18, height:18, display:'flex', alignItems:'center', justifyContent:'center', fontWeight:700 }}>{t.badge>99?'99+':t.badge}</span>}
            </button>
          ))}
        </nav>
      </div>

      <main style={{ maxWidth:1200, margin:'0 auto', padding:24 }}>
        {tab==='dashboard' && <Dashboard stats={stats} paths={paths} ips={ips} codes={codes} history={history}/>}
        {tab==='security'  && <SecurityPage secStats={secStats} threats={threats} attackers={attackers}/>}
        {tab==='users'     && <UsersPage currentUser={user}/>}
        {tab==='reports'   && <Reports/>}
        {tab==='map'       && <WorldMap/>}
      </main>

      <style>{`@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.5} } @keyframes spin { to{transform:rotate(360deg)} }`}</style>
    </div>
  )
}
