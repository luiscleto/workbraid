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

export function affectedDiagrams(r:ChangeReview,includePresentation=false):AffectedDiagram[] {
 if(r.diff==='')return []
 const before=r.before.diagrams??[],after=r.with_changes.diagrams??[],c=r.comparison
 const textIDs=[...new Set([...r.before.components,...r.with_changes.components].map(c=>c.id))].filter(id=>source(r.before,id)!==source(r.with_changes,id))
 const structural=new Set([...(c.appearances??[]),...(includePresentation?[...(c.node_positions??[]),...(c.node_sizes??[]),...(c.edge_routes??[]),...(c.node_shapes??[])]:[]),...(c.diagram_notes??[]).filter(n=>includePresentation||n.before?.text!==n.with?.text)].map(c=>c.diagram_id))
 const union=[...after,...before.filter(d=>!after.some(a=>a.id===d.id))],ordered:DiagramProjection[]=[],seen=new Set<string>()
 const visit=(d:DiagramProjection)=>{if(seen.has(d.id))return;seen.add(d.id);ordered.push(d);for(const child of union)if(child.parent_diagram_id===d.id)visit(child)}
 for(const d of union)if(!d.parent_diagram_id||!union.some(p=>p.id===d.parent_diagram_id))visit(d)
 for(const d of union)visit(d)
 return ordered.map(d=>{
  const b=before.find(x=>x.id===d.id),w=after.find(x=>x.id===d.id),v=new Set([...visible(b),...visible(w)])
  return {id:d.id,before:b,with:w,textIDs:textIDs.filter(id=>v.has(id))}
 }).filter(d=>!d.before||!d.with||d.before.title!==d.with.title||structural.has(d.id)||d.textIDs.length||d.before.parent_anchor_component_id!==d.with.parent_anchor_component_id||d.before.parent_diagram_id!==d.with.parent_diagram_id||JSON.stringify(d.before.boundaries.map(b=>[b.component_id,b.home_diagram_id,b.home_diagram_title]).sort())!==JSON.stringify(d.with.boundaries.map(b=>[b.component_id,b.home_diagram_id,b.home_diagram_title]).sort())||c.relationships.some(e=>e.diagram_projections?.some(p=>p.diagram_id===d.id)))
}

type SourcePart = {kind:'context'|'removed'|'added';text:string}
type SourceLine = SourcePart & {parts?:SourcePart[]}

// Exact source presentation only. The bounded table avoids quadratic memory
// for very large bodies; the fallback still returns every exact source byte.
function compareSourceParts(a:string[],b:string[]):SourcePart[] {
 let start=0,end=0
 while(start<a.length&&start<b.length&&a[start]===b[start])start++
 while(end<a.length-start&&end<b.length-start&&a[a.length-end-1]===b[b.length-end-1])end++
 const left=a.slice(start,a.length-end),right=b.slice(start,b.length-end)
 const result:SourcePart[]=a.slice(0,start).map(text=>({kind:'context',text}))
 if(left.length*right.length>1000000){
  result.push(...left.map(text=>({kind:'removed' as const,text})),...right.map(text=>({kind:'added' as const,text})))
 }else{
  const columns=right.length+1,table=new Uint32Array((left.length+1)*columns)
  for(let i=left.length-1;i>=0;i--)for(let j=right.length-1;j>=0;j--)table[i*columns+j]=left[i]===right[j]?1+table[(i+1)*columns+j+1]:Math.max(table[(i+1)*columns+j],table[i*columns+j+1])
  let i=0,j=0
  while(i<left.length||j<right.length){
   if(i<left.length&&j<right.length&&left[i]===right[j]){result.push({kind:'context',text:left[i++]});j++}
   else if(i<left.length&&(j===right.length||table[(i+1)*columns+j]>=table[i*columns+j+1]))result.push({kind:'removed',text:left[i++]})
   else result.push({kind:'added',text:right[j++]})
  }
 }
 result.push(...a.slice(a.length-end).map(text=>({kind:'context' as const,text})))
 return result
}

export function sourceDiff(before:string,after:string):SourceLine[] {
 const lines:SourceLine[]=compareSourceParts(before.match(/[^\n]*\n|[^\n]+$/g)??[],after.match(/[^\n]*\n|[^\n]+$/g)??[])
 for(let i=0;i<lines.length;){
  if(lines[i].kind==='context'){i++;continue}
  let end=i;while(end<lines.length&&lines[end].kind!=='context')end++
  const removed=lines.slice(i,end).filter(l=>l.kind==='removed'),added=lines.slice(i,end).filter(l=>l.kind==='added')
  for(let n=0;n<Math.min(removed.length,added.length);n++){
   const words=(text:string)=>text.match(/[\p{L}\p{N}_]+|[^\p{L}\p{N}_]/gu)??[]
   const parts=compareSourceParts(words(removed[n].text),words(added[n].text))
   removed[n].parts=parts.filter(p=>p.kind!=='added');added[n].parts=parts.filter(p=>p.kind!=='removed')
  }
  i=end
 }
 return lines
}

export function visibleSource(text:string) {
 return Array.from(text).map(char=>char===' '?'·':char==='\t'?'→':char==='\r'?'␍':char==='\n'?'↵':char==='\\'?'\\\\':'·→␍↵∎'.includes(char)?`\\u{${char.codePointAt(0)!.toString(16)}}`:char).join('')
}

function SourceDiff({before,after}:{before:string;after:string}) {
 return <div className="print-source-diff"><p className="print-key">Source text · − deleted / + added, with inline changes highlighted. Spaces ·, tabs →, CR ␍, LF ↵; ∎ means no final newline. Literal marker characters use Unicode escapes; backslashes are doubled.</p>
 <pre>{sourceDiff(before,after).map((line,i)=><span key={i} className={`diff-line diff-${line.kind}`}><span aria-hidden="true">{line.kind==='added'?'+ ':line.kind==='removed'?'− ':'  '}</span>{(line.parts??[{kind:line.kind,text:line.text}]).map((part,n)=><span key={n} className={`source-${part.kind}`}>{visibleSource(part.text)}</span>)}{line.text.endsWith('\n')?'':'∎'}{'\n'}</span>)}</pre></div>
}

function textHome(r:ChangeReview,id:string) {
 return (r.with_changes.diagrams??[]).find(d=>d.appearances.some(a=>a.component_id===id&&a.role==='home'))?.id??(r.before.diagrams??[]).find(d=>d.appearances.some(a=>a.component_id===id&&a.role==='home'))?.id
}
function DiagramFacts({diagram:d,review:r,includePresentation}:{diagram:AffectedDiagram;review:ChangeReview;includePresentation:boolean}) {
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
 if(includePresentation){
 for(const p of c.node_positions??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.component_id)} position: ${position(p.before)} → ${position(p.with)}.`)
 for(const p of c.node_sizes??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.component_id)} size: ${size(p.before)} → ${size(p.with)}.`)
 for(const p of c.node_shapes??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.component_id)} shape: ${p.before_visible?p.before??'Default':'not visible'} → ${p.with_visible?p.with??'Default':'not visible'}.`)
 for(const p of c.edge_routes??[])if(p.diagram_id===d.id)facts.push(`${title(r,p.source_id)} → ${title(r,p.target_id)} · “${p.label}” · occurrence ${p.occurrence} bend: ${p.before?`Custom ${p.before.bend}`:p.before_state} → ${p.with?`Custom ${p.with.bend}`:p.with_state}.`)
 }
 for(const b of d.with?.boundaries??[]){const old=d.before?.boundaries.find(n=>n.component_id===b.component_id);if(old&&(old.home_diagram_id!==b.home_diagram_id||old.home_diagram_title!==b.home_diagram_title))facts.push(`${b.title} home context: ${old.home_diagram_title} → ${b.home_diagram_title}.`)}
 const sameBoundary=(a:DiagramProjection['boundaries'][number],b:DiagramProjection['boundaries'][number])=>a.component_id===b.component_id&&a.title===b.title&&a.home_diagram_id===b.home_diagram_id&&a.home_diagram_title===b.home_diagram_title
 const contexts=(d.before?.boundaries??[]).map(boundary=>({boundary,side:d.with?.boundaries.some(b=>sameBoundary(b,boundary))?'Before and With changes':'Before'}))
 for(const boundary of d.with?.boundaries??[])if(!d.before?.boundaries.some(b=>sameBoundary(b,boundary)))contexts.push({boundary,side:'With changes'})
 return <section className="print-facts" id={`changes-${d.id}`}><h3>{d.with?.title??d.before?.title}</h3>
 {facts.length>0&&<ul>{facts.map((f,i)=><li key={i}>{f}</li>)}</ul>}
 {contexts.some(c=>c.side!=='Before and With changes')&&<section><h4>Boundary context changes</h4><ul>{contexts.filter(c=>c.side!=='Before and With changes').map(({boundary:b,side},i)=><li key={i}>{side}: {b.title} · Lives in {b.home_diagram_title}</li>)}</ul></section>}
 {d.textIDs.map(id=>textHome(r,id)===d.id?<section id={`source-${id}`} key={id}><h4>{title(r,id)} · Component text</h4><SourceDiff before={source(r.before,id)} after={source(r.with_changes,id)}/></section>:<p key={id}><a href={`#source-${id}`}>{title(r,id)} · text changes in {diagramTitle(r,textHome(r,id))}</a></p>)}
 {(c.diagram_notes??[]).filter(n=>n.diagram_id===d.id&&(includePresentation||n.before?.text!==n.with?.text)).map(n=><section key={n.note_id}><h4>Diagram note · {n.before?n.with?'changed':'removed':'added'}</h4>{includePresentation&&<p>Position: {position(n.before)} → {position(n.with)}. Size: {size(n.before)} → {size(n.with)}.</p>}{n.before?.text!==n.with?.text&&<SourceDiff before={n.before?.text??''} after={n.with?.text??''}/>}</section>)}
 </section>
}

function DiagramDrawings({diagram:d,review:r,includePresentation,onReady}:{diagram:AffectedDiagram;review:ChangeReview;includePresentation:boolean;onReady:(id:string,ok:boolean)=>void}) {
 const [images,setImages]=useState<string[]>(),[error,setError]=useState(''),[warnings,setWarnings]=useState<string[]>([])
 useEffect(()=>{
  let cancelled=false
  const c=r.comparison
  const changedIDs=new Set([...(c.appearances??[]),...(includePresentation?[...(c.node_positions??[]),...(c.node_sizes??[]),...(c.node_shapes??[])]:[])].filter(p=>p.diagram_id===d.id).map(p=>p.component_id))
  for(const n of c.diagram_notes??[])if(n.diagram_id===d.id&&(includePresentation||n.before?.text!==n.with?.text))changedIDs.add(`note:${n.note_id}`)
  const components=[...c.components,...d.textIDs.filter(id=>!c.components.some(c=>c.component_id===id)).map(id=>({component_id:id,status:'content_changed' as const,path:''})),...(c.diagram_notes??[]).filter(n=>n.diagram_id===d.id&&n.with&&n.before?.text!==n.with.text).map(n=>({component_id:`note:${n.note_id}`,status:n.before?'content_changed' as const:'added' as const,path:''}))]
  const routeKeys=[...(d.before?.relationships??[]),...(d.with?.relationships??[])].filter(e=>c.edge_routes?.some(p=>p.diagram_id===d.id&&p.source_id===e.source_component_id&&p.target_id===e.target_component_id&&p.label===e.label&&p.occurrence===e.routing?.occurrence)).map(e=>e.key)
  printDiagramImages(mapComponentsForDiagram(r.before,d.before),mapComponentsForDiagram(r.with_changes,d.with),{reviewSide:d.with?'with':'before',reviewDiagramID:d.id,reviewComponents:components,reviewPositionIDs:[...changedIDs],reviewRouteKeys:includePresentation?routeKeys:[],reviewRelationships:c.relationships}).then(({images,warnings})=>{if(!cancelled){setImages(images);setWarnings(warnings);onReady(d.id,true)}}).catch(e=>{if(!cancelled){setError(e instanceof Error?e.message:'The drawing failed.');onReady(d.id,false)}})
  return()=>{cancelled=true}
 },[r,d.id,includePresentation])
 const diagram=d.with??d.before!,side=d.with?'With changes':'Removed Diagram · Before only'
 return <section className="print-diagram" aria-label={`${d.with?.title??d.before?.title} drawings`}>
 {warnings.map(w=><p role="status" key={w}>{w}</p>)}
 {error?<p role="alert">Diagram could not be rendered. {error} Printing is unavailable; the exact changes remain below.</p>:<figure className="print-drawing">
   <figcaption><h3>{diagram.title} · {side}</h3><p className="print-key">Gold: content changed · green fill: added · green stroke: composition{includePresentation?' / presentation':''} changed · dashed red: removed Relationship / base-only endpoint</p></figcaption>
   {diagram.appearances.length+diagram.boundaries.length+(diagram.notes?.length??0)===0&&<p>No visible Components or notes in this {d.with?'candidate':'base'} Diagram.</p>}
   {images?<img src={images[0]} alt={`${side}: ${diagram.title}`}/>:<p>Preparing drawing…</p>}
  </figure>}
 </section>
}

export function PrintableProposal() {
 const [value,setValue]=useState<PrintDocument>(),[error,setError]=useState(''),[printError,setPrintError]=useState(''),[ready,setReady]=useState<Record<string,boolean>>({})
 const [includePresentation,setIncludePresentation]=useState(false),[includeDiff,setIncludeDiff]=useState(false)
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
 const diagrams=value?affectedDiagrams(value.review,includePresentation):[]
 const allReady=!!value&&diagrams.every(d=>ready[d.id]===true)
 async function print(){
  try{await document.fonts.ready;await Promise.all([...document.querySelectorAll<HTMLImageElement>('.print-drawing img')].map(image=>image.decode()));await new Promise<void>(resolve=>requestAnimationFrame(()=>resolve()));window.print()}
  catch{setPrintError('An image could not finish loading. Reload this exact page before printing.')}
 }
 return <main className="print-document"><nav className="print-actions"><a href={returnURL}>Return to proposal</a>{allReady&&<button onClick={print}>Print / Save as PDF</button>}</nav>
 {value&&<div className="print-options"><label><input type="checkbox" checked={includePresentation} onChange={e=>{setReady({});setIncludePresentation(e.target.checked)}}/> Include presentation changes</label><label><input type="checkbox" checked={includeDiff} onChange={e=>setIncludeDiff(e.target.checked)}/> Include complete raw diff</label></div>}
 {printError&&<p role="alert">{printError}</p>}
 {error?<h1 role="alert">{error}</h1>:!value?<p>Loading exact proposal…</p>:<>
 <header className="print-header"><p>{value.project_name} · {value.review_id?'Submitted review':value.lifecycle==='applied'?'Applied proposal':'Active proposal'} · generation {value.review.generation}</p><p>{value.name}</p></header>
 <section className="print-design" aria-label="Proposal design"><MarkdownBody source={value.proposal_markdown}/></section>
 {value.review.diff===''?<p>No Architecture changes; this proposal contains only proposal text.</p>:<>
 <p className="print-scope">{includePresentation?'Content and presentation changes included.':'Presentation-only changes excluded.'} Complete exact diff is available in normal Review{includeDiff?' and the appendix below.':'; use Include complete raw diff to print it.'}</p>
 <h2 className={diagrams.length?'print-diagrams-heading':''}>Affected Diagrams</h2>{diagrams.length===0?<p>{includePresentation?'No affected Diagrams. Other Architecture changes remain in the complete exact diff.':'No content changes in Diagrams. Presentation-only or other Architecture changes are excluded from this view.'}</p>:diagrams.map(d=><DiagramDrawings key={`${d.id}:${includePresentation}`} diagram={d} review={value.review} includePresentation={includePresentation} onReady={(id,ok)=>setReady(old=>({...old,[id]:ok}))}/>)}
 {diagrams.length>0&&<section className="print-changes"><h2>Changes by Diagram</h2>{diagrams.map(d=><DiagramFacts key={d.id} diagram={d} review={value.review} includePresentation={includePresentation}/>)}</section>}
 {includeDiff&&<section className="print-appendix"><h2>Technical appendix · complete canonical diff</h2><RawDiff diff={value.review.diff}/></section>}</>}
 <footer className="print-binding"><h2>Exact version</h2><p>Before base: {value.review.base_revision}<br/>With candidate: {value.review.candidate_tree}<br/>{value.review_id?'Immutable reviewed state':value.lifecycle==='applied'?'Applied receipt state':'Reviewed state'}: {value.state}{value.applied_revision&&<><br/>Applied revision: {value.applied_revision}</>}</p><p>Current Accepted observed separately: {value.accepted_revision}. {value.accepted_revision!==value.review.base_revision?'It differs from this document’s Before base.':''} Current proposal: {value.lifecycle==='no_longer_active'?'no longer active':value.lifecycle}.</p></footer>
 {!allReady&&diagrams.length>0&&<p className="print-pending" role="status">Printing is unavailable until every drawing is ready.</p>}
 </>}
 </main>
}
