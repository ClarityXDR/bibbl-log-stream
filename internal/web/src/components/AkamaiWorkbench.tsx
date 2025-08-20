import React, { useEffect, useState } from 'react'

interface Stream { streamId: number; streamName: string; status: string; activated: boolean; datasetType: string }

// Simple badge helper
const Badge = ({text, tone}:{text:string; tone?:'ok'|'warn'|'err'|'info'}) => {
  const colors: Record<string,string> = { ok:'#065f46', warn:'#92400e', err:'#742a2a', info:'#1e3a8a' }
  return <span style={{background: colors[tone||'info'], color:'#fff', padding:'2px 8px', borderRadius:12, fontSize:12, fontWeight:500}}>{text}</span>
}

export function AkamaiWorkbench({sourceId, sourceName, onClose}:{sourceId:string; sourceName:string; onClose:()=>void}){
  const [streams, setStreams] = useState<Stream[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const [dataset, setDataset] = useState<string>('COMMON')
  const [fields, setFields] = useState<any>(null)
  const [rawPath, setRawPath] = useState<string>('/datastream-config/v2/log/streams')
  const [rawMethod, setRawMethod] = useState<string>('GET')
  const [rawBody, setRawBody] = useState<string>('')
  const [rawResult, setRawResult] = useState<string>('')
  const [rawLoading, setRawLoading] = useState(false)
  const [hasCreds, setHasCreds] = useState<boolean|undefined>(undefined)
  const [streamFilter, setStreamFilter] = useState('')
  const [lastRawStatus, setLastRawStatus] = useState<string>('')

  const loadStreams = async () => {
    if(!sourceId) return
    setLoading(true); setError('')
    try{
      const r = await fetch(`/api/v1/sources/${sourceId}/akamai/streams`)
      if(!r.ok){
        let body=''; try{ body = await r.text() }catch{}
        if(r.status===400 && body.includes('missing credentials')){ setHasCreds(false); setStreams([]); return }
        throw new Error(`${r.status} ${r.statusText}${body?` - ${body}`:''}`)
      }
      const j = await r.json(); setStreams(j.streams||[]); setHasCreds(true)
    }catch(e:any){ setError(String(e?.message||e)); setStreams([]) }
    finally { setLoading(false) }
  }
  const loadFields = async () => {
    if(!sourceId||!dataset) return
    setError(''); setFields(null)
    try{ const r = await fetch(`/api/v1/sources/${sourceId}/akamai/datasets/${encodeURIComponent(dataset)}/fields`);
  if(!r.ok){ let body=''; try{ body=await r.text() }catch{}; if(r.status===400 && body.includes('missing credentials')){ setHasCreds(false); setError('Credentials not configured. Add Akamai host + tokens in source config.'); return } throw new Error(`${r.status} ${r.statusText}${body?` - ${body}`:''}`) }
      const j = await r.json(); setFields(j.fields)
    }catch(e:any){ setError(String(e?.message||e)) }
  }
  const runRaw = async () => {
    if(!rawPath.trim()) return
    setRawLoading(true); setRawResult('')
    try {
      const qp = new URLSearchParams({path: rawPath, method: rawMethod})
      const r = await fetch(`/api/v1/sources/${sourceId}/akamai/raw?`+qp.toString(), {
        method: rawMethod === 'GET' ? 'GET':'POST',
        headers: rawBody? {'Content-Type':'application/json'}:undefined,
        body: rawMethod==='GET'||!rawBody?undefined:rawBody
      })
      let text = '';
      try { text = await r.text() } catch { text = '' }
  setLastRawStatus(`${r.status} ${r.statusText}`)
  if(r.status===400 && text.includes('missing credentials')){ setHasCreds(false); setRawResult('Credentials not configured. Add Akamai host + tokens in source config.'); return }
  if(!r.ok){ setRawResult(`${r.status} ${r.statusText}\n${text}`.trim()); return }
      try { const parsed = JSON.parse(text); setRawResult(JSON.stringify(parsed,null,2)) } catch { setRawResult(text) }
    } catch(e:any) { setRawResult(String(e?.message||e)) } finally { setRawLoading(false) }
  }
  useEffect(()=>{ loadStreams() }, [sourceId])
  const activate = async (sid:number, on:boolean) => {
    setError('')
    try{ const url = `/api/v1/sources/${sourceId}/akamai/streams/${sid}/${on?'deactivate':'activate'}`; const r = await fetch(url,{method:'POST'}); if(!r.ok){ let body=''; try{ body=await r.text() }catch{}; if(r.status===400 && body.includes('missing credentials')){ setHasCreds(false); return } throw new Error(`${r.status} ${r.statusText}${body?` - ${body}`:''}`) }; loadStreams() }catch(e:any){ setError(String(e?.message||e)) }
  }
  return (
    <div className="modal-backdrop">
      <div className="modal" style={{maxWidth:1200}}>
        <header className="modal-header">
          <div style={{display:'flex', flexDirection:'column'}}>
            <h2 style={{margin:0}}>Akamai DataStream 2 Workbench</h2>
            <div style={{display:'flex', gap:8, flexWrap:'wrap', alignItems:'center'}}>
              <Badge text={sourceName} tone='info' />
              {hasCreds===false && <Badge text='Credentials Missing' tone='err' />}
              {hasCreds && <Badge text='Credentials OK' tone='ok' />}
              <button className="btn tiny" onClick={loadStreams} disabled={loading}>{loading?'Refreshing…':'Refresh Streams'}</button>
            </div>
          </div>
          <div style={{display:'flex', gap:8}}>
            <button className="btn secondary small" onClick={()=>{ setDataset('COMMON'); setFields(null); }}>Reset</button>
            <button className="btn small" onClick={onClose}>Close</button>
          </div>
        </header>
        <div className="modal-body" style={{display:'flex', flexDirection:'column', gap:12}}>
          <p className="muted" style={{marginTop:-4}}>Manage DataStream 2 resources, fetch dataset field definitions, and run signed exploratory requests. Docs: <a href="https://techdocs.akamai.com/datastream2/reference/api-summary" target="_blank" rel="noreferrer">API Summary</a></p>
          {hasCreds===false && (
            <div className="alert" style={{background:'#4b1d1d'}}>Akamai credentials not configured. Edit the source and add Host, Client Token, Client Secret, Access Token. Then toggle the source off/on and refresh.</div>
          )}
          {error && <div className="alert">{error}</div>}
          <div style={{display:'grid', gridTemplateColumns:'1fr 1fr', gap:16, alignItems:'start'}}>
            <section className="card" style={{display:'flex', flexDirection:'column'}}>
              <div style={{display:'flex', justifyContent:'space-between', alignItems:'center'}}>
                <h3 style={{margin:0}}>Streams</h3>
                <input placeholder='Filter' value={streamFilter} onChange={e=>setStreamFilter(e.target.value)} style={{maxWidth:160}} />
              </div>
              {!loading && streams.length>0 && (
                <div className="table-wrap" style={{marginTop:12, flex:1, overflow:'auto'}}>
                  <table className="table small">
                    <thead><tr><th>ID</th><th>Name</th><th>Status</th><th>A</th><th>Dataset</th><th></th></tr></thead>
                    <tbody>
                      {streams.filter(s=>!streamFilter||s.streamName.toLowerCase().includes(streamFilter.toLowerCase())||String(s.streamId).includes(streamFilter)).map(st => (
                        <tr key={st.streamId}>
                          <td>{st.streamId}</td>
                          <td style={{maxWidth:180, overflow:'hidden', textOverflow:'ellipsis'}} title={st.streamName}>{st.streamName}</td>
                          <td>{st.status}</td>
                          <td>{st.activated? '✔':'—'}</td>
                          <td>{st.datasetType}</td>
                          <td><button className="btn tiny" onClick={()=>activate(st.streamId, st.activated)}>{st.activated? 'Deactivate':'Activate'}</button></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              {loading && <div className='muted' style={{marginTop:8}}>Loading…</div>}
              {!loading && streams.length===0 && hasCreds && <div className="muted" style={{marginTop:8}}>No streams returned.</div>}
            </section>
            <div style={{display:'flex', flexDirection:'column', gap:16}}>
              <section className="card">
                <h3 style={{marginTop:0}}>Dataset Fields</h3>
                <div className="row" style={{gap:8, flexWrap:'wrap'}}>
                  <input value={dataset} onChange={e=>setDataset(e.target.value)} placeholder="Dataset (e.g. COMMON)" />
                  <button className="btn" onClick={loadFields} disabled={hasCreds===false}>Get Fields</button>
                  {fields && <button className="btn secondary" onClick={()=>{ const blob=new Blob([JSON.stringify(fields,null,2)],{type:'application/json'}); const a=document.createElement('a'); a.href=URL.createObjectURL(blob); a.download=`akamai_fields_${dataset}.json`; a.click(); }}>Download JSON</button>}
                </div>
                {fields && <pre style={{maxHeight:160, overflow:'auto', background:'#111827', padding:8, borderRadius:4, marginTop:8}}>{JSON.stringify(fields,null,2)}</pre>}
              </section>
              <section className="card">
                <h3 style={{marginTop:0}}>API Explorer</h3>
                <div className="row" style={{gap:8, flexWrap:'wrap'}}>
                  <select value={rawMethod} onChange={e=>setRawMethod(e.target.value)}>
                    {['GET','POST','PUT','DELETE'].map(m=> <option key={m}>{m}</option>)}
                  </select>
                  <input style={{flex:1,minWidth:220}} value={rawPath} onChange={e=>setRawPath(e.target.value)} placeholder="/datastream-config/v2/log/streams" />
                  <button className="btn" disabled={rawLoading||hasCreds===false} onClick={runRaw}>{rawLoading? 'Running…':'Send'}</button>
                </div>
                {rawMethod !== 'GET' && (
                  <textarea style={{width:'100%', minHeight:80, marginTop:8, fontFamily:'monospace'}} placeholder='Optional JSON body' value={rawBody} onChange={e=>setRawBody(e.target.value)} />
                )}
                {lastRawStatus && <div style={{marginTop:8, fontSize:12}} className='muted'>Last status: {lastRawStatus}</div>}
                {rawResult && <pre style={{maxHeight:180, overflow:'auto', background:'#111827', padding:8, borderRadius:4, marginTop:8}}>{rawResult}</pre>}
                <p className="muted" style={{marginTop:8}}>Path must begin with <code>/datastream-</code>. Requests are signed with the configured credentials.</p>
              </section>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default AkamaiWorkbench
