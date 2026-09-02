import { expect, test, type Page } from '@playwright/test'
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { closeSync, mkdirSync, mkdtempSync, openSync, rmSync } from 'node:fs'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repositoryRoot = resolve(frontendRoot, '..')

type RunningWorkBraid = { child: ChildProcess; origin: string; logFD: number }
type AgentEnvelope = {
  ok: boolean
  context: { project?: { store_id: string; slug: string }; accepted_revision?: string; pending_generation?: number | null }
  result?: Record<string, any>
  error?: { code: string }
}

test('agent work stays coherent with the production browser across stale submission, review, acceptance, and restart', async ({ page }) => {
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-agent-access-'))
  const dataRoot = join(runtimeRoot, 'app-data')
  const binary = join(runtimeRoot, 'workbraid')
  let application: RunningWorkBraid | undefined

  try {
    mkdirSync(dataRoot, { recursive: true })
    run('go', ['build', '-buildvcs=false', '-o', binary, './cmd/workbraid'], repositoryRoot)
    const port = await unusedLoopbackPort()
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot)

    await page.goto(application.origin)
    await page.getByLabel('New project').fill('Agent shared system')
    await page.getByRole('button', { name: 'Create project' }).click()
    await expect(page).toHaveURL(`${application.origin}/projects/agent-shared-system`)

    const accepted = agent(binary, application.origin, ['architecture', 'inspect'])
    const storeID = accepted.context.project!.store_id
    const revision = accepted.context.accepted_revision!
    const rootDiagramID = accepted.result!.root_diagram_id as string

    // Hold an editor based on generation null while the agent advances the
    // one backend-owned pending set. The stale browser submission must lose.
    await page.getByRole('button', { name: 'Add component' }).click()
    await page.getByLabel('Title').fill('Stale browser component')
    await page.getByLabel('Description').fill('Must not overwrite agent work.\n')
    const gateway = agent(binary, application.origin, [
      'component', 'create', '--store-id', storeID, '--accepted-revision', revision, '--generation', 'none',
      '--diagram-id', rootDiagramID, '--title', 'Agent gateway', '--description', 'Routes requests.\n',
    ])
    expect(gateway.context.pending_generation).toBe(1)
    await page.getByRole('button', { name: 'Keep change' }).click()
    await expect(page.getByRole('alert')).toHaveText('Changes are already in progress for another architecture.')
    const afterStale = agent(binary, application.origin, ['changes', 'inspect'])
    expect(afterStale.context.pending_generation).toBe(1)
    expect((afterStale.result!.components as Array<{ title: string }>).map((component) => component.title)).toEqual(['Agent gateway'])

    await page.reload()
    await expect(page.getByRole('heading', { name: 'Changes in progress' })).toBeVisible()
    await expect(page.locator('.changes-in-progress > ul > li').filter({ hasText: 'Agent gateway' })).toBeVisible()

    const gatewayID = gateway.result!.component_id as string
    const worker = agent(binary, application.origin, [
      'component', 'create', '--store-id', storeID, '--accepted-revision', revision, '--generation', '1',
      '--diagram-id', rootDiagramID, '--title', 'Agent worker', '--description', 'Processes work.\n',
    ])
    const workerID = worker.result!.component_id as string
    const relationship = agent(binary, application.origin, [
      'relationship', 'add', '--store-id', storeID, '--accepted-revision', revision, '--generation', '2',
      '--source-id', gatewayID, '--target-id', workerID, '--label', 'dispatches',
    ])
    const detail = agent(binary, application.origin, [
      'diagram', 'create-detail', '--store-id', storeID, '--accepted-revision', revision,
      '--generation', String(relationship.context.pending_generation), '--component-id', gatewayID, '--title', 'Gateway internals',
    ])
    expect(detail.context.pending_generation).toBe(4)
    const reviewed = agent(binary, application.origin, [
      'changes', 'review', '--store-id', storeID, '--accepted-revision', revision, '--generation', '4',
    ])
    expect(reviewed.result!.candidate_tree).toMatch(/^[0-9a-f]{40}$/)

    await page.reload()
    await expect(page.locator('.review-workspace-pane')).toBeVisible()
    await expect(page.getByTestId('raw-diff')).toContainText('Agent gateway')
    await expect(page.getByTestId('raw-diff')).toContainText('dispatches')
    await page.getByRole('button', { name: 'Update architecture' }).click()
    await expect(page.locator('.review-workspace-pane')).toHaveCount(0)

    const agentAccepted = agent(binary, application.origin, ['architecture', 'inspect'])
    expect(agentAccepted.context.accepted_revision).not.toBe(revision)
    expect(agentAccepted.context.pending_generation).toBeNull()
    const acceptedComponents = agentAccepted.result!.components as Array<{ id: string; title: string; relationships: Array<{ target_id: string; label: string }> }>
    expect(acceptedComponents.find((component) => component.id === gatewayID)?.relationships).toEqual([{ target_id: workerID, label: 'dispatches' }])
    expect((agentAccepted.result!.diagrams as Array<{ id: string; title: string }>).some((diagram) => diagram.title === 'Gateway internals')).toBe(true)

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'restart.log')
    await page.goto(`${application.origin}/projects/agent-shared-system`)
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByText('Agent gateway', { exact: true })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Gateway internals' })).toBeVisible()
    expect(await displayedRevision(page)).toBe(agentAccepted.context.accepted_revision)
    const reconstructed = agent(binary, application.origin, ['architecture', 'inspect'])
    expect(reconstructed.result).toEqual(agentAccepted.result)
  } finally {
    if (application) await stopWorkBraid(application)
    rmSync(runtimeRoot, { recursive: true, force: true })
  }
})

function agent(binary: string, origin: string, arguments_: string[]): AgentEnvelope {
  const output = run(binary, ['--server', origin, '--json', ...arguments_], repositoryRoot)
  const envelope = JSON.parse(output) as AgentEnvelope
  expect(envelope.ok, `${arguments_.join(' ')} failed: ${JSON.stringify(envelope.error)}`).toBe(true)
  return envelope
}

async function displayedRevision(page: Page) {
  const details = page.locator('details.technical-details')
  if (!(await details.getAttribute('open'))) await details.locator('summary').click()
  return (await details.locator('dt', { hasText: 'Revision' }).locator('xpath=following-sibling::dd[1]').textContent())?.trim()
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
