import { expect, test as base, type Page } from '@playwright/test'
import { spawn, spawnSync } from 'node:child_process'
import { closeSync, mkdtempSync, openSync } from 'node:fs'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const evidence = mkdtempSync(join(tmpdir(), 'workbraid-reconciliation-browser-'))
const binary = join(evidence, 'workbraid')
type Proposal = { id: string; generation: number; [key: string]: any }
type App = { origin: string; data: string; cli: (args: string[], input?: string) => any; store: string; slug: string; main: string; restart: () => Promise<void> }
const test = base.extend<{ app: App }>({
  app: async ({}, use, info) => {
    const portServer = createServer()
    await new Promise<void>((yes) => portServer.listen(0, '127.0.0.1', yes))
    const address = portServer.address(); if (!address || typeof address === 'string') throw new Error('No loopback port')
    await new Promise<void>((yes) => portServer.close(() => yes()))
    const origin = `http://127.0.0.1:${address.port}`
    const data = mkdtempSync(join(evidence, `case-${info.testId.replaceAll(/[^a-z0-9]/gi, '').slice(-12)}-`))
    const log = openSync(join(data, 'server.log'), 'a')
    const launch = () => spawn(binary, ['--listen', `127.0.0.1:${address.port}`, '--data-dir', data, '--ui-dir', join(root, 'frontend/dist')], { cwd: root, stdio: ['ignore', log, log] })
    let child = launch()
    const ready = async () => {
      const deadline = Date.now() + 10_000
      let ready = false
      while (Date.now() < deadline) {
        if (child.exitCode !== null) throw new Error('Task server exited')
        try { ready = (await fetch(origin)).ok } catch { /* awaiting this task's listener */ }
        if (ready) break
        await new Promise((yes) => setTimeout(yes, 40))
      }
      if (!ready) throw new Error('Task server did not bind')
    }
    const stop = async () => {
      if (child.exitCode !== null || child.signalCode !== null) return
      await new Promise<void>((yes) => {
        const timer = setTimeout(() => child.kill('SIGKILL'), 3000)
        child.once('exit', () => { clearTimeout(timer); yes() })
        child.kill('SIGTERM')
      })
    }
    try {
      await ready()
      const cli = (args: string[], input?: string) => {
        const command = spawnSync(binary, ['--server', origin, '--json', ...args], { cwd: root, encoding: 'utf8', input })
        const result = JSON.parse(command.stdout)
        expect(result.ok, `${args.join(' ')}: ${command.stdout} ${command.stderr}`).toBe(true)
        return result
      }
      cli(['project', 'create', '--name', 'Reconciliation browser'])
      const inspected = cli(['architecture', 'inspect'])
      await use({ origin, data, cli, store: inspected.context.project.store_id, slug: inspected.context.project.slug, main: inspected.result.root_diagram_id, restart: async () => { await stop(); child = launch(); await ready() } })
    } finally {
      await stop()
      closeSync(log)
    }
  },
})
test.beforeAll(() => {
  const built = spawnSync('go', ['build', '-buildvcs=false', '-o', binary, './cmd/workbraid'], { cwd: root, encoding: 'utf8' })
  expect(built.status, built.stderr).toBe(0)
})
test.use({ viewport: { width: 1440, height: 1000 } })
function state(app: App, proposal: Proposal) { return ['--store-id', app.store, '--change-set-id', proposal.id, '--generation', String(proposal.generation)] }
function proposal(app: App, name: string): Proposal { return app.cli(['change-set', 'create', '--store-id', app.store, '--accepted-revision', app.cli(['architecture', 'inspect']).context.accepted_revision, '--name', name]).result }
function mutate(app: App, p: Proposal, command: string[], args: string[], input?: string) { const result = app.cli([...command, ...state(app, p), ...args], input).result; p.generation = result.generation; return result }
function component(app: App, p: Proposal, name: string) { return mutate(app, p, ['component', 'create'], ['--diagram-id', app.main, '--title', name]).component_id as string }
function detail(app: App, p: Proposal, anchor: string, title: string) { return mutate(app, p, ['diagram', 'create-detail'], ['--component-id', anchor, '--title', title]).diagram_id as string }
function inspect(app: App, p: Proposal) { return app.cli(['change-set', 'inspect', '--store-id', app.store, '--change-set-id', p.id]).result }
function accept(app: App, p: Proposal) {
  const review = app.cli(['change-set', 'review', ...state(app, p)]).result
  app.cli(['architecture', 'update', ...state(app, p), '--base-revision', review.base_revision, '--candidate-tree', review.candidate_tree])
}
async function openProposal(app: App, page: Page, p: Proposal) {
  await page.goto(`${app.origin}/projects/${app.slug}/proposals/${p.id}`)
  await expect(page.getByRole('heading', { name: p.name, exact: true })).toBeVisible()
}
async function screenshot(page: Page, name: string) {
  await page.screenshot({ path: join(evidence, `${name}.png`), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
}

test('ordinary browser Change parent component keeps the existing child', async ({ app, page }) => {
  const setup = proposal(app, 'Initial composition')
  const gateway = component(app, setup, 'Gateway'); const worker = component(app, setup, 'Worker')
  const child = detail(app, setup, gateway, 'Operations'); accept(app, setup)
  await page.goto(`${app.origin}/projects/${app.slug}`)
  await page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: /Operations/ }).click()
  await page.getByRole('button', { name: 'Change parent component', exact: true }).click()
  const form = page.locator('.diagram-editor')
  await expect(form.getByRole('heading', { name: /Change parent component/ })).toBeVisible()
  await screenshot(page, 'ordinary-parent-options')
  const before = app.cli(['change-set', 'list', '--store-id', app.store]).result.change_sets
  expect(before.filter((item: Proposal) => item.lifecycle === 'active')).toHaveLength(0)
  await form.getByRole('radio', { name: 'Worker', exact: true }).check()
  await form.getByRole('button', { name: 'Keep change', exact: true }).click()
  await expect(form).toHaveCount(0)
  const active = app.cli(['change-set', 'list', '--store-id', app.store]).result.change_sets.find((item: Proposal) => item.lifecycle === 'active')
  expect(active.detail_reassignments).toEqual([{ diagram_id: child, anchor_component_id: worker }])
  expect(active.components).toEqual([])
  await screenshot(page, 'ordinary-parent-kept')
})

test('exact long Description decisions protect local choices and historical feedback', async ({ app, page }) => {
  const setup = proposal(app, 'Initial Gateway'); const gateway = component(app, setup, 'Gateway'); accept(app, setup)
  const p = proposal(app, 'Document Gateway')
  const proposedBody = '\n' + Array.from({ length: 70 }, (_, n) => `Proposed line ${n}: exact Markdown **details** and a long responsibility statement.\r\n`).join('') + 'PROPOSED END\r\n'
  mutate(app, p, ['component', 'edit'], ['--component-id', gateway, '--description-file', '-'], proposedBody)
  const binding = app.cli(['change-set', 'review', ...state(app, p)]).result
  const feedback = app.cli(['review-submission', 'submit', ...state(app, p), '--reviewed-state', binding.reviewed_state, '--base-revision', binding.base_revision, '--candidate-tree', binding.candidate_tree, '--verdict', 'comment', '--author', 'Historical reviewer', '--body', 'Keep this exact old feedback.']).result
  const a = proposal(app, 'Accepted documentation')
  const acceptedBody = '\n' + Array.from({ length: 65 }, (_, n) => `Accepted line ${n}: separately authored exact content.\n`).join('') + 'ACCEPTED END\n'
  mutate(app, a, ['component', 'edit'], ['--component-id', gateway, '--description-file', '-'], acceptedBody); accept(app, a)
  await openProposal(app, page, p)
  await page.getByRole('button', { name: 'Reconcile with Accepted', exact: true }).click()
  const task = page.locator('.reconciliation-task')
  await expect(task.getByRole('heading', { name: 'Reconcile with Accepted', exact: true })).toBeVisible()
  await expect(task.locator('.reconciliation-sides')).toContainText('PROPOSED END')
  expect(await task.locator('.reconciliation-sides pre').last().evaluate((node) => node.scrollHeight > node.clientHeight)).toBe(true)
  await screenshot(page, 'exact-long-descriptions')
  await task.getByRole('button', { name: 'Choose manually', exact: true }).click()
  await task.getByRole('textbox', { name: 'Exact Markdown Description', exact: true }).fill('A local exact decision\n\n')
  await task.getByRole('button', { name: 'Use proposed', exact: true }).click()
  await task.getByRole('button', { name: 'Back to proposal', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Leave without keeping?' })).toBeVisible()
  await page.getByRole('button', { name: 'Keep editing', exact: true }).click()
  await task.getByRole('button', { name: 'Check choices', exact: true }).click()
  await expect(task.getByRole('button', { name: 'Apply reconciliation', exact: true })).toBeVisible()
  await screenshot(page, 'description-checked')
  await task.getByRole('button', { name: 'Apply reconciliation', exact: true }).click()
  await expect(task).toHaveCount(0)
  await expect(page.getByText('Reconciliation applied to this proposal. Review changes to continue.')).toBeVisible()
  const result = inspect(app, p)
  expect(result.candidate.components.find((item: any) => item.id === gateway).description).toBe(proposedBody)
  expect(result.review).toBeNull()
  const historical = app.cli(['review-submission', 'inspect', '--store-id', app.store, '--change-set-id', p.id, '--review-id', feedback.id]).result
  expect(historical.reviewed_state).toBe(binding.reviewed_state)
  await page.goto(`${app.origin}${new URL(historical.review_url, app.origin).pathname}`)
  await expect(page.getByText('Keep this exact old feedback.')).toBeVisible()
  await screenshot(page, 'historical-feedback-route')
  await app.restart()
  await page.reload()
  await expect(page.getByText('Keep this exact old feedback.')).toBeVisible()
  const restarted = inspect(app, p)
  expect(restarted.change_set_state).toBe(result.change_set_state)
  expect(restarted.candidate_tree).toBe(result.candidate_tree)
  expect(restarted.candidate.components.find((item: any) => item.id === gateway).description).toBe(proposedBody)
})

for (const side of ['accepted', 'proposed'] as const) {
  test(`both competing children survive explicit ${side} preference`, async ({ app, page }) => {
    const setup = proposal(app, 'Initial parents'); const gateway = component(app, setup, 'Gateway'); const worker = component(app, setup, 'Worker'); accept(app, setup)
    const p = proposal(app, 'Runtime proposal'); const runtime = detail(app, p, gateway, 'Runtime')
    const a = proposal(app, 'Operations accepted'); const operations = detail(app, a, gateway, 'Operations')
    mutate(app, a, ['diagram', 'show-component'], ['--diagram-id', operations, '--component-id', gateway])
    mutate(app, a, ['diagram', 'show-component'], ['--diagram-id', operations, '--component-id', worker])
    accept(app, a)
    await openProposal(app, page, p)
    await page.getByRole('button', { name: 'Reconcile with Accepted', exact: true }).click()
    const task = page.locator('.reconciliation-task')
    await task.getByRole('button', { name: side === 'accepted' ? 'Keep Operations on Gateway' : 'Keep Runtime on Gateway', exact: true }).click()
    const displaced = side === 'accepted' ? 'Runtime' : 'Operations'
    await task.getByRole('group', { name: `Parent component for ${displaced}`, exact: true }).getByRole('radio', { name: /Worker/ }).check()
    await screenshot(page, `both-child-assignments-${side}`)
    await expect(task.locator('.reconciliation-assignments')).not.toContainText('.md')
    await task.getByRole('button', { name: 'Check choices', exact: true }).click()
    await expect(task.getByRole('button', { name: 'Apply reconciliation', exact: true })).toBeVisible()
    await task.getByRole('button', { name: 'Preview result', exact: true }).click()
    const map = task.getByTestId('architecture-map')
    await expect(map).toBeVisible()
    const rendering = await map.evaluate((element) => {
      const graph = (element as any)._cyreg.cy
      return { width: graph.width(), height: graph.height(), nodes: graph.nodes().map((node: any) => ({ title: node.data('displayLabel'), kind: node.data('nodeKind'), width: node.renderedWidth(), fontPixels: parseFloat(node.style('font-size')) * graph.zoom(), vertical: node.style('text-valign') })) }
    })
    expect(rendering.width).toBeGreaterThan(700)
    expect(rendering.height).toBeGreaterThanOrEqual(350)
    expect(rendering.nodes).toHaveLength(2)
    for (const node of rendering.nodes) {
      expect(node.kind).toBe('home')
      expect(node.width).toBeGreaterThan(100)
      expect(node.fontPixels).toBeGreaterThanOrEqual(12)
      expect(node.vertical).toBe('center')
    }
    await screenshot(page, `competing-ready-${side}`)
    const navigation = task.getByRole('navigation', { name: 'Result diagrams', exact: true })
    await navigation.getByRole('button', { name: 'Operations', exact: true }).click()
    // Same two UUIDs, different Diagram projection: reference styles must be
    // recalculated rather than retaining the root's home nodes at the same tree.
    expect(await map.evaluate((element) => (element as any)._cyreg.cy.nodes().map((node: any) => node.data('nodeKind')))).toEqual(['reference', 'reference'])
    await navigation.getByRole('button', { name: 'Runtime', exact: true }).click()
    await expect(task.getByText('This diagram has no components.', { exact: true })).toBeVisible()
    await navigation.getByRole('button', { name: 'Reconciliation browser', exact: true }).click()
    await expect(navigation.getByRole('button', { name: 'Reconciliation browser', exact: true })).toHaveAttribute('aria-current', 'true')
    await task.getByRole('button', { name: 'Apply reconciliation', exact: true }).click()
    await expect(task).toHaveCount(0)
    const diagrams = inspect(app, p).candidate.diagrams
    expect(diagrams.find((d: any) => d.id === operations).parent_anchor_component_id).toBe(side === 'accepted' ? gateway : worker)
    expect(diagrams.find((d: any) => d.id === runtime).parent_anchor_component_id).toBe(side === 'accepted' ? worker : gateway)
  })
}

test('no free anchor remains unresolved without deleting either child', async ({ app, page }) => {
  const setup = proposal(app, 'Only one parent'); const gateway = component(app, setup, 'Gateway'); accept(app, setup)
  const p = proposal(app, 'Runtime proposal'); detail(app, p, gateway, 'Runtime')
  const a = proposal(app, 'Operations accepted'); detail(app, a, gateway, 'Operations'); accept(app, a)
  const original = inspect(app, p)
  await openProposal(app, page, p)
  await page.getByRole('button', { name: 'Reconcile with Accepted', exact: true }).click()
  const task = page.locator('.reconciliation-task')
  await task.getByRole('button', { name: 'Keep Operations on Gateway', exact: true }).click()
  await expect(task.getByText(/There is no free valid parent component/)).toBeVisible()
  await task.getByRole('button', { name: 'Check choices', exact: true }).click()
  await expect(task.getByRole('button', { name: 'Apply reconciliation', exact: true })).toHaveCount(0)
  expect(inspect(app, p).change_set_state).toBe(original.change_set_state)
  await screenshot(page, 'no-free-anchor')
})

test('automatic same-result reconciliation leaves an empty active proposal', async ({ app, page }) => {
  const setup = proposal(app, 'Initial Gateway'); const gateway = component(app, setup, 'Gateway'); accept(app, setup)
  const p = proposal(app, 'Shared documentation')
  mutate(app, p, ['component', 'edit'], ['--component-id', gateway, '--description-file', '-'], '\nShared exact description.\n')
  const a = proposal(app, 'Accepted same documentation')
  mutate(app, a, ['component', 'edit'], ['--component-id', gateway, '--description-file', '-'], '\nShared exact description.\n'); accept(app, a)
  const before = inspect(app, p)
  await openProposal(app, page, p)
  await page.getByRole('button', { name: 'Reconcile with Accepted', exact: true }).click()
  const task = page.locator('.reconciliation-task')
  await expect(task.getByRole('heading', { name: 'Combined without conflicts', exact: true })).toBeVisible()
  await expect(task.getByText('No Architecture changes remain.', { exact: true })).toBeVisible()
  await task.getByText('See combined changes', { exact: true }).click()
  await screenshot(page, 'automatic-empty-residual')
  await task.getByRole('button', { name: 'Apply reconciliation', exact: true }).click()
  await expect(task).toHaveCount(0)
  const after = inspect(app, p)
  expect(after.generation).toBe(before.generation + 1)
  expect(after.lifecycle).toBe('active')
  expect(after.components).toEqual([])
  expect(after.review).toBeNull()
  await app.restart()
  await page.reload()
  await expect(page.getByRole('heading', { name: p.name, exact: true })).toBeVisible()
  expect(inspect(app, p).change_set_state).toBe(after.change_set_state)
})

test('divergent external UUID collision explains the unsupported operation', async ({ app, page }) => {
  const p = proposal(app, 'Independent Gateway'); const gateway = component(app, p, 'Gateway')
  mutate(app, p, ['component', 'edit'], ['--component-id', gateway, '--title', 'External Gateway'])
  const externalTree = inspect(app, p).candidate_tree
  mutate(app, p, ['component', 'edit'], ['--component-id', gateway, '--title', 'Gateway'])
  const before = inspect(app, p)
  // This bounded fixture supplies a valid external Accepted snapshot colliding
  // with P's new identity. Its tree was built by the ordinary constructor.
  const git = (args: string[], input?: string) => {
    const result = spawnSync('git', ['-c', 'core.hooksPath=/dev/null', '-c', 'commit.gpgSign=false', '-c', 'user.name=External fixture', '-c', 'user.email=fixture@example.test', '--git-dir', join(app.data, 'architecture', `${app.store}.git`), ...args], { encoding: 'utf8', input })
    expect(result.status, result.stderr).toBe(0)
    return result.stdout.trim()
  }
  const external = git(['commit-tree', externalTree], 'External independently authored Architecture\n')
  git(['update-ref', 'refs/heads/accepted', external, before.base_revision])
  app.cli(['architecture', 'refresh', '--store-id', app.store, '--accepted-revision', before.base_revision])
  await openProposal(app, page, p)
  await page.getByRole('button', { name: 'Reconcile with Accepted', exact: true }).click()
  const task = page.locator('.reconciliation-task')
  await expect(task.getByText(/same ID but have different content or dependent facts/)).toBeVisible()
  await expect(task.getByRole('button', { name: 'Apply reconciliation', exact: true })).toHaveCount(0)
  await screenshot(page, 'unsupported-identity-collision')
  expect(inspect(app, p).change_set_state).toBe(before.change_set_state)
})
