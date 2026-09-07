import {afterEach,expect,it,vi} from 'vitest'
import {cleanup,render,screen} from '@testing-library/react'
import {affectedDiagrams,sourceDiff,PrintableProposal} from './PrintableProposal'
import {type ChangeReview,type DiagramProjection,type ReviewSnapshot} from './App'

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
it.each([[' a\r\n',' a\n'],['# A\n','A\n=\n'],['a\t \n','a\n'],['a\n','a'],['','😀\r\n'],['old','']])('preserves exact source bytes %j → %j',(before,after)=>{
 const lines=sourceDiff(before,after)
 expect(lines.filter(l=>l.kind!=='added').map(l=>l.text).join('')).toBe(before)
 expect(lines.filter(l=>l.kind!=='removed').map(l=>l.text).join('')).toBe(after)
})
it.each([{diff:'',message:'No Architecture changes; this proposal contains only proposal text.',appendix:false},{diff:'diff --git a/architecture.yaml b/architecture.yaml\nold mode 100644\nnew mode 100755\n',message:'No Diagram presentation or content changes. The complete canonical changes are retained in the technical appendix.',appendix:true}])('keeps zero-Diagram print truthful (appendix=$appendix)',async({diff,message,appendix})=>{
 window.history.replaceState({},'', '/projects/example/proposals/proposal/print?reviewed_state=state')
 Object.defineProperty(document,'fonts',{configurable:true,value:{ready:Promise.resolve()}})
 const r=review();r.diff=diff
 const fetcher=vi.fn().mockResolvedValueOnce({ok:true,json:async()=>({projects:[{slug:'example',store_id:'store'}]})}).mockResolvedValueOnce({ok:true,json:async()=>({project_name:'Example',name:'Text only',proposal_markdown:'# Design first\n\n<script>bad()</script>\n\n![hidden](https://example.invalid/x)',lifecycle:'active',state:'state',accepted_revision:'new accepted',review:r})})
 vi.stubGlobal('fetch',fetcher)
 render(<PrintableProposal/>)
 expect(await screen.findByText(message)).toBeInTheDocument()
 expect(screen.getByRole('heading',{name:'Design first'})).toBeInTheDocument()
 expect(screen.queryAllByRole('img')).toHaveLength(0)
 expect(screen.queryByTestId('raw-diff')!==null).toBe(appendix)
 expect(screen.getByRole('button',{name:'Print / Save as PDF'})).toBeInTheDocument()
 expect(document.querySelector('.print-design script')).toBeNull()
 expect(fetcher.mock.calls.every(call=>call.length===1)).toBe(true)
 expect(fetcher.mock.calls[1][0]).toContain('/api/architecture/print?reviewed_state=state')
 vi.unstubAllGlobals()
})
