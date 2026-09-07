import {useState} from 'react'
import {createRoot} from 'react-dom/client'
import {ArchitectureMap, type MapComponent} from '../../src/ArchitectureMap'
import '../../src/styles.css'

function RoutingInteraction(){
 const config=(window as any).routingCase??{}
 const [side,setSide]=useState<'before'|'with'>('before')
 const [selected,setSelected]=useState<string>()
 const components=(bend:number):MapComponent[]=>[
  {id:'a',title:'Gateway',position:{x:0,y:0},size:{width:320,height:160},relationships:config.reverse?[]:[{target_id:'b',label:'calls',projection_key:'edge',routing:{diagram_id:'diagram',source_id:'a',target_id:'b',label:'calls',occurrence:1,count:1,route:{bend},display_bend:config.initialCoincident?0:bend,eligible:!config.initialCoincident}}]},
  {id:'b',title:'Worker',position:{x:config.initialCoincident?0:600,y:0},size:{width:160,height:80},node_kind:config.boundary?'boundary':'home',boundary_home_title:config.boundary?'Other diagram':undefined,relationships:config.reverse?[{target_id:'a',label:'calls',projection_key:'edge',routing:{diagram_id:'diagram',source_id:'b',target_id:'a',label:'calls',occurrence:1,count:1,route:{bend},display_bend:config.initialCoincident?0:bend,eligible:!config.initialCoincident}}]:[]},
 ]
 const bend=config.review?(side==='before'?100000:-100000):(config.bend??80)
 return <><button onClick={()=>setSide(side==='before'?'with':'before')}>Switch side</button><button onClick={()=>setSelected(selected?undefined:'a')}>Toggle node selection</button><div style={{width:900,height:650}}>
  <ArchitectureMap revision={side} viewKey="routing-test" components={components(bend)} selectedID={selected} onSelect={setSelected} selectedRelationshipKey="edge"
    onRoute={async(route,bend)=>{(window as any).routingSubmissions.push({route,bend});return true}}
    {...(config.review?{reviewSide:side,reviewOtherComponents:components(-bend)}:{})}/>
 </div></>
}
(window as any).routingSubmissions=[]
createRoot(document.getElementById('root')!).render(<RoutingInteraction/> )
