import { expect, test } from '@playwright/test'
import { build } from 'vite'
import react from '@vitejs/plugin-react'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontend = resolve(dirname(fileURLToPath(import.meta.url)), '..')
let script = '', css = ''
test.beforeAll(async () => {
  const result = await build({
    root: frontend, configFile: false, plugins: [react()],
    define: {'process.env.NODE_ENV': '"production"'},
    build: {write:false, lib:{entry:resolve(frontend,'e2e/fixtures/sizing-interaction.tsx'),name:'SizingInteraction',formats:['iife']}},
  })
  for (const chunk of (Array.isArray(result) ? result[0] : result).output) {
    if (chunk.type === 'chunk') script += chunk.code
    else if (chunk.fileName.endsWith('.css')) css += chunk.source.toString()
  }
})

for (const action of ['Select B', 'Clear selection', 'Leave editing', 'Rerender callback']) {
  test(`active resize handles ${action} without confusing callback identity with context`, async ({page}) => {
    await page.setViewportSize({width:1440,height:1000})
    await page.setContent('<div id="root"></div>')
    await page.addStyleTag({content:css})
    await page.addScriptTag({content:script})
    const map=page.getByTestId('architecture-map')
    const snapshot=()=>map.evaluate(el=>{
      const cy=(el as any)._cyreg.cy
      return {zoom:cy.zoom(),pan:{...cy.pan()},nodes:cy.nodes().map((n:any)=>({id:n.id(),p:{...n.position()},width:n.data('width'),height:n.data('height')}))}
    })
    const handle=page.getByRole('button',{name:'Resize selected node',exact:true})
    await expect(handle).toBeVisible()
    const before=await snapshot(),box=(await handle.boundingBox())!
    await page.mouse.move(box.x+14,box.y+14)
    await page.mouse.down()
    await page.mouse.move(box.x+44,box.y+34,{steps:5})
    expect((await snapshot()).nodes[0].width).not.toBe(200)
    await page.getByRole('button',{name:action,exact:true}).focus()
    await page.keyboard.press('Enter')
    await page.mouse.up()
    const submissions=await page.evaluate(()=>(window as any).submissions)
    if (action==='Rerender callback') {
      expect(submissions).toHaveLength(1)
      expect(submissions[0].id).toBe('a')
    } else {
      expect(submissions).toEqual([])
      expect(await snapshot()).toEqual(before)
    }
  })
}

test('resize handle is clipped to the canvas and remains usable after explicit Fit', async ({page}) => {
  await page.setViewportSize({width:1440,height:1000})
  await page.setContent('<div id="root"></div>')
  await page.addStyleTag({content:css})
  await page.addScriptTag({content:script})
  const map=page.getByTestId('architecture-map')
  const handle=page.getByRole('button',{name:'Resize selected node',exact:true})
  await expect(handle).toBeVisible()
  await map.evaluate(el=>(el as any)._cyreg.cy.viewport({zoom:1,pan:{x:1000,y:200}}))
  const outside=(await handle.boundingBox())!
  expect(await page.evaluate(p=>Boolean(document.elementFromPoint(p.x,p.y)?.closest('.map-resize-handle')),{x:outside.x+14,y:outside.y+14})).toBe(false)
  await page.getByRole('button',{name:'Fit map',exact:true}).click()
  const fitted=(await handle.boundingBox())!
  expect(await page.evaluate(p=>Boolean(document.elementFromPoint(p.x,p.y)?.closest('.map-resize-handle')),{x:fitted.x+14,y:fitted.y+14})).toBe(true)
})
