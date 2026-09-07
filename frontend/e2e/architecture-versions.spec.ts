import {expect,test} from '@playwright/test'
import {spawn,execFileSync,type ChildProcess} from 'node:child_process'
import {mkdtempSync} from 'node:fs'
import {createServer} from 'node:net'
import {tmpdir} from 'node:os'
import {dirname,join,resolve} from 'node:path'
import {fileURLToPath} from 'node:url'

const frontend=resolve(dirname(fileURLToPath(import.meta.url)),'..'),root=resolve(frontend,'..')
let directory:string,binary:string,origin:string,server:ChildProcess
test.beforeAll(async()=>{
 directory=mkdtempSync(join(tmpdir(),'workbraid-versions-browser-'));binary=join(directory,'workbraid')
 execFileSync('go',['build','-o',binary,'./cmd/workbraid'],{cwd:root})
 const port=await new Promise<number>(resolve=>{const s=createServer();s.listen(0,'127.0.0.1',()=>{const a=s.address();if(!a||typeof a==='string')throw new Error('No port');s.close(()=>resolve(a.port))})})
 origin=`http://127.0.0.1:${port}`
 server=spawn(binary,['--listen',`127.0.0.1:${port}`,'--data-dir',join(directory,'data'),'--ui-dir',join(frontend,'dist')],{stdio:['ignore','pipe','pipe']})
 await expect.poll(async()=>{try{return (await fetch(origin+'/api/agent/v2/status')).ok}catch{return false}}).toBe(true)
})
test.afterAll(async()=>{if(server){const exit=new Promise(resolve=>server.once('exit',resolve));server.kill('SIGTERM');await exit}})
function call(...args:string[]){const result=JSON.parse(execFileSync(binary,['--server',origin,'--json',...args],{encoding:'utf8'}));expect(result.ok).toBe(true);return result}

test('retained versions: real selector, report, narrow and PDF',async({page},info)=>{
 const project=call('project','create','--name','Delivery architecture'),store=project.context.project.store_id,slug=project.context.project.slug
 const accepted=call('architecture','inspect').result
 let proposal=call('change-set','create','--store-id',store,'--accepted-revision',accepted.revision,'--name','Document delivery').result
 const inspect=()=>call('change-set','inspect','--store-id',store,'--change-set-id',proposal.id).result
 const state=()=>['--store-id',store,'--change-set-id',proposal.id,'--generation',String(inspect().generation)]
 call('component','create',...state(),'--diagram-id',accepted.root_diagram_id,'--title','Request gateway','--description','Receives delivery requests.\n')
 call('component','create',...state(),'--diagram-id',accepted.root_diagram_id,'--title','Delivery service','--description','Coordinates delivery.\n')
 let components=inspect().candidate.components
 const gateway=components.find((c:any)=>c.title==='Request gateway').id,delivery=components.find((c:any)=>c.title==='Delivery service').id
 call('relationship','add',...state(),'--source-id',gateway,'--target-id',delivery,'--label','requests delivery')
 const binding=call('change-set','review',...state()).result
 call('architecture','update','--store-id',store,'--change-set-id',proposal.id,'--base-revision',binding.base_revision,'--candidate-tree',binding.candidate_tree,'--generation',String(binding.generation))
 const baseline=call('architecture','inspect').result.revision
 proposal=call('change-set','create','--store-id',store,'--accepted-revision',baseline,'--name','Clarify request validation').result
 call('change-set','edit-proposal',...state(),'--proposal','# Clarify request validation\n\nThe gateway validates requests before passing them to the delivery service. Keep delivery coordination with the service.\n')
 call('component','edit',...state(),'--component-id',gateway,'--description','Validates delivery requests and records a tracking reference.\n')
 const chosen=inspect(),refs=()=>execFileSync('git',['--git-dir',join(directory,'data','architecture',store+'.git'),'for-each-ref','--format=%(refname) %(objectname)'],{encoding:'utf8'}),initialRefs=refs()
 // A different process-current project must survive selector/report reads.
 call('project','create','--name','Other active work')
 const current=call('status').context,errors:string[]=[]
 page.on('pageerror',e=>errors.push(e.message))
 await page.setViewportSize({width:1440,height:1000})
 await page.goto(origin+`/projects/${slug}/compare`)
 await expect(page.getByRole('heading',{name:'Compare versions',exact:true})).toBeVisible()
 await page.getByLabel('From',{exact:true}).nth(1).selectOption('proposal')
 await expect(page.locator('#After-version option')).toHaveCount(3)
 await page.locator('#Before-version').selectOption({index:1})
 await page.locator('#After-version').selectOption({index:2})
 const selectedPair=new URL(page.url()).search
 await page.getByRole('button',{name:'Swap versions'}).click()
 await page.getByRole('button',{name:'Swap versions'}).click()
 expect(new URL(page.url()).search).toBe(selectedPair)
 await page.screenshot({path:info.outputPath('selector-wide.png'),fullPage:true})
 await page.getByRole('button',{name:'Report',exact:true}).click()
 await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toBeVisible()
 const exactReportURL=page.url()
 await expect(page.locator('.print-drawing img')).toHaveCount(1)
 await expect(page.getByRole('heading',{name:'After proposal document'})).toBeVisible()
 await page.screenshot({path:info.outputPath('report-wide.png'),fullPage:true})
 await page.pdf({path:info.outputPath('report.pdf'),preferCSSPageSize:true,printBackground:true})
 await page.setViewportSize({width:390,height:844})
 await page.screenshot({path:info.outputPath('report-narrow.png'),fullPage:true})
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true)
 await page.getByRole('link',{name:'Compare versions',exact:true}).click()
 await expect(page.locator('#After-version')).toBeEnabled()
 await page.screenshot({path:info.outputPath('selector-narrow.png'),fullPage:true})
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true)
 expect(refs()).toBe(initialRefs);expect(call('status').context).toEqual(current);expect(errors).toEqual([])
 // Exact unprepared proposal remains unprepared.
 call('project','open','--slug',slug)
 expect(inspect()).toEqual(chosen)
 // Normal-workspace navigation preserves the existing unsent editor guard.
 await page.setViewportSize({width:1440,height:1000})
 await page.goto(origin+`/projects/${slug}/proposals/${proposal.id}`)
 await page.getByRole('textbox',{name:'Proposal',exact:true}).fill('Unsent browser proposal text')
 await page.getByRole('button',{name:'Compare versions',exact:true}).click()
 await expect(page.getByRole('dialog',{name:'Leave without keeping?'})).toBeVisible()
 await page.getByRole('button',{name:'Keep editing',exact:true}).click()
 await expect(page.getByRole('textbox',{name:'Proposal',exact:true})).toHaveValue('Unsent browser proposal text')
 await page.getByRole('button',{name:'Compare versions',exact:true}).click()
 await page.getByRole('button',{name:'Leave without keeping',exact:true}).click()
 await expect(page.getByRole('heading',{name:'Compare versions',exact:true})).toBeVisible()
 expect(inspect()).toEqual(chosen);expect(refs()).toBe(initialRefs)
 // A process restart reconstructs this exact pair; no report captures it.
 const exit=new Promise(resolve=>server.once('exit',resolve));server.kill('SIGTERM');await exit
 server=spawn(binary,['--listen',new URL(origin).host,'--data-dir',join(directory,'data'),'--ui-dir',join(frontend,'dist')],{stdio:['ignore','pipe','pipe']})
 await expect.poll(async()=>{try{return(await fetch(origin+'/api/agent/v2/status')).ok}catch{return false}}).toBe(true)
 await page.goto(exactReportURL)
 await expect(page.getByRole('button',{name:'Print / Save as PDF'})).toBeVisible()
 expect(refs()).toBe(initialRefs)
 call('project','open','--slug',slug)
 call('change-set','edit-proposal',...state(),'--proposal','Later proposal context')
 const movedRefs=refs()
 await page.reload()
 await expect(page.getByRole('alert')).toContainText('selected version has changed')
 await page.setViewportSize({width:390,height:844})
 await page.screenshot({path:info.outputPath('stale-report-narrow.png'),fullPage:true})
 const q=new URLSearchParams({store_id:store,before:JSON.stringify({kind:'accepted',revision:baseline}),after:JSON.stringify({kind:'accepted',revision:baseline})})
 await page.goto(origin+`/projects/${slug}/compare/report?${q}`)
 await expect(page.getByText('No Architecture differences between these versions.')).toBeVisible()
 await expect(page.locator('.print-design')).toHaveCount(0)
 await expect(page.locator('.print-drawing')).toHaveCount(0)
 expect(refs()).toBe(movedRefs);expect(errors).toEqual([])
})
