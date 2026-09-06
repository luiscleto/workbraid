import { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { ArchitectureMap } from '../../src/ArchitectureMap'
import '../../src/styles.css'

declare global {
  interface Window { submissions: { id: string; size: {width:number;height:number} }[] }
}

function SizingInteraction() {
  const [selected, setSelected] = useState<string | undefined>('a')
  const [editable, setEditable] = useState(true)
  const [render, setRender] = useState(0)
  return <>
    <button onClick={() => setSelected('b')}>Select B</button>
    <button onClick={() => setSelected(undefined)}>Clear selection</button>
    <button onClick={() => setEditable(false)}>Leave editing</button>
    <button onClick={() => setRender(render + 1)}>Rerender callback</button>
    <div style={{width:900,height:650}}>
      <ArchitectureMap revision="fixed" viewKey="sizing-test" components={[
        {id:'a',title:'A',position:{x:0,y:0},size:{width:200,height:96},relationships:[]},
        {id:'b',title:'B',position:{x:320,y:0},size:{width:200,height:96},relationships:[]},
      ]} selectedID={selected} onSelect={setSelected}
        onResize={editable ? async (id,size) => {window.submissions.push({id,size});return true} : undefined} />
    </div>
  </>
}
window.submissions = []
createRoot(document.getElementById('root')!).render(<SizingInteraction />)
