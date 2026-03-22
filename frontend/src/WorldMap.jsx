import { useState, useEffect } from 'react'
import { ComposableMap, Geographies, Geography, Marker, ZoomableGroup } from 'react-simple-maps'

const BASE = ''
const GEO_URL = 'https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json'

export default function WorldMap() {
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(true)
  const [tooltip, setTooltip] = useState(null)

  useEffect(() => {
    const token = localStorage.getItem('token')
    fetch(`${BASE}/api/v1/geo/map`, { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(d => { setData(d || []); setLoading(false) })
      .catch(() => setLoading(false))
  }, [])

  const maxCount = Math.max(...(data.map(d => d.count) || [1]))

  const getRadius = (count) => {
    const min = 4, max = 24
    return min + ((count / maxCount) * (max - min))
  }

  const getColor = (count) => {
    const ratio = count / maxCount
    if (ratio > 0.7) return '#ef4444'
    if (ratio > 0.4) return '#f59e0b'
    if (ratio > 0.1) return '#6366f1'
    return '#22d3ee'
  }

  return (
    <div style={{ display:'flex', flexDirection:'column', gap:24 }}>
      <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div style={{ color:'#f1f5f9', fontWeight:700, fontSize:18 }}>World Map — Visitor Origins</div>
        <div style={{ display:'flex', alignItems:'center', gap:16, fontSize:12, color:'#64748b' }}>
          <div style={{ display:'flex', alignItems:'center', gap:6 }}>
            <div style={{ width:10, height:10, borderRadius:'50%', background:'#22d3ee' }}/>Low
          </div>
          <div style={{ display:'flex', alignItems:'center', gap:6 }}>
            <div style={{ width:10, height:10, borderRadius:'50%', background:'#6366f1' }}/>Medium
          </div>
          <div style={{ display:'flex', alignItems:'center', gap:6 }}>
            <div style={{ width:10, height:10, borderRadius:'50%', background:'#f59e0b' }}/>High
          </div>
          <div style={{ display:'flex', alignItems:'center', gap:6 }}>
            <div style={{ width:10, height:10, borderRadius:'50%', background:'#ef4444' }}/>Critical
          </div>
        </div>
      </div>

      <div style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:12, overflow:'hidden', position:'relative' }}>
        {loading ? (
          <div style={{ height:420, display:'flex', alignItems:'center', justifyContent:'center' }}>
            <div style={{ color:'#64748b' }}>Loading map...</div>
          </div>
        ) : (
          <>
            {tooltip && (
              <div style={{ position:'absolute', top:16, left:16, background:'#0f172a', border:'1px solid #2d3748', borderRadius:8, padding:'10px 14px', zIndex:10, minWidth:160 }}>
                <div style={{ color:'#f1f5f9', fontWeight:700, fontSize:13 }}>{tooltip.country}</div>
                <div style={{ color:'#6366f1', fontSize:12, marginTop:4 }}>{tooltip.count.toLocaleString()} requests</div>
                {tooltip.city && <div style={{ color:'#64748b', fontSize:11, marginTop:2 }}>{tooltip.city}</div>}
              </div>
            )}
            <ComposableMap
              style={{ width:'100%', height:'420px', background:'#0f1929' }}
              projectionConfig={{ scale: 147 }}
            >
              <ZoomableGroup>
                <Geographies geography={GEO_URL}>
                  {({ geographies }) =>
                    geographies.map(geo => (
                      <Geography
                        key={geo.rsmKey}
                        geography={geo}
                        fill="#1e3a5f"
                        stroke="#0f172a"
                        strokeWidth={0.5}
                        style={{
                          default: { outline:'none' },
                          hover: { fill:'#2d5a8e', outline:'none' },
                          pressed: { outline:'none' },
                        }}
                      />
                    ))
                  }
                </Geographies>
                {data.map((d, i) => (
                  d.lat && d.lon ? (
                    <Marker key={i} coordinates={[d.lon, d.lat]}>
                      <circle
                        r={getRadius(d.count)}
                        fill={getColor(d.count)}
                        fillOpacity={0.8}
                        stroke="white"
                        strokeWidth={1}
                        style={{ cursor:'pointer' }}
                        onMouseEnter={() => setTooltip({ country: d.country, city: d.city, count: d.count })}
                        onMouseLeave={() => setTooltip(null)}
                      />
                    </Marker>
                  ) : null
                ))}
              </ZoomableGroup>
            </ComposableMap>
          </>
        )}
      </div>

      {/* Country table */}
      <div style={{ background:'#1e293b', border:'1px solid #2d3748', borderRadius:12, overflow:'hidden' }}>
        <div style={{ padding:'16px 20px', borderBottom:'1px solid #2d3748', color:'#94a3b8', fontSize:11, fontWeight:600, textTransform:'uppercase', letterSpacing:1 }}>
          Requests by country
        </div>
        {data.length === 0 ? (
          <div style={{ padding:'32px', textAlign:'center', color:'#64748b' }}>
            No geolocation data — requests may be from localhost (127.0.0.1 / ::1)
          </div>
        ) : (
          <table style={{ width:'100%', borderCollapse:'collapse', fontSize:13 }}>
            <thead>
              <tr style={{ background:'#0f172a', color:'#64748b', fontSize:11, textTransform:'uppercase' }}>
                <th style={{ textAlign:'left', padding:'10px 16px' }}>#</th>
                <th style={{ textAlign:'left', padding:'10px 16px' }}>Country</th>
                <th style={{ textAlign:'right', padding:'10px 16px' }}>Requests</th>
                <th style={{ textAlign:'right', padding:'10px 16px' }}>Share</th>
              </tr>
            </thead>
            <tbody>
              {[...data].sort((a,b) => b.count-a.count).slice(0,15).map((d, i) => {
                const total = data.reduce((s,x) => s+x.count, 0)
                const pct = ((d.count/total)*100).toFixed(1)
                return (
                  <tr key={i} style={{ borderBottom:'1px solid #1e293b' }}>
                    <td style={{ padding:'10px 16px', color:'#64748b' }}>{i+1}</td>
                    <td style={{ padding:'10px 16px', color:'#e2e8f0' }}>
                      <span style={{ marginRight:8 }}>{d.country_code}</span>
                      {d.country}
                    </td>
                    <td style={{ padding:'10px 16px', textAlign:'right', fontWeight:600, color:'#f1f5f9' }}>{d.count.toLocaleString()}</td>
                    <td style={{ padding:'10px 16px', textAlign:'right' }}>
                      <div style={{ display:'flex', alignItems:'center', justifyContent:'flex-end', gap:8 }}>
                        <div style={{ width:64, background:'#0f172a', borderRadius:99, height:4 }}>
                          <div style={{ width:pct+'%', background:'#6366f1', height:4, borderRadius:99 }}/>
                        </div>
                        <span style={{ color:'#64748b', fontSize:11, width:36 }}>{pct}%</span>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
