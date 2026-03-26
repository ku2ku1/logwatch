import { useState } from 'react'

const BASE = ''

function download(url, filename) {
  const token = localStorage.getItem('token')
  fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    .then(r => r.blob())
    .then(blob => {
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = filename
      a.click()
      URL.revokeObjectURL(a.href)
    })
}

function ExportCard({ title, desc, icon, actions }) {
  return (
    <div style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:12, padding:24 }}>
      <div style={{ display:'flex', alignItems:'center', gap:12, marginBottom:12 }}>
        <span style={{ fontSize:28 }}>{icon}</span>
        <div>
          <div style={{ color:'#f1f5f9', fontWeight:700, fontSize:15 }}>{title}</div>
          <div style={{ color:'#64748b', fontSize:12, marginTop:2 }}>{desc}</div>
        </div>
      </div>
      <div style={{ display:'flex', flexWrap:'wrap', gap:8, marginTop:16 }}>
        {actions.map((a, i) => (
          <button key={i} onClick={a.onClick}
            style={{ background: a.primary?'#6366f1':'#0f172a', border:`1px solid ${a.primary?'#6366f1':'#2d3748'}`, borderRadius:8, padding:'8px 16px', color: a.primary?'white':'#94a3b8', fontSize:13, fontWeight:500, cursor:'pointer', display:'flex', alignItems:'center', gap:6 }}>
            <span style={{ fontSize:14 }}>{a.icon}</span>
            {a.label}
          </button>
        ))}
      </div>
    </div>
  )
}

export default function Reports() {
  const [range, setRange] = useState('24h')
  const [pdfLoading, setPdfLoading] = useState(false)
  const [pdfStatus, setPdfStatus] = useState('')

  const rangeParam = { '24h':'24h', '7d':'7d', '30d':'30d', 'all':'all' }[range]

  const generatePDF = async () => {
    setPdfLoading(true)
    setPdfStatus('Generating PDF...')
    try {
      const token = localStorage.getItem('token')
      const [statsR, pathsR, ipsR, codesR, secR, threatsR] = await Promise.all([
        fetch(`${BASE}/api/v1/stats`, { headers:{ Authorization:`Bearer ${token}` } }).then(r=>r.json()),
        fetch(`${BASE}/api/v1/top/paths`, { headers:{ Authorization:`Bearer ${token}` } }).then(r=>r.json()),
        fetch(`${BASE}/api/v1/top/ips`, { headers:{ Authorization:`Bearer ${token}` } }).then(r=>r.json()),
        fetch(`${BASE}/api/v1/status-codes`, { headers:{ Authorization:`Bearer ${token}` } }).then(r=>r.json()),
        fetch(`${BASE}/api/v1/security/stats`, { headers:{ Authorization:`Bearer ${token}` } }).then(r=>r.json()),
        fetch(`${BASE}/api/v1/security/threats`, { headers:{ Authorization:`Bearer ${token}` } }).then(r=>r.json()),
      ])

      const fmt = n => n>=1e9?(n/1e9).toFixed(1)+'GB':n>=1e6?(n/1e6).toFixed(1)+'MB':n>=1e3?(n/1e3).toFixed(1)+'KB':n+'B'
      const now = new Date().toLocaleString()

      const html = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Logvance Report — ${now}</title>
<style>
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family: Arial, sans-serif; background:#fff; color:#1e293b; padding:40px; }
  .header { display:flex; justify-content:space-between; align-items:center; border-bottom:3px solid #6366f1; padding-bottom:20px; margin-bottom:32px; }
  .logo { display:flex; align-items:center; gap:12px; }
  .logo-badge { width:44px; height:44px; background:#6366f1; border-radius:10px; display:flex; align-items:center; justify-content:center; color:white; font-weight:800; font-size:16px; }
  .logo-text h1 { font-size:22px; font-weight:800; color:#1e293b; }
  .logo-text p { font-size:12px; color:#64748b; }
  .meta { text-align:right; font-size:12px; color:#64748b; }
  .section { margin-bottom:32px; }
  .section-title { font-size:14px; font-weight:700; color:#64748b; text-transform:uppercase; letter-spacing:1px; margin-bottom:16px; padding-bottom:8px; border-bottom:1px solid #e2e8f0; }
  .stats-grid { display:grid; grid-template-columns:repeat(4,1fr); gap:16px; }
  .stat-card { background:#f8fafc; border:1px solid #e2e8f0; border-radius:10px; padding:16px; border-left:4px solid #6366f1; }
  .stat-card.red { border-left-color:#ef4444; }
  .stat-card.green { border-left-color:#22c55e; }
  .stat-card.amber { border-left-color:#f59e0b; }
  .stat-label { font-size:11px; font-weight:600; color:#64748b; text-transform:uppercase; letter-spacing:0.5px; }
  .stat-value { font-size:24px; font-weight:800; color:#1e293b; margin-top:4px; }
  table { width:100%; border-collapse:collapse; font-size:13px; }
  th { background:#f1f5f9; padding:10px 12px; text-align:left; font-size:11px; font-weight:600; color:#64748b; text-transform:uppercase; }
  td { padding:10px 12px; border-bottom:1px solid #f1f5f9; }
  tr:last-child td { border-bottom:none; }
  .badge { display:inline-block; padding:2px 8px; border-radius:99px; font-size:11px; font-weight:600; }
  .badge-red { background:#fef2f2; color:#ef4444; }
  .badge-amber { background:#fffbeb; color:#d97706; }
  .badge-blue { background:#eff6ff; color:#3b82f6; }
  .badge-gray { background:#f8fafc; color:#64748b; }
  .footer { margin-top:40px; padding-top:16px; border-top:1px solid #e2e8f0; text-align:center; font-size:11px; color:#94a3b8; }
  @media print { body { padding:20px; } }
</style>
</head>
<body>
<div class="header">
  <div class="logo">
    <div class="logo-badge">LW</div>
    <div class="logo-text">
      <h1>Logvance Report</h1>
      <p>Real-time VPS Log Analyzer</p>
    </div>
  </div>
  <div class="meta">
    <div><strong>Generated:</strong> ${now}</div>
    <div><strong>Range:</strong> ${range}</div>
  </div>
</div>

<div class="section">
  <div class="section-title">Traffic Overview</div>
  <div class="stats-grid">
    <div class="stat-card"><div class="stat-label">Total Requests</div><div class="stat-value">${statsR?.TotalRequests?.toLocaleString()??0}</div></div>
    <div class="stat-card green"><div class="stat-label">Unique IPs</div><div class="stat-value">${statsR?.UniqueIPs?.toLocaleString()??0}</div></div>
    <div class="stat-card amber"><div class="stat-label">Bandwidth</div><div class="stat-value">${fmt(statsR?.TotalBytes??0)}</div></div>
    <div class="stat-card red"><div class="stat-label">Error Rate</div><div class="stat-value">${(statsR?.ErrorRate??0).toFixed(1)}%</div></div>
  </div>
</div>

<div class="section">
  <div class="section-title">Security Overview</div>
  <div class="stats-grid">
    <div class="stat-card red"><div class="stat-label">Total Threats</div><div class="stat-value">${secR?.TotalEvents??0}</div></div>
    <div class="stat-card red"><div class="stat-label">Critical</div><div class="stat-value">${secR?.CriticalEvents??0}</div></div>
    <div class="stat-card amber"><div class="stat-label">Attackers</div><div class="stat-value">${secR?.UniqueAttackers??0}</div></div>
    <div class="stat-card"><div class="stat-label">Top Threat</div><div class="stat-value" style="font-size:14px">${secR?.TopThreatType||'—'}</div></div>
  </div>
</div>

<div class="section">
  <div class="section-title">Top Paths</div>
  <table>
    <tr><th>#</th><th>Path</th><th>Requests</th></tr>
    ${(pathsR||[]).slice(0,10).map((p,i)=>`<tr><td>${i+1}</td><td>${p.Key}</td><td><strong>${p.Count}</strong></td></tr>`).join('')}
  </table>
</div>

<div class="section">
  <div class="section-title">Top IPs</div>
  <table>
    <tr><th>#</th><th>IP Address</th><th>Requests</th></tr>
    ${(ipsR||[]).slice(0,10).map((ip,i)=>`<tr><td>${i+1}</td><td style="font-family:monospace">${ip.Key}</td><td><strong>${ip.Count}</strong></td></tr>`).join('')}
  </table>
</div>

<div class="section">
  <div class="section-title">Status Codes</div>
  <table>
    <tr><th>Code</th><th>Count</th></tr>
    ${(codesR||[]).map(c=>`<tr><td><span class="badge ${c.Key.startsWith('5')?'badge-red':c.Key.startsWith('4')?'badge-amber':c.Key.startsWith('2')?'badge-blue':'badge-gray'}">${c.Key}</span></td><td><strong>${c.Count}</strong></td></tr>`).join('')}
  </table>
</div>

<div class="section">
  <div class="section-title">Recent Security Threats</div>
  <table>
    <tr><th>IP</th><th>Path</th><th>Type</th><th>Severity</th><th>Score</th></tr>
    ${(threatsR||[]).slice(0,15).map(t=>`
    <tr>
      <td style="font-family:monospace">${t.IP}</td>
      <td style="font-family:monospace;font-size:11px">${t.Path}</td>
      <td>${t.ThreatType?.replace('_',' ')}</td>
      <td><span class="badge ${t.Severity==='critical'?'badge-red':t.Severity==='high'?'badge-amber':'badge-blue'}">${t.Severity}</span></td>
      <td><strong>${t.Score}</strong></td>
    </tr>`).join('')}
  </table>
</div>

<div class="footer">
  Logvance — Self-hosted VPS Log Analyzer · Report generated on ${now}
</div>
</body>
</html>`

      const win = window.open('', '_blank')
      win.document.write(html)
      win.document.close()
      setTimeout(() => { win.print(); setPdfStatus('Done!'); setTimeout(()=>setPdfStatus(''),3000) }, 500)
    } catch (e) {
      setPdfStatus('Error generating report')
    }
    setPdfLoading(false)
  }

  return (
    <div style={{ display:'flex', flexDirection:'column', gap:24 }}>
      <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div style={{ color:'#f1f5f9', fontWeight:700, fontSize:18 }}>Reports & Export</div>
        <div style={{ display:'flex', alignItems:'center', gap:8 }}>
          <span style={{ color:'#64748b', fontSize:13 }}>Range:</span>
          {['24h','7d','30d','all'].map(r => (
            <button key={r} onClick={() => setRange(r)}
              style={{ background: range===r?'#6366f1':'#1e293b', border:`1px solid ${range===r?'#6366f1':'#2d3748'}`, borderRadius:6, padding:'5px 12px', color: range===r?'white':'#64748b', fontSize:12, cursor:'pointer' }}>
              {r}
            </button>
          ))}
        </div>
      </div>

      {pdfStatus && (
        <div style={{ background:'rgba(99,102,241,0.15)', border:'1px solid rgba(99,102,241,0.3)', color:'#a5b4fc', borderRadius:8, padding:'10px 16px', fontSize:13 }}>
          {pdfStatus}
        </div>
      )}

      <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:16 }}>
        <ExportCard
          title="PDF Report"
          icon="📄"
          desc="Complete report with all metrics, charts summary, and security threats"
          actions={[
            { label: pdfLoading?'Generating...':'Generate PDF', icon:'📥', primary:true, onClick: generatePDF }
          ]}
        />

        <ExportCard
          title="JSON Export"
          icon="🗂️"
          desc="Full data export in JSON format — all metrics, IPs, paths, threats"
          actions={[
            { label:'Export JSON', icon:'⬇️', primary:true,
              onClick: () => download(`${BASE}/api/v1/export/json?range=${rangeParam}`, `logvance-${rangeParam}.json`) }
          ]}
        />

        <ExportCard
          title="CSV — Top Paths"
          icon="📊"
          desc="Top requested paths with request counts"
          actions={[
            { label:'Download CSV', icon:'⬇️', primary:true,
              onClick: () => download(`${BASE}/api/v1/export/csv?type=paths&range=${rangeParam}`, `logvance-paths-${rangeParam}.csv`) }
          ]}
        />

        <ExportCard
          title="CSV — Top IPs"
          icon="🌐"
          desc="Top IP addresses with request counts"
          actions={[
            { label:'Download CSV', icon:'⬇️', primary:true,
              onClick: () => download(`${BASE}/api/v1/export/csv?type=ips&range=${rangeParam}`, `logvance-ips-${rangeParam}.csv`) }
          ]}
        />

        <ExportCard
          title="CSV — Security Threats"
          icon="🛡️"
          desc="All detected threats with IP, path, type, severity, score"
          actions={[
            { label:'Download CSV', icon:'⬇️', primary:true,
              onClick: () => download(`${BASE}/api/v1/export/csv?type=threats&range=${rangeParam}`, `logvance-threats-${rangeParam}.csv`) }
          ]}
        />

        <ExportCard
          title="JSON — Threats Only"
          icon="⚠️"
          desc="Security threats export only — for SIEM integration"
          actions={[
            { label:'Export Threats', icon:'⬇️', primary:true,
              onClick: () => download(`${BASE}/api/v1/export/json?range=${rangeParam}`, `logvance-threats-${rangeParam}.json`) }
          ]}
        />
      </div>

      <div style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:12, padding:20 }}>
        <div style={{ color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1, marginBottom:12 }}>API endpoints</div>
        <div style={{ display:'flex', flexDirection:'column', gap:8 }}>
          {[
            { method:'GET', url:'/api/v1/export/json?range=24h', desc:'Full JSON export' },
            { method:'GET', url:'/api/v1/export/csv?type=paths', desc:'Top paths CSV' },
            { method:'GET', url:'/api/v1/export/csv?type=ips', desc:'Top IPs CSV' },
            { method:'GET', url:'/api/v1/export/csv?type=threats', desc:'Threats CSV' },
          ].map((e, i) => (
            <div key={i} style={{ display:'flex', alignItems:'center', gap:12, padding:'8px 12px', background:'#0f172a', borderRadius:8 }}>
              <span style={{ background:'rgba(99,102,241,0.2)', color:'#a5b4fc', fontSize:11, fontWeight:700, padding:'2px 8px', borderRadius:4, whiteSpace:'nowrap' }}>{e.method}</span>
              <code style={{ color:'#22d3ee', fontSize:12, flex:1 }}>{e.url}</code>
              <span style={{ color:'#475569', fontSize:12 }}>{e.desc}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
