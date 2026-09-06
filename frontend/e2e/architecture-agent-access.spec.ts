import { expect, test, type Page } from '@playwright/test'
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { closeSync, mkdirSync, mkdtempSync, openSync, rmSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repositoryRoot = resolve(frontendRoot, '..')

type RunningWorkBraid = { child: ChildProcess; origin: string; logFD: number }
type AgentEnvelope = {
  protocol: string
  ok: boolean
  context: { project?: { store_id: string; slug: string }; accepted_revision?: string }
  result?: Record<string, any>
  error?: { code: string }
}

test('durable placement uses real drag, model centers, one generation, reset, review frame and restart',async({page},testInfo)=>{
  page.setDefaultTimeout(15000)
  const runtimeRoot=mkdtempSync(join(tmpdir(),'workbraid-phase31-browser-'))
  const binary=join(runtimeRoot,'workbraid'),dataRoot=join(runtimeRoot,'data')
  let application:RunningWorkBraid|undefined
  try {
    mkdirSync(dataRoot,{recursive:true})
    run('go',['build','-o',binary,'./cmd/workbraid'],repositoryRoot)
    const port=await unusedLoopbackPort();application=await startWorkBraid(binary,dataRoot,port,runtimeRoot)
    const origin=application.origin
    const call=(args:string[])=>agent(binary,origin,args)
    const project=call(['project','create','--name','Placement browser'])
    const store=project.context.project!.store_id,slug=project.context.project!.slug
    const root=call(['architecture','inspect']).result!.root_diagram_id as string
    const seed=call(['change-set','create','--store-id',store,'--accepted-revision',project.context.accepted_revision!,'--name','Seed architecture']).result!
    const inspect=(id:string)=>call(['change-set','inspect','--store-id',store,'--change-set-id',id]).result!
    const state=(id:string)=>['--store-id',store,'--change-set-id',id,'--generation',String(inspect(id).generation)]
    const create=(title:string)=>call(['component','create',...state(seed.id),'--diagram-id',root,'--title',title,'--description',`${title} has a precise responsibility.\n\nThis documentation remains readable while arranging the Diagram.\n`]).result!.component_id as string
    const gateway=create('Gateway'),worker=create('Worker'),records=create('Records'),ops=create('Operations'),external=create('External delivery')
    const detail=call(['diagram','create-detail',...state(seed.id),'--component-id',gateway,'--title','Runtime']).result!.diagram_id as string
    for(const id of [worker,external])call(['component','move-home',...state(seed.id),'--component-id',id,'--diagram-id',detail])
    call(['diagram','show-component',...state(seed.id),'--component-id',worker,'--diagram-id',root])
    for(const [source,target]of [[gateway,worker],[worker,records],[gateway,external]])call(['relationship','add',...state(seed.id),'--source-id',source,'--target-id',target,'--label','calls'])
    const accept=(id:string)=>{const review=call(['change-set','review',...state(id)]).result!;return call(['architecture','update','--store-id',store,'--change-set-id',id,'--generation',String(review.generation),'--base-revision',review.base_revision,'--candidate-tree',review.candidate_tree])}
    const accepted=accept(seed.id).context.accepted_revision!
    await page.goto(`${origin}/projects/${slug}`)
    const map=page.getByTestId('architecture-map');await expect(map).toBeVisible()
    const node=async(id:string)=>map.evaluate((el,id)=>{const cy=(el as any)._cyreg.cy,n=cy.getElementById(id),r=el.getBoundingClientRect();return {x:r.left+n.renderedPosition().x,y:r.top+n.renderedPosition().y,model:n.position(),zoom:cy.zoom(),pan:cy.pan(),grab:n.grabbable()}},id)
    const frame=()=>map.evaluate(el=>{const cy=(el as any)._cyreg.cy;return {zoom:cy.zoom(),pan:cy.pan()}})
    const beforeNode=await node(worker);expect(beforeNode.grab).toBe(true)
    await page.mouse.move(beforeNode.x,beforeNode.y);await page.mouse.wheel(0,-140)
    await page.waitForTimeout(250)
    const box=(await map.boundingBox())!
    await page.mouse.move(box.x+25,box.y+25);await page.mouse.down();await page.mouse.move(box.x+65,box.y+60,{steps:6});await page.mouse.up()
    const before=await frame(),start=await node(worker)
    let drops=0;page.on('request',request=>{if(request.url().endsWith('/diagrams/set-position'))drops++})
    await page.mouse.move(start.x,start.y);await page.mouse.down();await page.mouse.move(start.x+90,start.y-65,{steps:12})
    expect(drops).toBe(0)
    expect(call(['change-set','list','--store-id',store]).result!.change_sets.filter((c:any)=>c.lifecycle==='active')).toHaveLength(0)
    const drop=page.waitForResponse(r=>r.url().endsWith('/diagrams/set-position'))
    await page.mouse.up();expect((await drop).ok()).toBe(true)
    await expect(page).toHaveURL(/\/proposals\//)
    const id=page.url().split('/').at(-1)!
    const proposed=inspect(id),appearance=proposed.candidate.diagrams.find((d:any)=>d.id===root).appearances.find((a:any)=>a.component_id===worker)
    const round=(v:number)=>Math.sign(v)*Math.floor(Math.abs(v)+.5)
    expect(appearance.position).toEqual({x:round(start.model.x+90/start.zoom),y:round(start.model.y-65/start.zoom)})
    expect(proposed.generation).toBe(1);expect(proposed.node_positions).toHaveLength(1);expect(drops).toBe(1)
    expect(call(['architecture','inspect']).context.accepted_revision).toBe(accepted)
    expect(await frame()).toEqual(before)
    const after=await node(worker);await page.mouse.click(after.x,after.y);expect(drops).toBe(1)
    await page.locator('.position-controls summary').click()
    await page.getByLabel('Position X',{exact:true}).fill('-320');await page.getByLabel('Position Y',{exact:true}).fill('180')
    await page.getByRole('button',{name:'Keep position',exact:true}).click()
    await expect.poll(()=>inspect(id).generation).toBe(2)
    await page.getByRole('button',{name:'Back to proposal',exact:true}).click()
    await page.getByRole('button',{name:'Review changes',exact:true}).click()
    await expect(page.getByText(/Position changed: Worker/)).toBeVisible()
    const reviewFrame=await frame()
    await page.getByRole('button',{name:'Before changes',exact:true}).click();expect(await frame()).toEqual(reviewFrame)
    expect((await node(worker)).grab).toBe(false)
    await page.getByRole('button',{name:'With changes',exact:true}).click();expect(await frame()).toEqual(reviewFrame)
    await expect(page.getByTestId('raw-diff')).toContainText('positions:')
    await page.screenshot({path:join(runtimeRoot,'review.png'),fullPage:true})
    await page.getByRole('button',{name:'Back to proposal',exact:true}).click()
    const cancelStart=await node(worker)
    await page.mouse.move(cancelStart.x,cancelStart.y);await page.mouse.down();await page.mouse.move(cancelStart.x-40,cancelStart.y-30,{steps:5});await page.keyboard.press('Escape');await page.mouse.up()
    expect(inspect(id).generation).toBe(2)
    await page.getByRole('button',{name:'Reset layout',exact:true}).click()
    await expect.poll(()=>inspect(id).generation).toBe(3)
    expect(inspect(id).candidate.diagrams.find((d:any)=>d.id===root).appearances.every((a:any)=>a.position===null)).toBe(true)
    const resetNode=await node(worker);await page.mouse.click(resetNode.x,resetNode.y)
    await page.locator('.position-controls summary').click()
    await page.getByLabel('Position X',{exact:true}).fill('-410');await page.getByLabel('Position Y',{exact:true}).fill('220')
    await page.getByRole('button',{name:'Keep position',exact:true}).click()
    await expect.poll(()=>inspect(id).generation).toBe(4)
    await page.getByRole('button',{name:'Back to proposal',exact:true}).click()
    await page.getByRole('button',{name:'Review changes',exact:true}).click()
    await page.getByRole('button',{name:'Update architecture',exact:true}).click()
    await expect(page.getByRole('button',{name:'Showing Accepted',exact:true})).toBeVisible()
    const final=call(['architecture','inspect'])
    await page.screenshot({path:join(runtimeRoot,'accepted.png'),fullPage:true})
    await stopWorkBraid(application);application=undefined
    expect(spawnSync(binary,['--server',origin,'--json','status'],{encoding:'utf8'}).status).not.toBe(0)
    application=await startWorkBraid(binary,dataRoot,port,runtimeRoot,'restart.log')
    await page.reload();await expect(map).toBeVisible()
    expect(call(['architecture','inspect'])).toEqual(final)
    expect((await node(worker)).model).toEqual({x:-410,y:220})
    writeFileSync(join(runtimeRoot,'evidence.json'),JSON.stringify({store,root,detail,gateway,worker,records,ops,external,accepted,proposal:id,before,start,proposed,final,drops},null,2))
    await testInfo.attach('placement-evidence',{path:join(runtimeRoot,'evidence.json'),contentType:'application/json'})
  } finally {if(application)await stopWorkBraid(application);writeFileSync(join(runtimeRoot,'runtime-stopped.txt'),'Task server stopped by runner teardown.\n');console.log(`Placement evidence: ${runtimeRoot}`)}
})

test('built browser and Agent v2 preserve independent active/applied proposals across restart', async ({ page }) => {
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-change-sets-'))
  const dataRoot = join(runtimeRoot, 'app-data')
  const binary = join(runtimeRoot, 'workbraid')
  let application: RunningWorkBraid | undefined

  try {
    mkdirSync(dataRoot, { recursive: true })
    run('go', ['build', '-buildvcs=false', '-o', binary, './cmd/workbraid'], repositoryRoot)
    const port = await unusedLoopbackPort()
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot)

    await page.goto(application.origin)
    await page.getByLabel('New project').fill('Change set evidence')
    await page.getByRole('button', { name: 'Create project' }).click()
    await expect(page).toHaveURL(`${application.origin}/projects/change-set-evidence`)

    const acceptedR0 = agent(binary, application.origin, ['architecture', 'inspect'])
    const storeID = acceptedR0.context.project!.store_id
    const revisionR0 = acceptedR0.context.accepted_revision!

    await page.getByRole('button', { name: 'New changes' }).click()
    const canceledTask = await visibleNewChangesTask(page)
    await canceledTask.getByLabel('Name').fill('Not created')
    await canceledTask.getByRole('button', { name: 'Cancel' }).click()
    await expect(canceledTask).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Showing Accepted' })).toBeVisible()

    await createBrowserChangeSet(page, 'Change A')
    await saveBrowserProposal(page, '# Change A\n\nRoute requests through a durable gateway.\n\n- Preserve retry state\n\n| Decision | Owner |\n| --- | --- |\n| Durable routing | Gateway |\n')
    await addBrowserComponent(page, 'Gateway', 'Routes requests.\n')
    await selectShowing(page, 'Accepted')
    await expect(page.getByRole('dialog', { name: 'Leave without keeping?' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Showing Accepted' })).toBeVisible()

    await createBrowserChangeSet(page, 'Change B')
    await saveBrowserProposal(page, '# Change B\n\nAdd an independent worker.\n')
    await addBrowserComponent(page, 'Worker', 'Processes work.\n')

    const listed = agent(binary, application.origin, ['change-set', 'list', '--store-id', storeID])
    const records = listed.result!.change_sets as Array<Record<string, any>>
    const changeA = records.find((record) => record.name === 'Change A')!
    const changeB = records.find((record) => record.name === 'Change B')!
    expect(changeA.id).not.toBe(changeB.id)
    expect(changeA.base_revision).toBe(revisionR0)
    expect(changeB.base_revision).toBe(revisionR0)

    const workerID = (changeB.components as Array<Record<string, any>>)[0].id as string
    const invalidB = agent(binary, application.origin, [
      'relationship', 'add', '--store-id', storeID, '--change-set-id', changeB.id, '--generation', String(changeB.generation),
      '--source-id', workerID, '--target-id', 'not-a-uuid', '--label', '   ',
    ])
    expect(invalidB.result!.candidate_valid).toBe(false)

    await page.getByRole('button', { name: 'Refresh' }).click()
    await expect(page.getByRole('heading', { name: 'Needs correction' })).toBeVisible()
    await expect(page.getByTestId('architecture-map')).toHaveCount(0)

    await openShowing(page)
    await expect(page.getByRole('group', { name: 'Open proposals' }).getByRole('option', { name: 'Change A' })).toBeVisible()
    await expect(page.getByRole('group', { name: 'Open proposals' }).getByRole('option', { name: 'Change B' })).toBeVisible()
    await page.getByRole('option', { name: 'Change A' }).click()
    await expect(page.getByRole('heading', { name: 'Change A', level: 2 })).toBeVisible()
    await expect(page.getByText('These changes have not updated Architecture yet.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'With changes' })).toHaveCount(0)
    await page.getByRole('button', { name: 'Review changes' }).click()
    const changeAReviewURL = `${application.origin}/projects/change-set-evidence/proposals/${changeA.id}/review`
    await expect(page).toHaveURL(changeAReviewURL)
    const reviewProposal = page.getByRole('group', { name: 'Proposal' })
    await expect(reviewProposal).toHaveAttribute('open', '')
    await expect(reviewProposal.getByText('Route requests through a durable gateway.')).toBeVisible()
    await expect(page.getByTestId('raw-diff')).toContainText('Gateway')
    await page.getByRole('button', { name: 'Update architecture' }).click()
    await expect(page).toHaveURL(`${application.origin}/projects/change-set-evidence`)
    await expect(page.getByRole('button', { name: 'Showing Accepted', exact: true })).toBeVisible()
    await expect(page.locator('.component-documentation').getByRole('heading', { name: 'Gateway', exact: true })).toBeVisible()
    await expect(page.locator('.component-documentation')).toContainText('Routes requests.')
    await expect(page.getByRole('group', { name: 'Review side', exact: true })).toHaveCount(0)
    const revisionR1 = agent(binary, application.origin, ['architecture', 'inspect']).context.accepted_revision!
    expect(revisionR1).not.toBe(revisionR0)
    expect(await displayedRevision(page)).toBe(revisionR1)
    await selectShowing(page, 'Change A')
    await expect(page).toHaveURL(`${application.origin}/projects/change-set-evidence/proposals/${changeA.id}`)
    await expect(page.getByRole('heading', { name: 'Change A', level: 2 })).toBeVisible()
    await expect(page.getByText('Accepted proposal')).toBeVisible()
    await expect(page.getByText('This is the proposal that updated Architecture. It cannot be changed.')).toBeVisible()
    const acceptedMarkdown = page.locator('.proposal-document .markdown-body')
    await expect(acceptedMarkdown.locator('li')).toContainText('Preserve retry state')
    await expect(acceptedMarkdown.locator('table')).toBeVisible()
    expect(await acceptedMarkdown.locator('li').evaluate((item) => getComputedStyle(item).display)).toBe('list-item')
    expect(await acceptedMarkdown.locator('table').evaluate((table) => getComputedStyle(table).display)).toBe('table')
    await expect(acceptedMarkdown.locator('table').locator('xpath=..')).toHaveClass(/markdown-table-scroll/)
    await expect(page.getByLabel('Name')).toHaveCount(0)
    await expect(page.getByText(/Out of date with Accepted/)).toHaveCount(0)

    await selectShowing(page, 'Change B · Out of date')
    await expect(page.getByText('Out of date with Accepted. You can still edit and review this proposal, but it cannot update Architecture until it matches Accepted.')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Needs correction' })).toBeVisible()
    const inspectedInvalidB = agent(binary, application.origin, ['change-set', 'inspect', '--store-id', storeID, '--change-set-id', changeB.id])
    const repairedB = agent(binary, application.origin, [
      'relationship', 'edit', '--store-id', storeID, '--change-set-id', changeB.id, '--generation', String(inspectedInvalidB.result!.generation),
      '--source-id', workerID, '--old-target-id', 'not-a-uuid', '--old-label', '   ', '--target-id', workerID, '--label', 'retries',
    ])
    expect(repairedB.result!.candidate_valid).toBe(true)
    const reviewedB = agent(binary, application.origin, [
      'change-set', 'review', '--store-id', storeID, '--change-set-id', changeB.id, '--generation', String(repairedB.result!.generation),
    ])
    expect(reviewedB.result!.review_url).toBe(`${application.origin}/projects/change-set-evidence/proposals/${changeB.id}/review`)
    const rejectedB = agentFailure(binary, application.origin, [
      'architecture', 'update', '--store-id', storeID, '--change-set-id', changeB.id,
      '--base-revision', reviewedB.result!.base_revision, '--candidate-tree', reviewedB.result!.candidate_tree,
      '--generation', String(reviewedB.result!.generation),
    ])
    expect(rejectedB.error?.code).toBe('change_set_out_of_date')
    expect(rejectedB.context.accepted_revision).toBe(revisionR1)

    await page.getByRole('button', { name: 'Refresh' }).click()
    await expect(page.getByRole('heading', { name: 'Change B', level: 2 })).toBeVisible()
    await page.getByRole('button', { name: 'Return to review' }).click()
    const changeBReviewURL = `${application.origin}/projects/change-set-evidence/proposals/${changeB.id}/review`
    await expect(page).toHaveURL(changeBReviewURL)
    await expect(page.getByRole('button', { name: 'With changes' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Showing Change B · Out of date' })).toBeVisible()
    await expect(page.getByText('Out of date with Accepted. You can inspect this review, but it cannot update Architecture until the proposal matches Accepted.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Update architecture' })).toHaveCount(0)
    await expect(page.getByText('Processes work.', { exact: true })).toBeVisible()
    await expect(page.getByRole('group', { name: 'Proposal' }).getByText('Add an independent worker.', { exact: true })).toBeVisible()

    await page.getByLabel('Reviewer name').fill('Browser reviewer')
    await page.getByRole('combobox', { name: /^Conclusion/ }).selectOption('request_changes')
    await page.getByRole('textbox', { name: /^Review summary/ }).fill('Please make the worker responsibility more explicit.\n')
    await page.getByRole('region', { name: 'Review context' }).getByRole('button', { name: 'Comment on lines' }).click()
    const commentEditor = page.getByRole('region', { name: 'Comment on Worker lines' })
    const sourceLine = commentEditor.getByRole('option').filter({ hasText: 'Processes work.' })
    await sourceLine.click()
    await expect(sourceLine).toHaveAttribute('aria-selected', 'true')
    await commentEditor.getByLabel('Comment', { exact: true }).fill('Clarify this sentence.\n')
    await commentEditor.getByRole('button', { name: 'Add comment' }).click()
    await page.getByRole('button', { name: 'Submit review' }).click()
    await expect(page.getByRole('heading', { name: 'Changes requested', level: 2 })).toBeVisible()
    await expect(page.getByText('Please make the worker responsibility more explicit.')).toBeVisible()
    await expect(page.getByText('Clarify this sentence.')).toBeVisible()
    await expect(page.locator('.review-anchor-excerpt')).toContainText('Processes work.')
    const submittedReviewURL = page.url()
    expect(submittedReviewURL).toMatch(new RegExp(`${application.origin}/projects/change-set-evidence/proposals/${changeB.id}/reviews/[0-9a-f-]+$`))

    await page.goBack()
    await expect(page).toHaveURL(changeBReviewURL)
    await expect(page.getByRole('heading', { name: 'Submit feedback', level: 3 })).toBeVisible()

    await page.goBack()
    await expect(page).toHaveURL(`${application.origin}/projects/change-set-evidence/proposals/${changeB.id}`)
    await expect(page.getByRole('heading', { name: 'Change B', level: 2 })).toBeVisible()
    await expect(page.getByRole('button', { name: 'With changes' })).toHaveCount(0)
    await page.goForward()
    await expect(page).toHaveURL(changeBReviewURL)
    await expect(page.getByRole('heading', { name: 'Review changes', level: 2 })).toBeVisible()

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'restart.log')
    await page.goto(submittedReviewURL)
    await expect(page.getByRole('heading', { name: 'Changes requested', level: 2 })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Submitted review feedback' }).getByText('Browser reviewer')).toBeVisible()
    await expect(page.getByText('Clarify this sentence.')).toBeVisible()
    await expect(page.locator('.review-anchor-excerpt')).toContainText('Processes work.')
    await page.goto(changeAReviewURL)
    await expect(page.getByRole('heading', { name: 'Review changes', level: 2 })).toBeVisible()
    await expect(page.getByText('Accepted proposal')).toBeVisible()
    await expect(page.getByText('Review the proposed architecture and its complete file changes.')).toBeVisible()
    await expect(page.getByRole('group', { name: 'Proposal' }).getByText('Route requests through a durable gateway.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Update architecture' })).toHaveCount(0)
    await page.getByRole('button', { name: 'Back to proposal' }).click()
    await expect(page).toHaveURL(`${application.origin}/projects/change-set-evidence/proposals/${changeA.id}`)
    await selectShowing(page, 'Accepted')
    await expect(page).toHaveURL(`${application.origin}/projects/change-set-evidence`)
    expect(await displayedRevision(page)).toBe(revisionR1)
    await selectShowing(page, 'Change B · Out of date')
    await expect(page.getByText(/Out of date with Accepted/)).toBeVisible()
    await expect(page.getByText('Add an independent worker.', { exact: true })).toBeVisible()

    const afterRestart = agent(binary, application.origin, ['change-set', 'list', '--store-id', storeID])
    const restartedRecords = afterRestart.result!.change_sets as Array<Record<string, any>>
    expect(restartedRecords.find((record) => record.id === changeA.id)?.lifecycle).toBe('applied')
    expect(restartedRecords.find((record) => record.id === changeB.id)?.out_of_date).toBe(true)
  } finally {
    if (application) await stopWorkBraid(application)
    rmSync(runtimeRoot, { recursive: true, force: true })
  }
})

async function createBrowserChangeSet(page: Page, name: string) {
  await page.getByRole('button', { name: 'New changes' }).click()
  const form = await visibleNewChangesTask(page)
  await form.getByLabel('Name').fill(name)
  await form.getByRole('button', { name: 'Create' }).click()
  await expect(page.getByRole('heading', { name, level: 2 })).toBeVisible()
  await expect(page.getByText('Open proposal')).toBeVisible()
}

async function visibleNewChangesTask(page: Page) {
  const pane = page.getByRole('complementary', { name: 'Architecture task' })
  const form = pane.locator('form')
  await expect(form).toBeVisible()
  await expect(form.getByRole('heading', { name: 'New changes', level: 2 })).toBeVisible()
  return form
}

async function openShowing(page: Page) {
  const trigger = page.getByRole('button', { name: /^Showing / })
  await trigger.click()
  await expect(trigger).toHaveAttribute('aria-expanded', 'true')
  await expect(page.getByRole('listbox', { name: 'Showing' })).toBeVisible()
}

async function selectShowing(page: Page, name: string) {
  const trigger = page.getByRole('button', { name: /^Showing / })
  await openShowing(page)
  await page.getByRole('option', { name, exact: true }).click()
  await expect(trigger).toHaveAttribute('aria-expanded', 'false')
}

async function saveBrowserProposal(page: Page, source: string) {
  await page.getByRole('textbox', { name: 'Proposal' }).fill(source)
  const save = page.getByRole('button', { name: 'Save proposal' })
  await save.click()
  await expect(save).toHaveCount(0)
}

async function addBrowserComponent(page: Page, title: string, description: string) {
  await page.locator('.pending-diagram-row').first().getByRole('button', { name: 'Add component' }).click()
  const pane = page.getByRole('complementary', { name: 'Architecture task' })
  const editor = pane.locator('form')
  await expect(editor.getByRole('heading', { name: 'Add component', level: 2 })).toBeVisible()
  await editor.getByLabel('Title').fill(title)
  await editor.getByLabel('Description').fill(description)
  await editor.getByRole('button', { name: 'Keep change' }).click()
  await expect(pane.getByRole('heading', { name: 'Add component', level: 2 })).toHaveCount(0)
  await expect(pane.getByRole('heading', { name: 'Architecture work in this proposal', level: 3 })).toBeVisible()
}

function agent(binary: string, origin: string, arguments_: string[]): AgentEnvelope {
  const envelope = runAgent(binary, origin, arguments_)
  expect(envelope.protocol).toBe('workbraid-agent-v2')
  expect(envelope.ok, `${arguments_.join(' ')} failed: ${JSON.stringify(envelope.error)}`).toBe(true)
  return envelope
}

function agentFailure(binary: string, origin: string, arguments_: string[]): AgentEnvelope {
  const result = spawnSync(binary, ['--server', origin, '--json', ...arguments_], { cwd: repositoryRoot, encoding: 'utf8' })
  expect(result.status).toBe(1)
  const envelope = JSON.parse(result.stdout) as AgentEnvelope
  expect(envelope.protocol).toBe('workbraid-agent-v2')
  expect(envelope.ok).toBe(false)
  return envelope
}

function runAgent(binary: string, origin: string, arguments_: string[]) {
  return JSON.parse(run(binary, ['--server', origin, '--json', ...arguments_], repositoryRoot)) as AgentEnvelope
}

async function displayedRevision(page: Page) {
  const pane = page.getByRole('complementary', { name: 'Architecture task' })
  const details = pane.locator('details.technical-details').filter({ hasText: 'Project slug' })
  await expect(details).toBeVisible({ timeout: 5_000 })
  if (!(await details.getAttribute('open'))) await details.locator('summary').click()
  const revisionLabel = details.getByText('Revision', { exact: true })
  await expect(revisionLabel).toBeVisible({ timeout: 5_000 })
  return revisionLabel.evaluate((label) => label.nextElementSibling?.textContent?.trim())
}

function run(command: string, arguments_: string[], cwd: string) {
  const result = spawnSync(command, arguments_, { cwd, encoding: 'utf8', env: { ...process.env, GIT_CONFIG_NOSYSTEM: '1', GIT_TERMINAL_PROMPT: '0' } })
  if (result.status !== 0) throw new Error(`${command} ${arguments_.join(' ')} failed (${result.status}):\n${result.stdout}\n${result.stderr}`)
  return result.stdout.trim()
}

async function unusedLoopbackPort() {
  const server = createServer()
  await new Promise<void>((resolvePromise, reject) => { server.once('error', reject); server.listen(0, '127.0.0.1', resolvePromise) })
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('could not allocate a loopback port')
  await new Promise<void>((resolvePromise, reject) => server.close((error) => error ? reject(error) : resolvePromise()))
  return address.port
}

async function startWorkBraid(binary: string, dataRoot: string, port: number, runtimeRoot: string, logName = 'workbraid.log'): Promise<RunningWorkBraid> {
  const origin = `http://127.0.0.1:${port}`
  const logFD = openSync(join(runtimeRoot, logName), 'a')
  const child = spawn(binary, ['-listen', `127.0.0.1:${port}`, '-data-dir', dataRoot, '-ui-dir', join(frontendRoot, 'dist')], {
    cwd: repositoryRoot, detached: true, stdio: ['ignore', logFD, logFD],
  })
  const application = { child, origin, logFD }
  try {
    await waitForServer(application)
    return application
  } catch (error) {
    await stopWorkBraid(application)
    throw error
  }
}

async function waitForServer(application: RunningWorkBraid) {
  const deadline = Date.now() + 15_000
  while (Date.now() < deadline) {
    if (application.child.exitCode !== null) throw new Error(`WorkBraid exited before becoming ready (${application.child.exitCode})`)
    try { if ((await fetch(application.origin)).ok) return } catch { /* not bound yet */ }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 50))
  }
  throw new Error('WorkBraid did not become ready within 15 seconds')
}

async function stopWorkBraid(application: RunningWorkBraid) {
  const pid = application.child.pid
  if (pid && application.child.exitCode === null && application.child.signalCode === null) {
    try { process.kill(-pid, 'SIGTERM') } catch (error) { if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error }
    await waitForExit(application.child, 3_000)
    if (application.child.exitCode === null && application.child.signalCode === null) {
      try { process.kill(-pid, 'SIGKILL') } catch (error) { if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error }
      await waitForExit(application.child, 3_000)
    }
  }
  closeSync(application.logFD)
}

async function waitForExit(child: ChildProcess, timeout: number) {
  const deadline = Date.now() + timeout
  while (child.exitCode === null && child.signalCode === null && Date.now() < deadline) await new Promise((resolvePromise) => setTimeout(resolvePromise, 25))
}
