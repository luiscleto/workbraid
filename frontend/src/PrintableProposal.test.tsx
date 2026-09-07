import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,fireEvent,render,screen} from '@testing-library/react'
import {affectedDiagrams,sourceDiff,visibleSource,PrintableProposal} from './PrintableProposal'
import {type ChangeReview,type DiagramProjection,type ReviewSnapshot} from './App'
import {printProjectionElements,type MapComponent} from './ArchitectureMap'

afterEach(()=>{cleanup();vi.restoreAllMocks()})
const diagram=(id:string):DiagramProjection=>({id,title:id,filename:`${id}.yaml`,depth:0,breadcrumbs:[],appearances:[],boundaries:[],relationships:[]})
const snapshot=(diagrams:DiagramProjection[]):ReviewSnapshot=>({revision:'base',components:[],component_count:0,component_titles:[],diagrams})
const review=():ChangeReview=>({reviewed_state:'state',base_revision:'base',candidate_tree:'tree',generation:2,diff:'nonempty',before:snapshot([diagram('root'),diagram('child')]),with_changes:snapshot([diagram('root'),diagram('child')]),comparison:{components:[],relationships:[]}})

it('includes visible boundary source-only H1 changes without Diagram blob or plain-title changes',()=>{
 const r=review(),component={id:'c',title:'Name',filename:'name.md',description:'body',relationships:[],markdown_source:'# *Name*\nbody'}
 r.before.components=[component];r.with_changes.components=[{...component,markdown_source:'Name\n====\nbody'}]
 r.before.diagrams![1].boundaries=[{key:'boundary:c',component_id:'c',title:'Name',home_diagram_id:'root',home_diagram_title:'root'}]
 r.with_changes.diagrams![1].boundaries=r.before.diagrams![1].boundaries
 r.before.diagrams![0].appearances=[{component_id:'c',role:'home'}];r.with_changes.diagrams![0].appearances=r.before.diagrams![0].appearances
 expect(affectedDiagrams(r).map(d=>[d.id,d.textIDs])).toEqual([['root',['c']],['child',['c']]])
})
it('includes exact visible Relationship changes without Diagram file changes, but excludes unrelated context',()=>{
 const r=review();r.comparison.relationships=[{key:'edge',source_id:'a',target_id:'b',label:'calls',status:'added',path:'components/a.md',occurrence:2,diagram_projections:[{side:'with',diagram_id:'child',key:'edge',source_node_key:'a',target_node_key:'boundary:b'}]}]
 expect(affectedDiagrams(r).map(d=>d.id)).toEqual(['child'])
})
it('retains Before-only and With-only Diagrams and derived parent changes',()=>{
 const r=review();r.before.diagrams!.push(diagram('removed'));r.with_changes.diagrams!.push(diagram('added'));r.with_changes.diagrams![1].parent_anchor_component_id='new-parent'
 expect(affectedDiagrams(r).map(d=>d.id)).toEqual(['child','added','removed'])
})
it('filters geometry, routes, shapes and note geometry while retaining note content and Diagram titles',()=>{
 const r=review(),address={diagram_id:'root',component_id:'c',path:'root.yaml'}
 r.comparison.node_positions=[{...address,before:{x:0,y:0},with:{x:10,y:20}}]
 r.comparison.node_sizes=[{...address,before:{width:100,height:80},with:{width:200,height:120}}]
 r.comparison.node_shapes=[{...address,before:null,with:'ellipse',before_visible:true,with_visible:true}]
 r.comparison.edge_routes=[{diagram_id:'root',source_id:'c',target_id:'d',label:'calls',occurrence:1,path:'root.yaml',before:null,with:{bend:80},before_state:'default',with_state:'custom'}]
 const note={text:'Same note',x:0,y:0,width:240,height:120}
 r.comparison.diagram_notes=[{diagram_id:'child',note_id:'n',path:'child.yaml',before:note,with:{...note,x:50}}]
 expect(affectedDiagrams(r)).toEqual([])
 expect(affectedDiagrams(r,true).map(d=>d.id)).toEqual(['root','child'])
 r.comparison.diagram_notes[0].with!.text='Changed note'
 expect(affectedDiagrams(r).map(d=>d.id)).toEqual(['child'])
 r.with_changes.diagrams![0].title='New root title'
 expect(affectedDiagrams(r).map(d=>d.id)).toEqual(['root','child'])
})
it('prints removed duplicate occurrences and base-only external endpoints without changing either snapshot',()=>{
 const before:MapComponent[]=[{id:'a',component_id:'a',title:'A',position:{x:0,y:0},relationships:[{target_id:'boundary:b',label:'calls',projection_key:'edge1'},{target_id:'boundary:b',label:'calls',projection_key:'edge2'}]},{id:'boundary:b',component_id:'b',title:'B',node_kind:'boundary',boundary_home_title:'Elsewhere',position:{x:200,y:0},relationships:[]}]
 const after:MapComponent[]=[{...before[0],position:{x:50,y:60},relationships:[]}]
 const saved=JSON.stringify({before,after})
 const relationships=[1,2].map(n=>({key:`removed${n}`,source_id:'a',target_id:'b',label:'calls',occurrence:n,path:'a.md',status:'removed' as const,diagram_projections:[{side:'before' as const,diagram_id:'root',key:`edge${n}`,source_node_key:'a',target_node_key:'boundary:b'}]}))
 const elements=printProjectionElements(before,after,{reviewDiagramID:'root',reviewRelationships:relationships})
 const ghosts=elements.filter(e=>e.data.reviewStatus==='removed')
 expect(ghosts.filter(e=>e.data.source).map(e=>[e.data.occurrence,e.data.source,e.data.target])).toEqual([[1,'a','print-base:boundary:b'],[2,'a','print-base:boundary:b']])
 expect(ghosts.filter(e=>!e.data.source)).toHaveLength(1)
 expect(ghosts.find(e=>!e.data.source)?.position).toEqual({x:200,y:0})
 expect(elements.find(e=>e.data.id==='a')?.position).toEqual({x:50,y:60})
 expect(JSON.stringify({before,after})).toBe(saved)
 expect(printProjectionElements(before,after,{reviewSide:'before',reviewDiagramID:'root',reviewRelationships:relationships}).filter(e=>e.data.source)).toHaveLength(2)
})
it.each([[' a\r\n',' a\n'],['# A\n','A\n=\n'],['a\t \n','a\n'],['a\n','a'],['','😀\r\n'],['old','']])('preserves exact source bytes %j → %j',(before,after)=>{
 const lines=sourceDiff(before,after)
 expect(lines.filter(l=>l.kind!=='added').map(l=>l.text).join('')).toBe(before)
 expect(lines.filter(l=>l.kind!=='removed').map(l=>l.text).join('')).toBe(after)
})
it('keeps unchanged interior lines and highlights Unicode and CRLF edits within each changed line',()=>{
 const before='routes λ\r\nunchanged middle\n😀 stays old\n',after='validates λ\nunchanged middle\n😀 stays new\n'
 const lines=sourceDiff(before,after)
 expect(lines.find(l=>l.text==='unchanged middle\n')?.kind).toBe('context')
 for(const line of lines){if(line.parts)expect(line.parts.map(p=>p.text).join('')).toBe(line.text)}
 expect(lines.find(l=>l.text==='😀 stays new\n')?.parts?.filter(p=>p.kind==='context').map(p=>p.text).join('')).toContain('😀 stays ')
 expect(lines.find(l=>l.text==='routes λ\r\n')?.parts?.some(p=>p.kind==='removed'&&p.text==='\r')).toBe(true)
 expect(visibleSource(' \t')).not.toBe(visibleSource('·→'))
 expect(visibleSource('\\u{b7}')).not.toBe(visibleSource('·'))
})
it.each([{diff:'',message:'No Architecture changes; this proposal contains only proposal text.'},{diff:'diff --git a/architecture.yaml b/architecture.yaml\nold mode 100644\nnew mode 100755\n',message:'No content changes in Diagrams. Presentation-only or other Architecture changes are excluded from this view.'}])('keeps zero-Diagram print truthful ($diff)',async({diff,message})=>{
 window.history.replaceState({},'', '/projects/example/proposals/proposal/print?reviewed_state=state')
 Object.defineProperty(document,'fonts',{configurable:true,value:{ready:Promise.resolve()}})
 const r=review();r.diff=diff
 const fetcher=vi.fn().mockResolvedValueOnce({ok:true,json:async()=>({projects:[{slug:'example',store_id:'store'}]})}).mockResolvedValueOnce({ok:true,json:async()=>({project_name:'Example',name:'Text only',proposal_markdown:'# Design first\n\n<script>bad()</script>\n\n![hidden](https://example.invalid/x)',lifecycle:'active',state:'state',accepted_revision:'new accepted',review:r})})
 vi.stubGlobal('fetch',fetcher)
 render(<PrintableProposal/>)
 expect(await screen.findByText(message)).toBeInTheDocument()
 expect(screen.getByRole('heading',{name:'Design first'})).toBeInTheDocument()
 expect(screen.queryAllByRole('img')).toHaveLength(0)
 expect(screen.queryByTestId('raw-diff')).toBeNull()
 if(diff){fireEvent.click(screen.getByLabelText('Include complete raw diff'));expect(screen.getByTestId('raw-diff')).toHaveTextContent('old mode 100644')}
 expect(screen.getByRole('button',{name:'Print / Save as PDF'})).toBeInTheDocument()
 expect(document.querySelector('.print-design script')).toBeNull()
 expect(fetcher.mock.calls.every(call=>call.length===1)).toBe(true)
 expect(fetcher.mock.calls[1][0]).toContain('/api/architecture/print?reviewed_state=state')
 vi.unstubAllGlobals()
})
