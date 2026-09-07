import { useEffect, useState } from 'react'
import { type ChangeReview, type DiagramProjection, type ReviewSnapshot, mapComponentsForDiagram } from './App'
import { printDiagramImages } from './ArchitectureMap'
import { MarkdownBody } from './MarkdownBody'
import { RawDiff } from './RawDiff'
import './print.css'

type PrintDocument = { project_name:string;store_id:string;name:string;proposal_markdown:string;lifecycle:string;state:string;review_id:string;applied_revision:string;accepted_revision:string;review:ChangeReview }
type AffectedDiagram = {id:string;before?:DiagramProjection;with?:DiagramProjection;textIDs:string[]}
const visible = (d?:DiagramProjection) => new Set([...(d?.appearances??[]),...(d?.boundaries??[])].map(n=>n.component_id))
const source = (s:ReviewSnapshot,id:string) => s.components.find(c=>c.id===id)?.markdown_source??''
const title = (r:ChangeReview,id:string) => r.with_changes.components.find(c=>c.id===id)?.title??r.before.components.find(c=>c.id===id)?.title??id
const diagramTitle = (r:ChangeReview,id?:string) => [...(r.with_changes.diagrams??[]),...(r.before.diagrams??[])].find(d=>d.id===id)?.title??'None'

export function affectedDiagrams(r:ChangeReview):AffectedDiagram[] {
 if(r.diff==='')return []
 const before=r.before.diagrams??[],after=r.with_changes.diagrams??[],c=r.comparison
 const textIDs=[...new Set([...r.before.components,...r.with_changes.components].map(c=>c.id))].filter(id=>source(r.before,id)!==source(r.with_changes,id))
 const structural=new Set([...(c.diagrams??[]),...(c.appearances??[]),...(c.node_positions??[]),...(c.node_sizes??[]),...(c.edge_routes??[]),...(c.node_shapes??[]),...(c.diagram_notes??[])].map(c=>c.diagram_id))
 const ordered=[...after,...before.filter(d=>!after.some(a=>a.id===d.id))]
 return ordered.map(d=>{
  const b=before.find(x=>x.id===d.id),w=after.find(x=>x.id===d.id),v=new Set([...visible(b),...visible(w)])
  return {id:d.id,before:b,with:w,textIDs:textIDs.filter(id=>v.has(id))}
 }).filter(d=>!d.before||!d.with||structural.has(d.id)||d.textIDs.length||d.before.parent_anchor_component_id!==d.with.parent_anchor_component_id||d.before.parent_diagram_id!==d.with.parent_diagram_id||JSON.stringify(d.before.boundaries.map(b=>[b.component_id,b.home_diagram_id,b.home_diagram_title]).sort())!==JSON.stringify(d.with.boundaries.map(b=>[b.component_id,b.home_diagram_id,b.home_diagram_title]).sort())||c.relationships.some(e=>e.diagram_projections?.some(p=>p.diagram_id===d.id)))
}

export function sourceDiff(before:string,after:string) {
 const a=before.match(/[^\n]*\n|[^\n]+$/g)??[],b=after.match(/[^\n]*\n|[^\n]+$/g)??[]
 let start=0,end=0
 while(start<a.length&&start<b.length&&a[start]===b[start])start++
 while(end<a.length-start&&end<b.length-start&&a[a.length-end-1]===b[b.length-end-1])end++
 return [...a.slice(0,start).map(text=>({kind:'context',text})),...a.slice(start,a.length-end).map(text=>({kind:'removed',text})),...b.slice(start,b.length-end).map(text=>({kind:'added',text})),...a.slice(a.length-end).map(text=>({kind:'context',text}))]
}

function SourceDiff({before,after}:{before:string;after:string}) {
 return <div className="print-source-diff"><p className="print-key">Source text · − deleted / + added · spaces ·, tabs →, CR ␍, LF ↵; ∎ means no final newline.</p>
 <pre>{sourceDiff(before,after).map((line,i)=><span key={i} className={`diff-line diff-${line.kind}`}><span aria-hidden="true">{line.kind==='added'?'+ ':line.kind==='removed'?'− ':'  '}</span>{line.text.replace(/ /g,'·').replace(/\t/g,'→').replace(/\r/g,'␍').replace(/\n$/,'↵')}{line.text.endsWith('\n')?'':'∎'}{'\n'}</span>)}</pre></div>
}

function textHome(r:ChangeReview,id:string) {
 return (r.with_changes.diagrams??[]).find(d=>d.appearances.some(a=>a.component_id===id&&a.role==='home'))?.id??(r.before.diagrams??[]).find(d=>d.appearances.some(a=>a.component_id===id&&a.role==='home'))?.id
}
function DiagramFacts({diagram:d,review:r}:{diagram:AffectedDiagram;review:ChangeReview}) {
 const c=r.comparison, facts:string[]=[]
 if(!d.before)facts.push('Diagram added.')
 if(!d.with)facts.push('Diagram removed.')
 if(d.before&&d.with){
  if(d.before.title!==d.with.title)facts.push(`Title: “${d.before.title}” → “${d.with.title}”.`)
  if(d.before.parent_anchor_component_id!==d.with.parent_anchor_component_id||d.before.parent_diagram_id!==d.with.parent_diagram_id)facts.push(`Parent: ${diagramTitle(r,d.before.parent_diagram_id)} / ${d.before.parent_anchor_component_id?title(r,d.before.parent_anchor_component_id):'Root'} → ${diagramTitle(r,d.with.parent_diagram_id)} / ${d.with.parent_anchor_component_id?title(r,d.with.parent_anchor_component_id):'Root'}.`)
 }
 for(const a of c.appearances??[])if(a.diagram_id===d.id)facts.push(`${title(r,a.component_id)}: ${a.role==='home'?'lives here':'included here'} ${a.status==='added'?'added':a.status==='removed'?'removed':'detail link changed'}${a.detail_diagram_id?` · opens ${diagramTitle(r,a.detail_diagram_id)}`:''}.`)
 for(const e of c.relationships)if(e.diagram_projections?.some(p=>p.diagram_id===d.id)){
  const count=(s:ReviewSnapshot)=>s.components.find(n=>n.id===e.source_id)?.relationships.filter(n=>n.target_id===e.target_id&&n.label===e.label).length??0
  facts.push(`${e.status==='added'?'Added':'Removed'} relationship: ${e.source_title??title(r,e.source_id)} → ${e.target_title??title(r,e.target_id)} · “${e.label}” · occurrence ${e.occurrence}; total ${count(r.before)} → ${count(r.with_changes)}.`)
 }
 const position=(p:{x:number;y:number}|null)=>p?`(${p.x}, ${p.y})`:'not visible'
 const size=(p:{width:number;height:number}|null)=>p?`${p.width} × ${p.height}`:'not visible'
 for(const p of c.node_positions??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.component_id)} position: ${position(p.before)} → ${position(p.with)}.`)
 for(const p of c.node_sizes??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.component_id)} size: ${size(p.before)} → ${size(p.with)}.`)
 for(const p of c.node_shapes??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.component_id)} shape: ${p.before_visible?p.before??'Default':'not visible'} → ${p.with_visible?p.with??'Default':'not visible'}.`)
 for(const p of c.edge_routes??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.source_id)} → ${title(r,p.target_id)} · “${p.label}” · occurrence ${p.occurrence} bend: ${p.before?`Custom ${p.before.bend}`:p.before_state} → ${p.with?`Custom ${p.with.bend}`:p.with_state}.`)
 for(const b of d.with?.boundaries??[]){const old=d.before?.boundaries.find(n=>n.component_id===b.component_id);if(old&&(old.home_diagram_id!==b.home_diagram_id||old.home_diagram_title!==b.home_diagram_title))facts.push(`${b.title} home context: ${old.home_diagram_title} → ${b.home_diagram_title}.`)}
 return <section className="print-facts" id={`changes-${d.id}`}><h3>{d.with?.title??d.before?.title}</h3>
 {facts.length>0&&<ul>{facts.map((f,i)=><li key={i}>{f}</li>)}</ul>}
 {d.textIDs.map(id=>textHome(r,id)===d.id?<section id={`source-${id}`} key={id}><h4>{title(r,id)} · Component text</h4><SourceDiff before={source(r.before,id)} after={source(r.with_changes,id)}/></section>:<p key={id}><a href={`#source-${id}`}>{title(r,id)} · text changes in {diagramTitle(r,textHome(r,id))}</a></p>)}
 {(c.diagram_notes??[]).filter(n=>n.diagram_id===d.id).map(n=><section key={n.note_id}><h4>Diagram note · {n.before?n.with?'changed':'removed':'added'}</h4><p>Position: {position(n.before)} → {position(n.with)}. Size: {size(n.before)} → {size(n.with)}.</p>{n.before?.text!==n.with?.text&&<SourceDiff before={n.before?.text??''} after={n.with?.text??''}/>}</section>)}
 </section>
}

function DiagramDrawings({diagram:d,review:r,onReady}:{diagram:AffectedDiagram;review:ChangeReview;onReady:(id:string,ok:boolean)=>void}) {
 const [images,setImages]=useState<string[]>(),[error,setError]=useState(''),[warnings,setWarnings]=useState<string[]>([])
 useEffect(()=>{
  let cancelled=false
  const c=r.comparison
  const changedIDs=new Set([...(c.appearances??[]),...(c.node_positions??[]),...(c.node_sizes??[]),...(c.node_shapes??[])].filter(p=>p.diagram_id===d.id).map(p=>p.component_id))
  for(const n of c.diagram_notes??[])if(n.diagram_id===d.id)changedIDs.add(`note:${n.note_id}`)
  const components=[...c.components,...d.textIDs.filter(id=>!c.components.some(c=>c.component_id===id)).map(id=>({component_id:id,status:'content_changed' as const,path:''}))]
  const routeKeys=[...(d.before?.relationships??[]),...(d.with?.relationships??[])].filter(e=>c.edge_routes?.some(p=>p.diagram_id===d.id&&p.source_id===e.source_component_id&&p.target_id===e.target_component_id&&p.label===e.label&&p.occurrence===e.routing?.occurrence)).map(e=>e.key)
  printDiagramImages(mapComponentsForDiagram(r.before,d.before),mapComponentsForDiagram(r.with_changes,d.with),{reviewDiagramID:d.id,reviewComponents:components,reviewPositionIDs:[...changedIDs],reviewRouteKeys:routeKeys,reviewRelationships:c.relationships}).then(({images,warnings})=>{if(!cancelled){setImages(images);setWarnings(warnings);onReady(d.id,true)}}).catch(e=>{if(!cancelled){setError(e instanceof Error?e.message:'The drawing failed.');onReady(d.id,false)}})
  return()=>{cancelled=true}
 },[r,d.id])
 return <section className="print-diagram" aria-label={`${d.with?.title??d.before?.title} drawings`}>
 {warnings.map(w=><p role="status" key={w}>{w}</p>)}
 {error?<p role="alert">Diagram could not be rendered. {error} Printing is unavailable; the exact changes remain below.</p>:['Before','With changes'].map((side,i)=><figure className="print-drawing" key={side}><figcaption><h3>{d.with?.title??d.before?.title} · {side}</h3><p className="print-key">Same frame · green stroke: presentation/composition changed · gold: text changed · green fill: added · dashed red link: removed</p></figcaption>{!(i===0?d.before:d.with)?<p>Diagram does not exist on this side.</p>:images?<img src={images[i]} alt={`${side}: ${i===0?d.before?.title:d.with?.title}`}/>:<p>Preparing drawing…</p>}</figure>)}
 </section>
}

export function PrintableProposal() {
 const [value,setValue]=useState<PrintDocument>(),[error,setError]=useState(''),[printError,setPrintError]=useState(''),[ready,setReady]=useState<Record<string,boolean>>({})
 const match=window.location.pathname.match(/^\/projects\/([^/]+)\/proposals\/([^/]+)(?:\/reviews\/([^/]+))?\/print\/?$/)
 const returnURL=match?`/projects/${match[1]}/proposals/${match[2]}`:'/'
 useEffect(()=>{
  let cancelled=false
  async function load(){
   if(!match)throw new Error('This printable link is invalid.')
   const slug=decodeURIComponent(match[1]),id=decodeURIComponent(match[2]),query=new URLSearchParams(window.location.search)
   if(match[3]){if(query.size)throw new Error('This printable link contains conflicting versions.');query.set('review_id',decodeURIComponent(match[3]))}
   const response=await fetch('/api/projects'),catalog=await response.json()
   const projects=catalog.projects.filter((p:{slug:string})=>p.slug===slug)
   if(!response.ok||projects.length!==1||projects[0].unavailable||projects[0].conflict)throw new Error('The project is unavailable. Return to the proposal to inspect its context.')
   query.set('project_slug',slug);query.set('store_id',projects[0].store_id);query.set('change_set_id',id)
   const result=await fetch(`/api/architecture/print?${query}`),body=await result.json()
   if(!result.ok)throw new Error(body.code==='review_required'?'This proposal has no prepared Review. Return to the proposal.':body.code==='review_invalidated'?'This proposal version has moved. Return to the proposal for its current state.':'This exact printable version is unavailable. Return to the proposal to inspect its context.')
   await document.fonts.ready
   if(!cancelled)setValue(body)
  }
  load().catch(e=>{if(!cancelled)setError(e.message)})
  return()=>{cancelled=true}
 },[])
 const diagrams=value?affectedDiagrams(value.review):[]
 const allReady=!!value&&diagrams.every(d=>ready[d.id]===true)
 async function print(){
  try{await document.fonts.ready;await Promise.all([...document.querySelectorAll<HTMLImageElement>('.print-drawing img')].map(image=>image.decode()));await new Promise<void>(resolve=>requestAnimationFrame(()=>resolve()));window.print()}
  catch{setPrintError('An image could not finish loading. Reload this exact page before printing.')}
 }
 return <main className="print-document"><nav className="print-actions"><a href={returnURL}>Return to proposal</a>{allReady&&<button onClick={print}>Print / Save as PDF</button>}</nav>
 {printError&&<p role="alert">{printError}</p>}
 {error?<h1 role="alert">{error}</h1>:!value?<p>Loading exact proposal…</p>:<>
 <header className="print-header"><p>{value.project_name} · {value.review_id?'Submitted review':value.lifecycle==='applied'?'Applied proposal':'Active proposal'} · generation {value.review.generation}</p><p>{value.name}</p></header>
 <section className="print-design" aria-label="Proposal design"><MarkdownBody source={value.proposal_markdown}/></section>
 {value.review.diff===''?<p>No Architecture changes; this proposal contains only proposal text.</p>:<>
 <h2 className={diagrams.length?'print-diagrams-heading':''}>Affected Diagrams</h2>{diagrams.length===0?<p>No Diagram presentation or content changes. The complete canonical changes are retained in the technical appendix.</p>:diagrams.map(d=><DiagramDrawings key={d.id} diagram={d} review={value.review} onReady={(id,ok)=>setReady(old=>({...old,[id]:ok}))}/>)}
 {diagrams.length>0&&<section className="print-changes"><h2>Changes by Diagram</h2>{diagrams.map(d=><DiagramFacts key={d.id} diagram={d} review={value.review}/>)}</section>}
 <section className="print-appendix"><h2>Technical appendix · complete canonical diff</h2><RawDiff diff={value.review.diff}/></section></>}
 <footer className="print-binding"><h2>Exact version</h2><p>Before base: {value.review.base_revision}<br/>With candidate: {value.review.candidate_tree}<br/>{value.review_id?'Immutable reviewed state':value.lifecycle==='applied'?'Applied receipt state':'Reviewed state'}: {value.state}{value.applied_revision&&<><br/>Applied revision: {value.applied_revision}</>}</p><p>Current Accepted observed separately: {value.accepted_revision}. {value.accepted_revision!==value.review.base_revision?'It differs from this document’s Before base.':''} Current proposal: {value.lifecycle==='no_longer_active'?'no longer active':value.lifecycle}.</p></footer>
 {!allReady&&diagrams.length>0&&<p className="print-pending" role="status">Printing is unavailable until every drawing is ready.</p>}
 </>}
 </main>
}
