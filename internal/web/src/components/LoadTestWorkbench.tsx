import React, { useEffect, useState } from 'react'

type Status = { sourceId: string; produced: number; eps: number; config: any }

interface Props {
  sourceId?: string
  sourceName?: string
  sourceConfig?: Record<string, any>
  onClose?: () => void
  inline?: boolean // when true render as full page section (legacy tab)
}

export default function LoadTestWorkbench({sourceId, sourceName, sourceConfig, onClose, inline}: Props){
  const [rate, setRate] = useState(20000)
  const [size, setSize] = useState(300)
  const [workers, setWorkers] = useState(4)
  const [template, setTemplate] = useState(sourceName?`synthetic from ${sourceName} ${'${seq}'}`:'zeek conn log synthetic ${seq}')
  const [compress, setCompress] = useState(false)
  const [status, setStatus] = useState<Status|null>(null)
  const [err, setErr] = useState<string>('')
  const [running, setRunning] = useState(false)

  const loadStatus = async () => {
    try {
      const r = await fetch('/api/v1/loadtest/status')
      if(!r.ok) throw new Error(`${r.status}`)
      const j = await r.json(); setStatus(j); setRunning(true)
    } catch { setRunning(false) }
  }
  useEffect(()=>{ const id = setInterval(loadStatus, 2000); loadStatus(); return ()=>clearInterval(id) }, [])

  const start = async () => {
    setErr('')
    try{
  const body = { rate, size, workers, template, compress }
      const r = await fetch('/api/v1/loadtest/start', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)})
      if(!r.ok) throw new Error(`${r.status}`)
      await loadStatus()
    }catch(e:any){ setErr(String(e?.message||e)) }
  }
  const stop = async () => { try{ await fetch('/api/v1/loadtest/stop', {method:'POST'}); setRunning(false) }catch{} }

  const content = (
    <section className="card" style={inline?undefined:{minWidth:600}}>
      <h2>Load Test {sourceName?`(from ${sourceName})`: 'Workbench'}</h2>
      <p className="muted" style={{marginTop:-6}}>Generate synthetic events to evaluate throughput and queue behavior. A temporary synthetic source is created; original sources are untouched.</p>
      <div className="form">
        <div className="row"><label>Rate (EPS)</label><input type="number" value={rate} onChange={e=>setRate(parseInt(e.target.value)||0)} /></div>
        <div className="row"><label>Size (bytes)</label><input type="number" value={size} onChange={e=>setSize(parseInt(e.target.value)||0)} /></div>
        <div className="row"><label>Workers</label><input type="number" value={workers} onChange={e=>setWorkers(parseInt(e.target.value)||0)} /></div>
        <div className="row"><label>Template</label><input value={template} onChange={e=>setTemplate(e.target.value)} style={{flex:1}} /></div>
  <div className="row"><label>Compress (gzip+base64)</label><input type="checkbox" checked={compress} onChange={e=>setCompress(e.target.checked)} /></div>
        <div className="row" style={{gap:8}}>
          <button className="btn" onClick={start} disabled={running}>Start</button>
          <button className="btn secondary" onClick={stop} disabled={!running}>Stop</button>
          <button className="btn secondary" onClick={loadStatus}>Refresh</button>
          {onClose && <button className="btn secondary" onClick={onClose}>Close</button>}
        </div>
      </div>
      {err && <div className="alert">{err}</div>}
      <div className="grid" style={{gridTemplateColumns:'1fr 1fr 1fr', gap:12}}>
        <section className="card"><h3>Current EPS</h3><div style={{fontSize:28}}>{status ? status.eps.toFixed(0) : '—'}</div></section>
        <section className="card"><h3>Total Produced</h3><div style={{fontSize:28}}>{status ? status.produced : '—'}</div></section>
        <section className="card"><h3>Source ID</h3><div style={{fontSize:16}}>{status?.sourceId||'—'}</div></section>
      </div>
      <section className="card"><h3>Config</h3><pre className="pre">{status? JSON.stringify(status.config, null, 2): JSON.stringify({rate,size,workers,template,compress},null,2)}</pre></section>
      {sourceConfig && <section className="card"><h3>Source Settings Snapshot</h3><pre className="pre">{JSON.stringify(sourceConfig,null,2)}</pre></section>}
    </section>
  )

  if (inline) return <main className="grid">{content}</main>
  return (
    <div className="modal-backdrop">
      <div className="modal" style={{maxWidth:900}}>
        {content}
      </div>
    </div>
  )
}
