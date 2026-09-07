import {expect,test} from '@playwright/test'
import {spawn,execFileSync,type ChildProcess} from 'node:child_process'
import {mkdtempSync} from 'node:fs'
import {createServer} from 'node:net'
import {tmpdir} from 'node:os'
import {dirname,join,resolve} from 'node:path'
import {fileURLToPath} from 'node:url'

const frontend=resolve(dirname(fileURLToPath(import.meta.url)),'..'),root=resolve(frontend,'..')
let directory:string,binary:string,origin:string,process:ChildProcess
test.beforeAll(async()=>{
 directory=mkdtempSync(join(tmpdir(),'workbraid-print-browser-'));binary=join(directory,'workbraid')
 execFileSync('go',['build','-o',binary,'./cmd/workbraid'],{cwd:root})
 const port=await new Promise<number>(resolve=>{const server=createServer();server.listen(0,'127.0.0.1',()=>{const address=server.address();if(!address||typeof address==='string')throw new Error('No port');server.close(()=>resolve(address.port))})})
 origin=`http://127.0.0.1:${port}`
 process=spawn(binary,['--listen',`127.0.0.1:${port}`,'--data-dir',join(directory,'data'),'--ui-dir',join(frontend,'dist')],{stdio:['ignore','pipe','pipe']})
 await expect.poll(async()=>{try{return (await fetch(origin+'/api/agent/v2/status')).ok}catch{return false}}).toBe(true)
})
test.afterAll(async()=>{if(process){const exit=new Promise(resolve=>process.once('exit',resolve));process.kill('SIGTERM');await exit}})
function call(...args:string[]){const result=JSON.parse(execFileSync(binary,['--server',origin,'--json',...args],{encoding:'utf8'}));expect(result.ok).toBe(true);return result}
function fixture(name:string){
 const project=call('project','create','--name',name),store=project.context.project.store_id,slug=project.context.project.slug
 const architecture=call('architecture','inspect').result,diagram=architecture.root_diagram_id
 const proposal=call('change-set','create','--store-id',store,'--accepted-revision',architecture.revision,'--name','Printable changes').result
 const inspect=()=>call('change-set','inspect','--store-id',store,'--change-set-id',proposal.id).result
 const state=()=>['--store-id',store,'--change-set-id',proposal.id,'--generation',String(inspect().generation)]
 return {store,slug,diagram,proposal,inspect,state}
}

for(const scenario of ['content only','presentation only','removed external relationship'] as const){
 test(`concise real PDF: ${scenario}`,async({page},info)=>{
  const f=fixture(`Concise ${scenario}`)
  call('component','create',...f.state(),'--diagram-id',f.diagram,'--title','Gateway','--description','Original responsibility.\n')
  call('component','create',...f.state(),'--diagram-id',f.diagram,'--title','Delivery')
  const component=(name:string)=>f.inspect().candidate.components.find((c:any)=>c.title===name).id
  const gateway=component('Gateway'),delivery=component('Delivery')
  call('diagram','create-detail',...f.state(),'--component-id',delivery,'--title','Delivery detail')
  const detail=f.inspect().candidate.diagrams.find((d:any)=>d.title==='Delivery detail').id
  call('component','create',...f.state(),'--diagram-id',detail,'--title','External worker')
  const worker=component('External worker')
  call('relationship','add',...f.state(),'--source-id',gateway,'--target-id',worker,'--label','dispatches')
  const base=call('change-set','review',...f.state()).result
  call('architecture','update','--store-id',f.store,'--change-set-id',f.proposal.id,'--base-revision',base.base_revision,'--candidate-tree',base.candidate_tree,'--generation',String(base.generation))
  const proposal=call('change-set','create','--store-id',f.store,'--accepted-revision',call('architecture','inspect').result.revision,'--name',scenario).result
  const inspect=()=>call('change-set','inspect','--store-id',f.store,'--change-set-id',proposal.id).result
  const state=()=>['--store-id',f.store,'--change-set-id',proposal.id,'--generation',String(inspect().generation)]
  call('change-set','edit-proposal',...state(),'--proposal',`# ${scenario}\n\nOne concise view of this exact proposal.`)
  if(scenario==='content only')call('component','edit',...state(),'--component-id',gateway,'--description','Validates requests before dispatch.\n')
  if(scenario==='removed external relationship')call('relationship','remove',...state(),'--source-id',gateway,'--target-id',worker,'--label','dispatches','--occurrence','1')
  if(scenario==='presentation only'){
   call('diagram','set-position',...state(),'--diagram-id',f.diagram,'--component-id',gateway,'--x','-300','--y','200')
   call('diagram','set-size',...state(),'--diagram-id',f.diagram,'--component-id',gateway,'--width','320','--height','160')
   call('diagram','set-shape',...state(),'--diagram-id',f.diagram,'--component-id',gateway,'--shape','ellipse')
   call('diagram','set-route',...state(),'--diagram-id',f.diagram,'--source-id',gateway,'--target-id',worker,'--label','dispatches','--occurrence','1','--bend','100')
  }
  const review=call('change-set','review',...state()).result,before=inspect()
  const refs=()=>execFileSync('git',['--git-dir',join(directory,'data','architecture',f.store+'.git'),'for-each-ref','--format=%(refname) %(objectname)'],{encoding:'utf8'})
  const initialRefs=refs(),requests:string[]=[],errors:string[]=[]
  page.on('request',r=>{if(r.method()!=='GET')requests.push(r.method()+' '+r.url())});page.on('pageerror',e=>errors.push(e.message))
  await page.goto(review.printable_url)
  await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toBeVisible()
  await expect(page.locator('.raw-diff')).toHaveCount(0)
  await expect(page.locator('.print-facts')).not.toContainText(['position:','size:','bend:'])
  if(scenario==='presentation only'){
   await expect(page.getByText('No content changes in Diagrams. Presentation-only or other Architecture changes are excluded from this view.')).toBeVisible()
   await expect(page.locator('.print-drawing img')).toHaveCount(0)
   await expect(page.getByText('No Architecture changes; this proposal contains only proposal text.')).toHaveCount(0)
  }else{
   await expect(page.locator('.print-drawing img')).toHaveCount(2)
   if(scenario==='removed external relationship'){
    await expect(page.locator('.print-facts').first()).toContainText('Removed relationship: Gateway → External worker')
    await page.locator('.print-drawing').first().screenshot({path:info.outputPath('removed-external-ghost.png')})
   }
  }
  await page.pdf({path:info.outputPath('default.pdf'),preferCSSPageSize:true,printBackground:true})
  await page.getByLabel('Include presentation changes').check()
  await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toBeVisible()
  const diagramCount=scenario==='presentation only'?1:2
  await expect(page.locator('.print-drawing img')).toHaveCount(diagramCount)
  if(scenario==='presentation only')await expect(page.locator('.print-facts')).toContainText('Gateway position:')
  await page.pdf({path:info.outputPath('presentation-opt-in.pdf'),preferCSSPageSize:true,printBackground:true})
  await page.getByLabel('Include complete raw diff').check()
  await expect(page.getByTestId('raw-diff')).toHaveText(review.diff)
  await expect(page.locator('.print-drawing img')).toHaveCount(diagramCount)
  expect(inspect()).toEqual(before);expect(refs()).toBe(initialRefs)
  expect(requests).toEqual([]);expect(errors).toEqual([])
 })
}

test('real bound print reads safely, prints full content, and rejects stale state without preparing Review',async({page},info)=>{
 const f=fixture('Print production path')
 call('change-set','edit-proposal',...f.state(),'--proposal','# Design first\n\n<script>window.authoredScript=true</script>\n\n![inert](https://example.invalid/image)\n\n'+Array.from({length:25},(_,i)=>`Paragraph ${i+1}: long source remains readable. λ café.\n\n`).join(''))
 call('component','create',...f.state(),'--diagram-id',f.diagram,'--title','Gateway','--description','Full exact source · → \\ λ\r\n')
 call('diagram','add-note',...f.state(),'--diagram-id',f.diagram,'--text','Plain note <b>inert</b> · → λ')
 call('diagram','edit-title',...f.state(),'--diagram-id',f.diagram,'--title','Proposed delivery')
 const review=call('change-set','review',...f.state()).result,before=f.inspect(),requests:{url:string;method:string}[]=[],errors:string[]=[]
 page.on('request',r=>requests.push({url:r.url(),method:r.method()}));page.on('pageerror',e=>errors.push(e.message))
 await page.goto(review.printable_url)
 await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toBeVisible()
 expect(await page.locator('.print-drawing img').count()).toBe(1)
 await expect(page.getByRole('heading',{name:'Print production path · Before',exact:true})).toHaveCount(0)
 await expect(page.getByRole('heading',{name:'Proposed delivery · With changes',exact:true})).toBeVisible()
 expect(await page.evaluate(()=>Boolean((window as any).authoredScript))).toBe(false)
 expect(await page.locator('.print-design script').count()).toBe(0)
 const order=await page.locator('.print-design,.print-diagram,.print-changes,.print-appendix').evaluateAll(elements=>elements.map(e=>e.className))
 expect(order).toEqual(['print-design','print-diagram','print-changes'])
 const pdf=await page.pdf({path:info.outputPath('proposal.pdf'),preferCSSPageSize:true,printBackground:true})
 expect(pdf.subarray(0,5).toString()).toBe('%PDF-')
 await page.getByLabel('Include complete raw diff').check()
 await page.emulateMedia({media:'print'})
 const flow=await page.evaluate(()=>{const diff=document.querySelector('.raw-diff')!,footer=document.querySelector('.print-binding')!;return {diff:diff.getBoundingClientRect().bottom,footer:footer.getBoundingClientRect().top,scroll:diff.scrollHeight,height:diff.clientHeight}})
 expect(flow.footer).toBeGreaterThanOrEqual(flow.diff);expect(flow.scroll).toBeLessThanOrEqual(flow.height+1)
 await page.evaluate(()=>{window.print=()=>{(window as any).printCalls=((window as any).printCalls??0)+1}})
 await page.emulateMedia({media:'screen'});await page.getByRole('button',{name:'Print / Save as PDF'}).click()
 await expect.poll(()=>page.evaluate(()=>(window as any).printCalls)).toBe(1)
 await page.pdf({path:info.outputPath('complete-diff.pdf'),preferCSSPageSize:true,printBackground:true})
 expect(errors).toEqual([]);expect(requests.every(r=>r.method==='GET'&&r.url.startsWith(origin))).toBe(true)
 expect(f.inspect()).toEqual(before)
 call('change-set','edit-proposal',...f.state(),'--proposal','# Later version')
 const later=f.inspect();await page.reload();await expect(page.getByRole('alert')).toContainText('version has moved')
 await expect(page.getByRole('link',{name:'Return to proposal'})).toHaveAttribute('href',`/projects/${f.slug}/proposals/${f.proposal.id}`)
 expect(f.inspect()).toEqual(later);expect(later.review).toBeNull()
})

test('real Markdown-only print shows zero drawings and no empty appendix',async({page},info)=>{
 const f=fixture('Markdown only print')
 call('change-set','edit-proposal',...f.state(),'--proposal','# Text only\n\nNo Architecture changes are proposed.')
 const review=call('change-set','review',...f.state()).result,before=f.inspect()
 await page.goto(review.printable_url)
 await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toBeVisible()
 await expect(page.getByText('No Architecture changes; this proposal contains only proposal text.')).toBeVisible()
 await expect(page.locator('.print-drawing,.raw-diff')).toHaveCount(0)
 await page.pdf({path:info.outputPath('markdown-only.pdf'),preferCSSPageSize:true})
 expect(f.inspect()).toEqual(before)
})

test('actual image export failure is visible and preserves exact text and diff',async({page})=>{
 const f=fixture('Failed image print')
 call('component','create',...f.state(),'--diagram-id',f.diagram,'--title','Gateway')
 call('change-set','edit-proposal',...f.state(),'--proposal','# Design remains available')
 const review=call('change-set','review',...f.state()).result,before=f.inspect()
 await page.addInitScript(()=>{HTMLCanvasElement.prototype.toDataURL=()=>{throw new Error('Image export failed')}})
 await page.goto(review.printable_url)
 await expect(page.getByRole('alert')).toContainText('Diagram could not be rendered')
 await expect(page.getByRole('heading',{name:'Design remains available'})).toBeVisible()
 await page.getByLabel('Include complete raw diff').check()
 await expect(page.getByTestId('raw-diff')).toBeVisible()
 await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toHaveCount(0)
 expect(f.inspect()).toEqual(before)
})
