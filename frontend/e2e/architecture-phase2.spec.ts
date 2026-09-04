import { expect, test, type Locator, type Page } from '@playwright/test'
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { closeSync, existsSync, mkdirSync, mkdtempSync, openSync, readdirSync, rmSync } from 'node:fs'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repositoryRoot = resolve(frontendRoot, '..')

type RunningWorkBraid = { child: ChildProcess; origin: string; logFD: number }

test('Phase 2 production path creates a slug project, nested diagrams, and reusable references across restart', async ({ page }) => {
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-phase2-'))
  const dataRoot = join(runtimeRoot, 'app-data')
  const binary = join(runtimeRoot, 'workbraid')
  let application: RunningWorkBraid | undefined

  try {
    mkdirSync(dataRoot, { recursive: true })
    run('go', ['build', '-buildvcs=false', '-o', binary, './cmd/workbraid'], repositoryRoot)
    const port = await unusedLoopbackPort()
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot)

    await page.goto(application.origin)
    await expect(page.getByRole('heading', { name: 'Projects' })).toBeVisible()
    await page.getByLabel('New project').fill('Phase Two System')
    await page.getByRole('button', { name: 'Create project' }).click()
    await expect(page).toHaveURL(`${application.origin}/projects/phase-two-system`)
    await expect(page.locator('.workspace-context strong')).toHaveText('Phase Two System')
    const bootstrap = await displayedRevision(page)

    const storePath = onlyArchitectureStore(dataRoot)
    expect(gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])).toBe(bootstrap)
    expect(gitBare(storePath, ['ls-tree', '-r', bootstrap])).toMatch(
      /^100644 blob [0-9a-f]{40}\tarchitecture\.yaml\n100644 blob [0-9a-f]{40}\tdiagrams\/root\.yaml$/,
    )
    expect(gitBare(storePath, ['show', `${bootstrap}:architecture.yaml`])).toContain('slug: phase-two-system')

    await addComponent(page, 'Gateway', 'Routes requests.\n')
    await addPendingComponent(page, 'Worker', 'Processes work.\n')
    await addPendingComponent(page, 'Records', 'Stores records.\n')
    await editPendingRelationships(page, 'Gateway', [
      { target: 'Worker — New component', label: 'calls' },
      { target: 'Worker — New component', label: 'calls async' },
    ])

    const acceptedIndex = page.getByRole('navigation', { name: 'Diagrams and components' })
    await expect(page.locator('form.component-editor')).toHaveCount(0)
    const proposalName = (await page.getByRole('button', { name: /^Showing / }).locator('span').first().textContent())?.trim()
    expect(proposalName).toBeTruthy()
    expect(proposalName).not.toBe('Accepted')
    const leaveGuard = page.getByRole('dialog', { name: 'Leave without keeping?' })
    await expect(leaveGuard).toHaveCount(0)
    await selectShowing(page, 'Accepted')
    await expect(leaveGuard).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Showing Accepted' })).toBeVisible()
    await expect(acceptedIndex.getByText('No components', { exact: true })).toBeVisible()
    await selectShowing(page, proposalName!)
    await expect(page.getByRole('heading', { name: proposalName! })).toBeVisible()
    await reviewAndAccept(page)
    await selectShowing(page, 'Accepted')
    const firstAccepted = await displayedRevision(page)
    expect(firstAccepted).not.toBe(bootstrap)
    await expect(page.getByRole('button', { name: 'Showing Accepted' })).toBeVisible()

    await acceptedIndex.getByText('Gateway', { exact: true }).click()
    await page.getByRole('button', { name: 'Edit component' }).click()
    await page.getByLabel('Description').fill('Routes requests and audits.\n')
    await page.locator('.relationship-row').first().getByLabel('Label').fill('dispatches')
    await page.getByRole('button', { name: 'Keep change' }).click()
    await createPendingDetail(page, 'Phase Two System', 'Gateway', 'Gateway internals')

    await movePendingHome(page, 'Phase Two System', 'Worker', 'Gateway internals')
    await createPendingDetail(page, 'Gateway internals', 'Worker', 'Worker internals')
    await movePendingHome(page, 'Phase Two System', 'Records', 'Worker internals')
    await showPendingReference(page, 'Gateway internals', 'Gateway')
    await showPendingReference(page, 'Gateway internals', 'Records')

    const rootPending = pendingDiagram(page, 'Phase Two System')
    await rootPending.getByRole('button', { name: 'Edit title' }).click()
    await page.getByLabel('Diagram title').fill('Platform')
    await page.getByRole('button', { name: 'Keep change' }).click()

    await page.getByRole('button', { name: 'Review changes' }).click()
    const review = page.locator('.review-workspace-pane')
    await expect(review).toBeVisible()
    await page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: /Gateway internals, Added/ }).click()
    await page.getByRole('button', { name: 'Before changes' }).click()
    await expect(page.getByText('That diagram exists only with the changes.')).toBeVisible()
    await expect(review.getByTestId('raw-diff')).toContainText('role: reference')
    await expect(review.getByTestId('raw-diff')).toContainText('Routes requests and audits.')
    await expect(review.getByTestId('raw-diff')).toContainText('dispatches')
    await page.getByRole('button', { name: 'With changes' }).click()
    await page.getByRole('button', { name: 'Update architecture' }).click()
    await selectShowing(page, 'Accepted')
    const secondAccepted = await displayedRevision(page)
    await expect(page.getByRole('button', { name: 'Showing Accepted' })).toBeVisible()

    await page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Gateway internals' }).click()
    const diagramIndex = page.getByRole('navigation', { name: 'Diagrams and components' })
    await expect(diagramIndex.getByRole('button', { name: /Gateway, Included here · Lives in Platform/ })).toBeVisible()
    await expect(diagramIndex.getByRole('button', { name: /Records, Included here · Lives in Worker internals/ })).toBeVisible()
    await diagramIndex.getByRole('button', { name: /Gateway, Included here/ }).click()
    await page.getByRole('button', { name: 'Stop showing here' }).click()
    await expect(pendingDiagram(page, 'Gateway internals').locator('li').filter({ hasText: 'Gateway' }).getByRole('button', { name: 'Stop showing here' })).toHaveCount(0)
    await stopPendingReference(page, 'Gateway internals', 'Records')

    await page.getByRole('button', { name: 'Review changes' }).click()
    await expect(page.getByTestId('raw-diff')).toContainText('role: reference')
    await page.getByRole('button', { name: 'Update architecture' }).click()
    await selectShowing(page, 'Accepted')
    const finalAccepted = await displayedRevision(page)
    expect(finalAccepted).not.toBe(secondAccepted)
    expect(gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])).toBe(finalAccepted)
    await expect(page.getByRole('button', { name: 'Showing Accepted' })).toBeVisible()

    await page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Gateway internals' }).click()
    await expect(page.getByRole('navigation', { name: 'Components that live elsewhere' }).getByRole('button', { name: /Gateway.*Lives in Platform/ })).toBeVisible()
    await expect(diagramIndex.getByText('Records', { exact: true })).toHaveCount(0)

    const externallyRenamed = replaceAcceptedSlug(storePath, finalAccepted, 'phase-two-renamed')
    expect(await displayedRevision(page)).toBe(finalAccepted)
    await page.getByRole('button', { name: 'Refresh' }).click()
    await expect(page).toHaveURL(`${application.origin}/projects/phase-two-renamed`)
    expect(await displayedRevision(page)).toBe(externallyRenamed)
    await page.goto(`${application.origin}/projects/phase-two-system`)
    await expect(page.getByRole('heading', { name: 'Project not found' })).toBeVisible()
    await page.goto(`${application.origin}/projects/phase-two-renamed`)
    expect(await displayedRevision(page)).toBe(externallyRenamed)

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'restart.log')
    await page.goto(`${application.origin}/projects/phase-two-renamed`)
    await expect(page.locator('.workspace-context strong')).toHaveText('Phase Two System')
    expect(await displayedRevision(page)).toBe(externallyRenamed)
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Gateway internals' })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Worker internals' })).toBeVisible()

    await page.reload()
    expect(await displayedRevision(page)).toBe(externallyRenamed)
    await page.getByRole('button', { name: 'Open another project' }).click()
    await expect(page.getByRole('heading', { name: 'Projects' })).toBeVisible()
    await page.getByRole('button', { name: 'Phase Two System' }).click()
    expect(await displayedRevision(page)).toBe(externallyRenamed)
    expect(findDatabaseFiles(dataRoot)).toEqual([])
  } finally {
    if (application) await stopWorkBraid(application)
    rmSync(runtimeRoot, { recursive: true, force: true })
  }
})

async function addComponent(page: Page, title: string, description: string) {
  await page.getByRole('button', { name: 'Add component' }).click()
  await fillComponent(page, title, description)
}

async function addPendingComponent(page: Page, title: string, description: string) {
  await pendingDiagram(page, 'Phase Two System').getByRole('button', { name: 'Add component' }).click()
  await fillComponent(page, title, description)
}

async function fillComponent(page: Page, title: string, description: string) {
  await page.getByLabel('Title').fill(title)
  await page.getByLabel('Description').fill(description)
  await page.getByRole('button', { name: 'Keep change' }).click()
  await expect(page.locator('.changes-in-progress')).toBeVisible()
}

async function editPendingRelationships(page: Page, title: string, relationships: Array<{ target: string; label: string }>) {
  const item = page.locator('.changes-in-progress > ul > li').filter({ hasText: title })
  await item.getByRole('button', { name: 'Edit' }).click()
  for (const relationship of relationships) {
    await page.getByRole('button', { name: 'Add relationship' }).click()
    const rows = page.locator('.relationship-row')
    const row = rows.nth((await rows.count()) - 1)
    await row.getByLabel('Target').selectOption({ label: relationship.target })
    await row.getByLabel('Label').fill(relationship.label)
  }
  await page.getByRole('button', { name: 'Keep change' }).click()
}

function pendingDiagram(page: Page, title: string): Locator {
  return page.locator('.pending-diagram-row').filter({ has: page.locator('.pending-diagram-title strong', { hasText: title }) })
}

async function movePendingHome(page: Page, sourceDiagram: string, component: string, destination: string) {
  const row = pendingDiagram(page, sourceDiagram).locator('li').filter({ hasText: component })
  await row.getByRole('button', { name: `Change where ${component} lives`, exact: true }).click()
  await page.locator('form.diagram-editor').getByRole('combobox').selectOption({ label: destination })
  await page.getByRole('button', { name: 'Keep change' }).click()
}

async function createPendingDetail(page: Page, sourceDiagram: string, component: string, title: string) {
  const row = pendingDiagram(page, sourceDiagram).locator('li').filter({ hasText: component })
  await row.getByRole('button', { name: 'Create detail diagram' }).click()
  await page.getByLabel('Diagram title').fill(title)
  await page.getByRole('button', { name: 'Keep change' }).click()
}

async function showPendingReference(page: Page, diagram: string, component: string) {
  const picker = pendingDiagram(page, diagram).getByLabel('Show component here')
  const value = await picker.locator('option').filter({ hasText: component }).getAttribute('value')
  expect(value).toBeTruthy()
  await picker.selectOption(value!)
  await expect(pendingDiagram(page, diagram).locator('li').filter({ hasText: component }).getByRole('button', { name: 'Stop showing here' })).toBeVisible()
}

async function stopPendingReference(page: Page, diagram: string, component: string) {
  const row = pendingDiagram(page, diagram).locator('li').filter({ hasText: component })
  await row.getByRole('button', { name: 'Stop showing here' }).click()
  await expect(row.getByRole('button', { name: 'Stop showing here' })).toHaveCount(0)
}

async function reviewAndAccept(page: Page) {
  await page.getByRole('button', { name: 'Review changes' }).click()
  await expect(page.locator('.review-workspace-pane')).toBeVisible()
  await page.getByRole('button', { name: 'Update architecture' }).click()
}

async function selectShowing(page: Page, name: string) {
  const trigger = page.getByRole('button', { name: /^Showing / })
  await trigger.click()
  await expect(trigger).toHaveAttribute('aria-expanded', 'true')
  await page.getByRole('option', { name, exact: true }).click()
  await expect(trigger).toHaveAttribute('aria-expanded', 'false')
}

async function displayedRevision(page: Page) {
  const details = page.locator('details.technical-details')
  if (!(await details.getAttribute('open'))) await details.locator('summary').click()
  const value = details.locator('dt', { hasText: 'Revision' }).locator('xpath=following-sibling::dd[1]')
  return (await value.textContent())?.trim() ?? ''
}

function onlyArchitectureStore(dataRoot: string) {
  const entries = readdirSync(join(dataRoot, 'architecture')).filter((name) => name.endsWith('.git'))
  expect(entries).toHaveLength(1)
  return join(dataRoot, 'architecture', entries[0])
}

function findDatabaseFiles(root: string): string[] {
  if (!existsSync(root)) return []
  const found: string[] = []
  const visit = (directory: string) => {
    for (const name of readdirSync(directory, { withFileTypes: true })) {
      const path = join(directory, name.name)
      if (name.isDirectory()) visit(path)
      else if (name.name.endsWith('.db') || name.name.endsWith('.sqlite')) found.push(path)
    }
  }
  visit(root)
  return found
}

function gitBare(storePath: string, arguments_: string[]) {
  return run('git', ['--git-dir', storePath, ...arguments_], repositoryRoot)
}

function replaceAcceptedSlug(storePath: string, parent: string, slug: string) {
  const manifest = gitBare(storePath, ['show', `${parent}:architecture.yaml`]).replace('slug: phase-two-system', `slug: ${slug}`) + '\n'
  const manifestBlob = runInput('git', ['--git-dir', storePath, 'hash-object', '-w', '--stdin'], repositoryRoot, manifest)
  const entries = gitBare(storePath, ['ls-tree', parent]).split('\n').map((entry) => entry.endsWith('\tarchitecture.yaml')
    ? `100644 blob ${manifestBlob}\tarchitecture.yaml`
    : entry)
  const tree = runInput('git', ['--git-dir', storePath, 'mktree'], repositoryRoot, entries.join('\n') + '\n')
  const commit = runInput('git', [
    '-c', 'user.name=External Human', '-c', 'user.email=human@workbraid.invalid',
    '--git-dir', storePath, 'commit-tree', tree, '-p', parent,
  ], repositoryRoot, 'Change project address\n')
  run('git', ['--git-dir', storePath, 'update-ref', 'refs/heads/accepted', commit, parent], repositoryRoot)
  return commit
}

function run(command: string, arguments_: string[], cwd: string) {
  const result = spawnSync(command, arguments_, { cwd, encoding: 'utf8', env: { ...process.env, GIT_CONFIG_NOSYSTEM: '1', GIT_TERMINAL_PROMPT: '0' } })
  if (result.status !== 0) throw new Error(`${command} ${arguments_.join(' ')} failed (${result.status}):\n${result.stdout}\n${result.stderr}`)
  return result.stdout.trim()
}

function runInput(command: string, arguments_: string[], cwd: string, input: string) {
  const result = spawnSync(command, arguments_, { cwd, input, encoding: 'utf8', env: { ...process.env, GIT_CONFIG_NOSYSTEM: '1', GIT_TERMINAL_PROMPT: '0' } })
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
