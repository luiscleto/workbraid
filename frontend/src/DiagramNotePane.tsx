import {useEffect,useState} from 'react'

export type DiagramNote = {id:string;text:string;x:number;y:number;width:number;height:number}
export function DiagramNotePane({note,readOnly,busy,onDirty,onKeep,onDelete,onClear}:{note?:DiagramNote;readOnly:boolean;busy:boolean;onDirty:(v:boolean)=>void;onKeep:(v:Omit<DiagramNote,'id'>)=>Promise<boolean>;onDelete:()=>Promise<boolean>;onClear:()=>void}) {
 const [text,setText]=useState(note?.text??'')
 const [geometry,setGeometry]=useState({x:String(note?.x??0),y:String(note?.y??0),width:String(note?.width??240),height:String(note?.height??120)})
 const [deleting,setDeleting]=useState(false)
 useEffect(()=>()=>onDirty(false),[onDirty])
 return <article className="component-documentation"><div className="pane-heading pane-heading-with-action"><div><p className="eyebrow">Diagram note</p><h2>{note?'Note':'Add note'}</h2></div><button className="text-action" type="button" onClick={onClear}>Clear selection</button></div>
 {readOnly?<pre style={{whiteSpace:'pre-wrap',overflowWrap:'anywhere'}}>{text}</pre>:<form onSubmit={async e=>{e.preventDefault();if(!text.trim()||Array.from(text).length>2000)return;const values={text,x:Number(geometry.x),y:Number(geometry.y),width:Number(geometry.width),height:Number(geometry.height)};if(await onKeep(values))onDirty(false)}}>
 <label>Text<textarea aria-label="Note text" rows={8} required value={text} onChange={e=>{setText(e.target.value);onDirty(true)}}/></label><p className="field-hint">Plain text · up to 2000 characters</p>
 {note&&<details className="position-controls"><summary>Position and size</summary><div className="position-fields">{(['x','y','width','height'] as const).map(k=><label key={k}>{k[0].toUpperCase()+k.slice(1)}<input aria-label={`Note ${k}`} type="number" step={1} required min={k==='width'?120:k==='height'?48:-100000} max={k==='width'?800:k==='height'?600:100000} value={geometry[k]} onChange={e=>{setGeometry({...geometry,[k]:e.target.value});onDirty(true)}}/></label>)}</div></details>}
 <div className="button-group"><button className="inline-action" type="submit" disabled={busy||!text.trim()||Array.from(text).length>2000}>Keep note</button>{note&&<button className="text-action" type="button" disabled={busy} onClick={()=>setDeleting(true)}>Delete note</button>}</div>
 {deleting&&<div role="group" aria-label="Delete note confirmation"><p>Delete this note from the Diagram?</p><button className="secondary-action" type="button" disabled={busy} onClick={async()=>{if(await onDelete())onDirty(false)}}>Delete this note</button><button className="text-action" type="button" onClick={()=>setDeleting(false)}>Keep editing</button></div>}
 </form>}</article>
}
