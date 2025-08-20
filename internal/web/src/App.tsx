import React, { useEffect, useMemo, useState } from 'react'
import SourcesConfig from './components/SourcesConfig'
import TransformWorkbench from './components/TransformWorkbench'
import DestinationsConfig from './components/DestinationsConfig'

type Info = {
  host: string
  port: number
  http_addr: string
  tls_enabled: boolean
  tls_min: string
}

function Stat({label, value}: {label: string; value: React.ReactNode}) {
  return (
    <div className="card stat">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
    </div>
  )
}

function useFetcher<T>(url: string, intervalMs?: number) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState<boolean>(false)
  const fetcher = useMemo(() => async () => {
    try {
      setLoading(true)
      setError(null)
      const r = await fetch(url)
      if (!r.ok) throw new Error(`${r.status}`)
      const json = await r.json()
      const unwrapped: any = Array.isArray(json) ? json : (Array.isArray(json?.items) ? json.items : json)
      setData(unwrapped)
    } catch (e: any) {
      setError(e?.message || 'error')
    } finally {
      setLoading(false)
    }
  }, [url])
  useEffect(() => {
    fetcher()
    if (!intervalMs) return
    const id = setInterval(fetcher, intervalMs)
    return () => clearInterval(id)
  }, [fetcher, intervalMs])
  return { data, error, loading, refresh: fetcher }
}

export default function App() {
  const health = useFetcher<{status: string}>(`/api/v1/health`, 5000)
  const version = useFetcher<{version: string}>(`/api/v1/version`, 30000)
  const info = useFetcher<Info>(`/api/v1/info`, 60000)
  const [tab, setTab] = useState<'home'|'sources'|'transform'|'destinations'|'logevents'|'azure'>('home')
  const [sourcesSummary, setSourcesSummary] = useState<{total:number; active:number; flowing:number}>({total:0, active:0, flowing:0})
  useEffect(()=>{
    let stop=false
  const load = async()=>{ try{ const r=await fetch('/api/v1/sources'); if(!r.ok) return; const j=await r.json(); if(stop) return; const arr = Array.isArray(j)? j : (Array.isArray(j.items)? j.items : (j.sources||[])); const total=arr.length; const active=arr.filter((s:any)=>s.enabled).length; const flowing=arr.filter((s:any)=>s.flow).length; setSourcesSummary({total, active, flowing}) }catch{} }
    load(); const id=setInterval(load, 10000); return ()=>{ stop=true; clearInterval(id) }
  },[])
  // Allow deep-link style navigation from other components (e.g., Sources -> Filters)
  const [filtersInitialSelected, setFiltersInitialSelected] = useState<string | undefined>(undefined)
  useEffect(() => {
    const handler = (e: Event) => {
      // CustomEvent with detail { file: string }
      const ce = e as CustomEvent<{file?: string}>
      const file = ce.detail?.file
      if (file) setFiltersInitialSelected(file)
      setTab('transform')
    }
    window.addEventListener('open-filters', handler as EventListener)
    return () => window.removeEventListener('open-filters', handler as EventListener)
  }, [])

  const statusColor = health.data?.status === 'ok' ? '#22c55e' : '#ef4444'
  const startedOk = health.data?.status === 'ok'

  return (
    <div className="container home-layout">
    <header className="header">
        <div className="title">
          <img alt="logo" src="/logo.svg" className="logo" />
          <div>
            <h1>Bibbl Log Stream</h1>
            <p className="subtitle">Single-binary, cross-platform log pipeline</p>
          </div>
        </div>
        <div className="header-actions">
          <button className="btn secondary" onClick={() => { health.refresh(); version.refresh(); info.refresh(); }}>Refresh</button>
        </div>
      </header>

    <nav className="tabs">
        {[
          {k:'home', label:'Home'},
          {k:'sources', label:'Sources'},
      {k:'transform', label:'Transform'},
          {k:'destinations', label:'Destinations'},
      {k:'logevents', label:'Log Events'},
          {k:'azure', label:'Azure'},
        ].map(t => (
          <button key={t.k} className={`tab ${tab===t.k?'active':''}`} onClick={() => setTab(t.k as any)}>{t.label}</button>
        ))}
      </nav>

      {tab === 'home' && (
        <main className="home-grid">
          <section className="card span-4">
            <h2 style={{marginTop:0}}>System</h2>
            <div className="stats stats-tight">
              <Stat label="Health" value={<span style={{color: statusColor}}>{startedOk ? 'Healthy' : 'Down'}</span>} />
              <Stat label="Version" value={version.data?.version ?? '…'} />
              <Stat label="HTTP" value={info.data?.http_addr ?? '…'} />
              <Stat label="TLS" value={info.data ? (info.data.tls_enabled ? `Yes (>=${info.data.tls_min})` : 'No') : '…'} />
            </div>
            {(health.error || version.error || info.error) && (<div className="alert small">{(health.error || version.error || info.error)}</div>)}
          </section>
          <section className="card span-4">
            <h2 style={{marginTop:0}}>Sources</h2>
            <div className="mini-cards">
              <div className="mini"><div className="mini-label">Total</div><div className="mini-value">{sourcesSummary.total}</div></div>
              <div className="mini"><div className="mini-label">Enabled</div><div className="mini-value">{sourcesSummary.active}</div></div>
              <div className="mini"><div className="mini-label">Flowing</div><div className="mini-value">{sourcesSummary.flowing}</div></div>
            </div>
            <p className="muted" style={{marginTop:12, fontSize:12}}>Flowing = emitted a log in the last poll interval.</p>
            <button className="btn tiny secondary" onClick={()=>setTab('sources')}>Manage Sources →</button>
          </section>
          <section className="card span-4">
            <h2 style={{marginTop:0}}>Throughput</h2>
            <p className="muted" style={{marginTop:4}}>Event rate graph placeholder.</p>
            <div className="sparkline-placeholder">Coming soon</div>
            <ul className="links compact" style={{marginTop:12}}>
              <li><a href="/metrics" target="_blank" rel="noreferrer">Prometheus endpoint</a></li>
            </ul>
          </section>
          <section className="card span-8">
            <h2 style={{marginTop:0}}>Recent Activity</h2>
            <LiveTailPreview />
          </section>
            <section className="card span-4">
              <h2 style={{marginTop:0}}>Quick Links</h2>
              <ul className="links compact">
                <li><a href="/api/v1/health" target="_blank" rel="noreferrer">Health JSON</a></li>
                <li><a href="/api/v1/version" target="_blank" rel="noreferrer">Version JSON</a></li>
                <li><a href="/metrics" target="_blank" rel="noreferrer">Prometheus Metrics</a></li>
                <li><a href="https://github.com" target="_blank" rel="noreferrer">Docs (placeholder)</a></li>
              </ul>
              <h3 style={{margin:'18px 0 6px'}}>About</h3>
              <p style={{fontSize:13, lineHeight:1.4}}>Single binary includes inputs, processors, outputs & UI. Rebuild UI: <code>npm run build</code> then rebuild Go binary.</p>
            </section>
        </main>
      )}

  {tab === 'sources' && <SourcesConfig />}
  {tab === 'transform' && <TransformWorkbench filtersInitialSelected={filtersInitialSelected} />}
  {tab === 'destinations' && <DestinationsConfig />}
  {tab === 'logevents' && <MetricsPage />}
  {tab === 'azure' && <AzurePage />}

      <footer className="footer">© {new Date().getFullYear()} Bibbl</footer>
    </div>
  )
}

function CardTable<T extends object>({title, columns, rows}: {title: string; columns: {key: keyof T; label: string}[]; rows: T[]}){
  return (
    <main className="grid"><section className="card">
      <h2>{title}</h2>
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>{columns.map(c => <th key={String(c.key)}>{c.label}</th>)}</tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={i}>
                {columns.map(c => <td key={String(c.key)}>{String((r as any)[c.key])}</td>)}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section></main>
  )
}

// Lightweight tail preview polling the first enabled source stream endpoint.
function LiveTailPreview(){
  const [lines, setLines] = useState<string[]>([])
  const [sourceId, setSourceId] = useState<string>('')
  useEffect(()=>{
    let canceled=false
    const pick = async()=>{
  try{ const r=await fetch('/api/v1/sources'); if(!r.ok) return; const j=await r.json(); const arr = Array.isArray(j)? j : (Array.isArray(j.items)? j.items : (j.sources||[])); const first=arr.find((s:any)=>s.enabled); if(first) setSourceId(first.id) }catch{}
    }
    pick()
  },[])
  useEffect(()=>{
    if(!sourceId) return
    let stop=false
    const fetchLines = async()=>{
      try{ const r=await fetch(`/api/v1/sources/${sourceId}/stream?limit=10`); if(!r.ok) return; const text=await r.text(); const newLines=text.split(/\r?\n/).filter(Boolean).slice(-20); setLines(prev=>{ const merged=[...prev, ...newLines]; return merged.slice(-100) }) }catch{}
    }
    fetchLines(); const id=setInterval(fetchLines, 4000); return ()=>{ stop=true; clearInterval(id) }
  },[sourceId])
  if(!sourceId) return <div className="muted" style={{fontSize:12}}>No active source yet.</div>
  return <pre className="tail-box">{lines.slice(-15).join('\n')||'Waiting for logs...'}</pre>
}

function AzurePage(){
  const [dceResp, setDceResp] = useState<any>()
  const [dcrResp, setDcrResp] = useState<any>()
  const [err, setErr] = useState<string>()
  const [workspaceId, setWorkspaceId] = useState('')
  const [tableName, setTableName] = useState('Custom_BibblLogs_CL')
  const [dceName, setDceName] = useState('bibbl-dce')
  const [dcrName, setDcrName] = useState('bibbl-dcr')
  const doCreate = async (kind: 'dce'|'dcr') => {
    setErr(undefined)
    try{
      const body = kind==='dcr' ? {workspaceId, tableName, dcrName} : {dceName}
      const r = await fetch(`/api/v1/azure/sentinel/${kind}`, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)})
      const j = await r.json()
      kind==='dcr' ? setDcrResp(j) : setDceResp(j)
    }catch(e:any){ setErr(String(e?.message||e)) }
  }
  return (
    <main className="grid"><section className="card">
  <h2>Azure Automation</h2>
  <p className="muted" style={{marginTop:-6}}>Quick helpers to create a Data Collection Endpoint (DCE) and Data Collection Rule (DCR) for Microsoft Sentinel. Provide values, then create resources via the API.</p>
      <div className="form">
        <div className="row">
          <label>Workspace ID</label>
          <input value={workspaceId} onChange={e=>setWorkspaceId(e.target.value)} placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" />
        </div>
        <div className="row">
          <label>Table Name</label>
          <input value={tableName} onChange={e=>setTableName(e.target.value)} />
        </div>
        <div className="row">
          <label>DCE Name</label>
          <input value={dceName} onChange={e=>setDceName(e.target.value)} />
          <button className="btn" onClick={()=>doCreate('dce')}>Create DCE</button>
        </div>
        <div className="row">
          <label>DCR Name</label>
          <input value={dcrName} onChange={e=>setDcrName(e.target.value)} />
          <button className="btn" onClick={()=>doCreate('dcr')}>Create DCR</button>
        </div>
      </div>
      {err && <div className="alert">{err}</div>}
      <div className="grid">
        <section className="card"><h3>DCE Response</h3><pre className="pre">{JSON.stringify(dceResp||{}, null, 2)}</pre></section>
        <section className="card"><h3>DCR Response</h3><pre className="pre">{JSON.stringify(dcrResp||{}, null, 2)}</pre></section>
      </div>
    </section></main>
  )
}

// Removed legacy table-only pages in favor of full-featured MUI config components

function RegexPreview({initialSelected}: {initialSelected?: string}){
  const [pattern, setPattern] = useState('(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)\\s+(?P<method>\\w+)\\s+(?P<path>\\S+)')
  const [library, setLibrary] = useState<{name:string; size:number; modTime:string}[]>([])
  const [selected, setSelected] = useState<string>('')
  const [before, setBefore] = useState<string>('10.0.0.1 GET /index.html 200 123ms')
  const [after, setAfter] = useState<string>('')
  const [error, setError] = useState<string|undefined>()

  const loadLib = async () => {
    try{ const r = await fetch('/api/v1/library'); setLibrary(await r.json()) }catch{}
  }
  const loadFile = async (name: string) => {
    if(!name) return
    try{ const r = await fetch(`/api/v1/library/${encodeURIComponent(name)}`); setBefore(await r.text()) }catch(e:any){ setError(String(e?.message||e)) }
  }
  useEffect(()=>{ loadLib() }, [])
  // When initialSelected changes, preselect and load
  useEffect(()=>{
    if (initialSelected && initialSelected !== selected) {
      setSelected(initialSelected)
      loadFile(initialSelected)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialSelected])

  const run = async () => {
    setError(undefined)
    const lines = before.split(/\r?\n/)
    const outputs: string[] = []
    for (const line of lines) {
      if (!line) continue
      try {
        const r = await fetch('/api/v1/preview/regex', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({sample: line, pattern})})
        if (!r.ok) {
          // Try to parse JSON error, else plain text
          let msg = `HTTP ${r.status}`
          try { const j = await r.json(); msg = j.error||msg } catch {}
          throw new Error(msg)
        }
        const j = await r.json()
        outputs.push(JSON.stringify(j.captures||{}, null, 0))
      } catch(e:any) {
        outputs.push('{}')
        setError(String(e?.message||e))
      }
    }
    setAfter(outputs.join('\n'))
  }

  useEffect(()=>{ run() }, [])

  return (
    <main className="grid"><section className="card">
  <h2>Filters</h2>
      <p className="muted" style={{marginTop:-6}}>Tip: Use named capture groups like <code>(?P&lt;field&gt;...)</code>. Try pasting a captured syslog line from Sources ▶ eye and test your regex here.</p>
      <div className="row" style={{gap:12, alignItems:'center'}}>
        <label>Pattern</label>
        <input value={pattern} onChange={e=>setPattern(e.target.value)} style={{flex:1}} />
        <button className="btn" onClick={run}>Run</button>
        <button className="btn secondary" onClick={()=>setPattern('(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)\\s+(?P<app>\\w+)\\s+-\\s+-\\s+demo\\s+message\\s+(?P<seq>\\d+)')}>Sample regex</button>
        <div style={{width:16}} />
        <label>Sample Library</label>
        <select value={selected} onChange={e=>{ setSelected(e.target.value); loadFile(e.target.value) }}>
          <option value="">— choose —</option>
          {library.map(i => <option key={i.name} value={i.name}>{i.name}</option>)}
        </select>
        <button className="btn secondary" onClick={loadLib}>Refresh</button>
      </div>
      {error && <div className="alert">{error}</div>}
      <div className="grid" style={{gridTemplateColumns:'1fr 1fr', gap:12}}>
        <section className="card"><h3>Before</h3>
          <textarea value={before} onChange={e=>setBefore(e.target.value)} rows={16} style={{width:'100%'}} />
        </section>
        <section className="card"><h3>After</h3>
          <textarea value={after} readOnly rows={16} style={{width:'100%'}} />
        </section>
      </div>
    </section></main>
  )
}

function MetricsPage(){
  const [raw, setRaw] = useState<string>('')
  const [q, setQ] = useState<string>('')
  const [err, setErr] = useState<string>('')
  const load = async () => {
    setErr('')
    try{
      const r = await fetch('/metrics', {headers:{'Accept':'text/plain'}})
      const t = await r.text()
      setRaw(t)
    }catch(e:any){ setErr(String(e?.message||e)) }
  }
  useEffect(()=>{ load() }, [])
  const lines = raw.split(/\r?\n/)
  const filtered = q ? lines.filter(l => l.toLowerCase().includes(q.toLowerCase())) : lines
  return (
    <main className="grid"><section className="card">
  <h2>Log Events</h2>
      <div className="row" style={{display:'flex', gap:8, alignItems:'center'}}>
        <input placeholder="Filter (e.g., http_requests_total)" value={q} onChange={e=>setQ(e.target.value)} style={{flex:1}} />
        <button className="btn" onClick={load}>Refresh</button>
        <a className="btn secondary" href="/metrics" target="_blank" rel="noreferrer">Open raw</a>
      </div>
      {err && <div className="alert">{err}</div>}
      <pre className="pre" style={{whiteSpace:'pre-wrap'}}>{filtered.join('\n')}</pre>
    </section></main>
  )
}
