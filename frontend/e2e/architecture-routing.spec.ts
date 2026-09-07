import {expect,test, type Page} from '@playwright/test'
import {build} from 'vite'
import react from '@vitejs/plugin-react'
import {dirname,resolve} from 'node:path'
import {fileURLToPath} from 'node:url'

const frontend=resolve(dirname(fileURLToPath(import.meta.url)),'..')
let script='',css=''
test.beforeAll(async()=>{
 const result=await build({root:frontend,configFile:false,plugins:[react()],define:{'process.env.NODE_ENV':'"production"'},build:{write:false,lib:{entry:resolve(frontend,'e2e/fixtures/routing-interaction.tsx'),name:'RoutingInteraction',formats:['iife']}}})
 for(const chunk of (Array.isArray(result)?result[0]:result).output){if(chunk.type==='chunk')script+=chunk.code;else if(chunk.fileName.endsWith('.css'))css+=chunk.source.toString()}
})
async function open(page:Page,config:object){
 await page.setViewportSize({width:1440,height:1000});await page.setContent('<div id="root"></div>');await page.addStyleTag({content:css});await page.evaluate(config=>{(window as any).routingCase=config},config);await page.addScriptTag({content:script});await expect(page.getByTestId('architecture-map')).toBeVisible()
}
for(const boundary of [false,true])for(const reverse of [false,true])for(const bend of [-100,100])for(const geometry of ['separated','overlap','touching']){
 test(`directed normal ${boundary?'boundary':'ordinary'} ${reverse?'reverse':'forward'} bend ${bend} ${geometry}`,async({page})=>{
  await open(page,{boundary,reverse,bend})
  const map=page.getByTestId('architecture-map')
  await map.evaluate((el,geometry)=>{const cy=(el as any)._cyreg.cy;const a=cy.getElementById('a'),b=cy.getElementById('b');b.position({x:geometry==='touching'?(a.outerWidth()+b.outerWidth())/2:geometry==='overlap'?200:600,y:0})},geometry)
  const read=()=>map.evaluate(el=>{const cy=(el as any)._cyreg.cy,e=cy.edges().first(),rs=e._private.rscratch,c=e.controlPoints()[0],a=e.source().position(),b=e.target().position(),l=Math.hypot(b.x-a.x,b.y-a.y);return {control:c,expected:{x:(rs.srcIntn[0]+rs.tgtIntn[0])/2-e.data('distance')*(b.y-a.y)/l,y:(rs.srcIntn[1]+rs.tgtIntn[1])/2+e.data('distance')*(b.x-a.x)/l},distance:e.data('distance'),zoom:cy.zoom(),pan:cy.pan()}})
  await expect.poll(async()=>{const r=await read();return Math.hypot(r.control.x-r.expected.x,r.control.y-r.expected.y)}).toBeLessThan(0.00001)
  const before=await read();expect(before.distance).toBe(bend)
  await page.getByRole('button',{name:'Toggle node selection'}).click();expect(await read()).toEqual(before)
  await page.getByRole('button',{name:'Toggle node selection'}).click();expect(await read()).toEqual(before)
  expect(await page.evaluate(()=>(window as any).routingSubmissions)).toEqual([])
 })
}
for(const distance of [0,100])test(`fallback ${distance===0?'coincident':'undefined intersections'} retains scalar and recovers without writes`,async({page})=>{
 await open(page,{bend:125})
 const map=page.getByTestId('architecture-map')
 await map.evaluate((el,x)=>(el as any)._cyreg.cy.getElementById('b').position({x,y:0}),distance)
 await expect(page.getByRole('status')).toContainText('A curve may be unavailable')
 await expect(page.getByRole('button',{name:'Bend selected link'})).toBeHidden()
 expect(await map.evaluate(el=>(el as any)._cyreg.cy.edges().first().data('routing').route.bend)).toBe(125)
 await map.evaluate(el=>(el as any)._cyreg.cy.getElementById('b').position({x:600,y:0}))
 await expect(page.getByRole('status')).toHaveCount(0)
 await expect(page.getByRole('button',{name:'Bend selected link'})).toBeVisible()
 expect(await map.evaluate(el=>(el as any)._cyreg.cy.edges().first().data('distance'))).toBe(125)
 expect(await page.evaluate(()=>(window as any).routingSubmissions)).toEqual([])
})
test('opposing extreme Before With curves share initial and explicit Fit frame',async({page})=>{
 await open(page,{review:true,boundary:true})
 const map=page.getByTestId('architecture-map'),read=()=>map.evaluate(el=>{const cy=(el as any)._cyreg.cy;return {zoom:cy.zoom(),pan:cy.pan()}})
 const before=await read()
 await page.getByRole('button',{name:'Fit map',exact:true}).click();expect(await read()).toEqual(before)
 await page.getByRole('button',{name:'Switch side'}).click();expect(await read()).toEqual(before)
 await page.getByRole('button',{name:'Fit map',exact:true}).click();expect(await read()).toEqual(before)
 const bounds=await map.evaluate(el=>{const cy=(el as any)._cyreg.cy;return {box:cy.elements().renderedBoundingBox(),w:cy.width(),h:cy.height()}})
 expect(bounds.box.x1).toBeGreaterThanOrEqual(0);expect(bounds.box.y1).toBeGreaterThanOrEqual(0);expect(bounds.box.x2).toBeLessThanOrEqual(bounds.w);expect(bounds.box.y2).toBeLessThanOrEqual(bounds.h)
})
