import React, { useEffect, useState } from 'react'
import './styles.css'

type Status = { sourceId: string; produced: number; eps: number; config: any; error?: string }

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
      const j = await r.json(); 
      setStatus(j); 
      if (j.error) {
        setRunning(false)
      } else {
        setRunning(j.sourceId !== "")
      }
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
    <section className={`card ${inline ? 'load-test-card inline' : 'load-test-card'}`}>
      <h2>Load Test {sourceName?`(from ${sourceName})`: 'Workbench'}</h2>
      <p className="muted load-test-muted">Generate synthetic events to evaluate throughput and queue behavior. A temporary synthetic source is created; original sources are untouched.</p>
      <div className="form">
        <div className="row"><label htmlFor="rate-input">Rate (EPS)</label><input id="rate-input" type="number" value={rate} onChange={e=>setRate(parseInt(e.target.value)||0)} /></div>
        <div className="row"><label htmlFor="size-input">Size (bytes)</label><input id="size-input" type="number" value={size} onChange={e=>setSize(parseInt(e.target.value)||0)} /></div>
        <div className="row"><label htmlFor="workers-input">Workers</label><input id="workers-input" type="number" value={workers} onChange={e=>setWorkers(parseInt(e.target.value)||0)} /></div>
        <div className="row"><label htmlFor="template-input">Template</label><input id="template-input" className="load-test-template-input" value={template} onChange={e=>setTemplate(e.target.value)} /></div>
  <div className="row"><label htmlFor="compress-input">Compress (gzip+base64)</label><input id="compress-input" type="checkbox" checked={compress} onChange={e=>setCompress(e.target.checked)} /></div>
        <div className="row load-test-row-gap">
          <button className="btn" onClick={start} disabled={running}>Start</button>
          <button className="btn secondary" onClick={stop} disabled={!running}>Stop</button>
          <button className="btn secondary" onClick={loadStatus}>Refresh</button>
          {onClose && <button className="btn secondary" onClick={onClose}>Close</button>}
        </div>
      </div>
      {err && <div className="alert">{err}</div>}
      {status?.error && <div className="alert load-test-alert-warning">
        ℹ️ {status.error}. You can still create a temporary synthetic source using the controls above.
      </div>}
      <div className="grid load-test-grid">
        <section className="card"><h3>Current EPS</h3><div className="load-test-stat-large">{status ? status.eps.toFixed(0) : '—'}</div></section>
        <section className="card"><h3>Total Produced</h3><div className="load-test-stat-large">{status ? status.produced : '—'}</div></section>
        <section className="card"><h3>Source ID</h3><div className="load-test-stat-small">{status?.sourceId||'—'}</div></section>
      </div>
      <section className="card"><h3>Config</h3><pre className="pre">{status? JSON.stringify(status.config, null, 2): JSON.stringify({rate,size,workers,template,compress},null,2)}</pre></section>
      {sourceConfig && <section className="card"><h3>Source Settings Snapshot</h3><pre className="pre">{JSON.stringify(sourceConfig,null,2)}</pre></section>}
    </section>
  )

  if (inline) return <main className="grid">{content}</main>
  return (
    <div className="modal-backdrop">
      <div className="modal load-test-modal">
        {content}
      </div>
    </div>
  )
}
