import { useEffect, useMemo, useState } from 'react'
import type { ChangeReview, ReviewSnapshot } from './App'
import { affectedDiagrams, ReportBody } from './PrintableProposal'
import { MarkdownBody } from './MarkdownBody'
import './comparison.css'

type Selector = {kind:string;revision?:string;change_set_id?:string;state?:string;review_id?:string;side?:string}
type Version = {selector:Selector;label:string;context?:string;revision:string;generation?:number;document?:string;document_source?:string;unavailable?:boolean}
type Project = {store_id:string;slug:string;name:string;unavailable?:boolean;conflict?:boolean}
type Page = {versions:Version[];next_cursor?:string}
type ReviewProposal = {change_set_id:string;label:string}
type Comparison = {project_name:string;project_slug:string;store_id:string;before_version:Version;after_version:Version;before:ReviewSnapshot;after:ReviewSnapshot;changes:ChangeReview['comparison'];diff:string;report_url:string}
const sources=[['accepted','Accepted history'],['proposal','Open proposals'],['applied','Accepted proposals'],['submitted_review','Submitted reviews']] as const
const key=(s:Selector)=>JSON.stringify(s)

async function read<T>(operation:string,input:unknown,signal?:AbortSignal):Promise<T> {
 const response=await fetch(`/api/agent/v2/architecture/${operation}`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(input),signal})
 const body=await response.json()
 if(!response.ok||!body.ok)throw new Error(body.error?.message??'These versions could not be loaded. Try again.')
 return body.result
}
function exactPair():{before?:Selector;after?:Selector;store?:string} {
 const q=new URLSearchParams(window.location.search)
 for(const [name] of q)if(!['store_id','before','after'].includes(name)||q.getAll(name).length!==1)throw new Error('This comparison link is invalid. Choose the versions again.')
 const parse=(name:string)=>q.has(name)?JSON.parse(q.get(name)!):undefined
 return {before:parse('before'),after:parse('after'),store:q.get('store_id')??undefined}
}
function pairQuery(store:string,before?:Selector,after?:Selector) {
 const q=new URLSearchParams({store_id:store})
 if(before)q.set('before',key(before));if(after)q.set('after',key(after))
 return q.toString()
}

export function VersionComparison() {
 const [project,setProject]=useState<Project>(),[error,setError]=useState('')
 const path=window.location.pathname.match(/^\/projects\/([^/]+)\/compare(\/report)?\/?$/)
 useEffect(()=>{
  const controller=new AbortController()
  async function load(){
   if(!path)throw new Error('This comparison link is invalid. Return to projects.')
   const response=await fetch('/api/projects',{signal:controller.signal}),catalog=await response.json()
   const matches=(catalog.projects??[]).filter((p:Project)=>p.slug===decodeURIComponent(path![1]))
   if(!response.ok||matches.length!==1||matches[0].unavailable||matches[0].conflict)throw new Error('This project is unavailable. Return to projects and inspect its status.')
   const pair=exactPair();if(pair.store&&pair.store!==matches[0].store_id)throw new Error('This link belongs to a different project. Choose the versions again.')
   setProject(matches[0])
  }
  load().catch(e=>{if(!controller.signal.aborted)setError(e.message)})
  return()=>controller.abort()
 },[])
 if(error)return <main className="compare-page"><a className="comparison-button secondary" href={path?`/projects/${path[1]}/compare`:'/'}>Return to {path?'Compare versions':'projects'}</a><h1>Comparison unavailable</h1><p role="alert">{error}</p></main>
 if(!project)return <main className="compare-page"><p role="status">Loading project…</p></main>
 return path?.[2]?<ComparisonReport project={project}/>:<VersionPicker project={project}/>
}

function VersionPicker({project}:{project:Project}) {
 const initial=useMemo(()=>exactPair(),[])
 const [before,setBefore]=useState<Selector|undefined>(initial.before),[after,setAfter]=useState<Selector|undefined>(initial.after)
 const [busy,setBusy]=useState(false),[error,setError]=useState('')
 function select(side:'before'|'after',value?:Selector) {
  if(side==='before')setBefore(value);else setAfter(value)
  setError('')
  window.history.replaceState({},'',`${window.location.pathname}?${pairQuery(project.store_id,side==='before'?value:before,side==='after'?value:after)}`)
 }
 async function report(){
  if(!before||!after)return
  setBusy(true);setError('')
  try {const result=await read<Comparison>('compare',{store_id:project.store_id,before,after});window.location.assign(result.report_url)}
  catch(e){setError(e instanceof Error?e.message:'The comparison could not be loaded.');setBusy(false)}
 }
 return <main className="compare-page">
  <nav className="compare-nav"><a className="comparison-button secondary" href={`/projects/${encodeURIComponent(project.slug)}`}>Return to architecture</a><span>WorkBraid</span></nav>
  <header className="compare-heading"><p className="eyebrow">{project.name}</p><h1>Compare versions</h1><p>Choose two versions to see what changed.</p></header>
  <div className="compare-pair">
   <VersionSelect side="Before" project={project} selected={before} onSelect={v=>select('before',v)}/>
   <VersionSelect side="After" project={project} selected={after} onSelect={v=>select('after',v)}/>
  </div>
  <div className="compare-submit"><button className="comparison-button secondary" disabled={!before||!after||busy} onClick={()=>{setBefore(after);setAfter(before);setError('');window.history.replaceState({},'',`${window.location.pathname}?${pairQuery(project.store_id,after,before)}`)}}>Swap versions</button><button className="comparison-button" disabled={!before||!after||busy} onClick={report}>{busy?'Loading comparison…':'Report'}</button></div>
  {error&&<p className="compare-error" role="alert">{error}</p>}
  <p className="compare-footnote">Reports read the versions you choose. They do not prepare Review or update Architecture.</p>
 </main>
}

function VersionSelect({side,project,selected,onSelect}:{side:string;project:Project;selected?:Selector;onSelect:(s:Selector)=>void}) {
 const [source,setSource]=useState(selected?.kind??'accepted'),[versions,setVersions]=useState<Version[]>([]),[cursor,setCursor]=useState<string>(),[busy,setBusy]=useState(false),[error,setError]=useState(''),[retry,setRetry]=useState(0)
 const [proposal,setProposal]=useState(selected?.kind==='submitted_review'?selected.change_set_id:''),[proposals,setProposals]=useState<ReviewProposal[]>([]),[proposalCursor,setProposalCursor]=useState<string>()
 useEffect(()=>{if(selected){setSource(selected.kind);if(selected.kind==='submitted_review')setProposal(selected.change_set_id)}},[selected?.kind,selected?.change_set_id])
 useEffect(()=>{
  if(source!=='submitted_review')return
  const controller=new AbortController()
  read<{review_proposals?:ReviewProposal[];next_cursor?:string}>('versions',{store_id:project.store_id,source:'review_proposals'},controller.signal).then(p=>{setProposals(p.review_proposals??[]);setProposalCursor(p.next_cursor)}).catch(e=>{if(!controller.signal.aborted)setError(e.message)})
  return()=>controller.abort()
 },[project.store_id,source,retry])
 useEffect(()=>{
  const controller=new AbortController();setBusy(true);setError('');setVersions([]);setCursor(undefined)
  if(source==='submitted_review'&&!proposal){setBusy(false);return()=>controller.abort()}
  read<Page>('versions',{store_id:project.store_id,source,...(source==='submitted_review'?{change_set_id:proposal}:{})},controller.signal).then(p=>{setVersions(p.versions);setCursor(p.next_cursor)}).catch(e=>{if(!controller.signal.aborted)setError(e.message)}).finally(()=>{if(!controller.signal.aborted)setBusy(false)})
  return()=>controller.abort()
 },[project.store_id,source,proposal,retry])
 async function more(){setBusy(true);setError('');try{const p=await read<Page>('versions',{store_id:project.store_id,source,cursor,...(source==='submitted_review'?{change_set_id:proposal}:{})});setVersions(v=>[...v,...p.versions]);setCursor(p.next_cursor)}catch(e){setError(e instanceof Error?e.message:'More versions could not be loaded.')}finally{setBusy(false)}}
 async function moreProposals(){setBusy(true);try{const p=await read<{review_proposals?:ReviewProposal[];next_cursor?:string}>('versions',{store_id:project.store_id,source:'review_proposals',cursor:proposalCursor});setProposals(old=>[...new Map([...old,...p.review_proposals??[]].map(p=>[p.change_set_id,p])).values()]);setProposalCursor(p.next_cursor)}catch(e){setError(e instanceof Error?e.message:'Proposals could not be loaded.')}finally{setBusy(false)}}
 const selectedVersion=versions.find(v=>key(v.selector)===key(selected??{kind:''}))
 return <section className="version-side" aria-label={`${side} version`}>
  <h2>{side}</h2>
  <label htmlFor={`${side}-source`}>From</label><select id={`${side}-source`} value={source} onChange={e=>setSource(e.target.value)}>{sources.map(([id,label])=><option key={id} value={id}>{label}</option>)}</select>
  {source==='submitted_review'&&<><label htmlFor={`${side}-proposal`}>Proposal with submitted reviews</label><select id={`${side}-proposal`} value={proposal} onChange={e=>setProposal(e.target.value)}><option value="">Choose a proposal</option>{proposals.map(p=><option key={p.change_set_id} value={p.change_set_id}>{p.label}{proposals.filter(other=>other.label===p.label).length>1?` · ${p.change_set_id.slice(0,8)}`:''}</option>)}</select>{proposalCursor&&<button className="comparison-text-button" disabled={busy} onClick={moreProposals}>More proposals</button>}</>}
  <label htmlFor={`${side}-version`}>Version</label><select id={`${side}-version`} value={selectedVersion?key(selectedVersion.selector):''} onChange={e=>{if(e.target.value)onSelect(JSON.parse(e.target.value))}} disabled={busy&&versions.length===0}>
   <option value="">{busy?'Loading versions…':'Choose a version'}</option>
   {versions.map((v,i)=><option key={`${key(v.selector)}:${i}`} value={key(v.selector)} disabled={v.unavailable}>{v.label}{versions.filter(other=>other.label===v.label).length>1?` · ${(v.selector.review_id??v.selector.change_set_id??v.selector.revision??'').slice(0,8)}`:''}</option>)}
  </select>
  <div className="version-page-actions">{cursor&&<button className="comparison-text-button" disabled={busy} onClick={more}>{busy?'Loading…':'Load more'}</button>}{error&&<button className="comparison-text-button" onClick={()=>setRetry(n=>n+1)}>Reload choices</button>}</div>
  {error?<p className="compare-error" role="alert">{error}</p>:!busy&&versions.length===0?<p>No versions in this group.</p>:null}
  <div className="version-selection" aria-live="polite">{selected?<><span className="eyebrow">Selected</span><p>{selectedVersion?.label??`${sources.find(s=>s[0]===selected.kind)?.[1]??'Retained version'} · exact selection retained`}</p><details><summary>Version details</summary><p>{selectedVersion?.context}</p><code>{JSON.stringify(selected,null,2)}</code></details></>:<p>Select a version above.</p>}</div>
  {source==='accepted'&&<p className="version-explanation">Earlier versions reachable from current Accepted. External merge ancestors may not have been individually accepted in WorkBraid.</p>}
 </section>
}

function ComparisonReport({project}:{project:Project}) {
 const [value,setValue]=useState<Comparison>(),[error,setError]=useState(''),[printError,setPrintError]=useState(''),[ready,setReady]=useState<Record<string,boolean>>({})
 const [includePresentation,setIncludePresentation]=useState(false),[includeDiff,setIncludeDiff]=useState(false)
 const pair=useMemo(()=>exactPair(),[]),returnURL=`/projects/${encodeURIComponent(project.slug)}/compare?${pairQuery(project.store_id,pair.before,pair.after)}`
 useEffect(()=>{const controller=new AbortController();if(!pair.before||!pair.after||!pair.store){setError('This report needs two exact versions. Return to Compare versions.');return}
  read<Comparison>('compare',{store_id:project.store_id,before:pair.before,after:pair.after},controller.signal).then(setValue).catch(e=>{if(!controller.signal.aborted)setError(e.message)});return()=>controller.abort()
 },[project.store_id,pair])
 // Shared presentation has no authority fields. The adapter only supplies
 // existing renderer vocabulary; the server's payload remains a comparison.
 const review=useMemo(()=>value?{diff:value.diff,before:value.before,with_changes:value.after,comparison:value.changes} as ChangeReview:undefined,[value])
 const allReady=!!review&&affectedDiagrams(review,includePresentation).every(d=>ready[d.id])
 async function print(){try{await document.fonts.ready;await Promise.all([...document.querySelectorAll<HTMLImageElement>('.print-drawing img')].map(i=>i.decode()));window.print()}catch{setPrintError('A drawing could not finish loading. Reload this report before printing.')}}
 const shared=value&&value.before_version.document_source&&value.before_version.document_source===value.after_version.document_source
 const identities=value?JSON.stringify({store_id:value.store_id,before:{...pair.before,tree_or_revision:value.before_version.revision,context:value.before_version.context,generation:value.before_version.generation},after:{...pair.after,tree_or_revision:value.after_version.revision,context:value.after_version.context,generation:value.after_version.generation}},null,2):''
 return <main className="print-document comparison-document"><nav className="print-actions"><a className="comparison-button secondary" href={returnURL}>Compare versions</a>{allReady&&<button className="comparison-button" onClick={print}>Print / Save as PDF</button>}</nav>
 {error?<><h1>Comparison unavailable</h1><p role="alert">{error}</p></>:!value||!review?<p role="status">Loading exact comparison…</p>:<>
 <header className="compare-heading"><p className="eyebrow">{value.project_name}</p><h1>Architecture comparison</h1><div className="report-provenance"><div><h2>Before</h2><p>{value.before_version.label}</p></div><div><h2>After</h2><p>{value.after_version.label}</p></div></div></header>
 <div className="print-options"><label><input type="checkbox" checked={includePresentation} onChange={e=>{setReady({});setIncludePresentation(e.target.checked)}}/> Include presentation changes</label><label><input type="checkbox" checked={includeDiff} onChange={e=>setIncludeDiff(e.target.checked)}/> Include complete raw diff</label></div>
 {printError&&<p role="alert">{printError}</p>}
 {([['Before',value.before_version],['After',value.after_version]] as const).filter(([side,v])=>v.document!==undefined&&!(shared&&side==='After')).map(([side,v])=><section className="print-design" key={side}><h2>{shared?'Before and After':side} proposal document</h2><p className="print-key">Context retained with this version.</p>{v.document?<MarkdownBody source={v.document}/>:<p>No proposal text was recorded.</p>}</section>)}
 <ReportBody review={review} includePresentation={includePresentation} includeDiff={includeDiff} comparison onReady={(id,ok)=>setReady(old=>({...old,[id]:ok}))}/>
 <footer className="print-binding"><p>This report compares the selected versions. It does not prepare Review or approve an Architecture update.</p><details className="screen-identities"><summary>Exact versions</summary><code>{identities}</code></details><section className="printed-identities"><h2>Exact versions</h2><p>{value.project_name} · Store <span className="identity-value">{value.store_id}</span></p><div className="printed-pair"><PrintedVersion side="Before" version={value.before_version}/><PrintedVersion side="After" version={value.after_version}/></div></section></footer>
 {!allReady&&<p role="status">Printing is available when every drawing is ready.</p>}
 </>}
 </main>
}

function PrintedVersion({side,version:v}:{side:string;version:Version}) {
 const identities=[['Architecture '+(v.selector.side==='candidate'?'tree':'revision'),v.revision],['Proposal',v.selector.change_set_id],['Proposal state',v.selector.state],['Submitted review',v.selector.review_id]].filter(([,value])=>value)
 return <div><h3>{side} · {v.label}</h3>{v.context&&<p>{v.context}</p>}<dl>{identities.map(([label,value])=><div key={label}><dt>{label}</dt><dd className="identity-value">{value}</dd></div>)}</dl></div>
}
