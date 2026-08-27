import { expect, test, type Locator, type Page } from '@playwright/test'
import { createHash } from 'node:crypto'
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { closeSync, lstatSync, mkdirSync, mkdtempSync, openSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repositoryRoot = resolve(frontendRoot, '..')

type SourceEvidence = {
  head: string
  status: string
  index: string
  files: string
}

type RunningWorkBraid = {
  child: ChildProcess
  origin: string
  logFD: number
}

test('Phase 1 reviews one bound multi-change candidate and reconstructs the accepted result', async ({ page }) => {
  page.setDefaultTimeout(10_000)
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-phase1-review-'))
  const sourceRoot = join(runtimeRoot, 'source-project')
  const dataRoot = join(runtimeRoot, 'app-data')
  const binary = join(runtimeRoot, 'workbraid')
  let application: RunningWorkBraid | undefined

  try {
    createSourceRepository(sourceRoot)
    mkdirSync(dataRoot, { recursive: true })
    const sourceBefore = sourceEvidence(sourceRoot)

    run('go', ['build', '-buildvcs=false', '-o', binary, './cmd/workbraid'], repositoryRoot)
    const port = await unusedLoopbackPort()
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot)

    await openProject(page, application.origin, sourceRoot)
    await page.getByRole('button', { name: 'Set up architecture' }).click()
    await page.getByRole('button', { name: 'Set up', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Components', exact: true })).toBeVisible()

    await addComponent(page, 'Gateway', 'Accepted gateway body.\n')
    await addComponent(page, 'Worker', 'Accepted worker body.\n')
    await addComponent(page, 'Docs', 'Accepted docs body.\n')
    await editPending(page, 'Gateway')
    await addRelationship(page, 'Worker — New component', 'calls')
    await addRelationship(page, 'Worker — New component', 'calls')
    await keepChange(page)
    await editPending(page, 'Worker')
    await addRelationship(page, 'Gateway — New component', 'reports to')
    await keepChange(page)

    await page.getByRole('button', { name: 'Review changes' }).click()
    await expect(page.getByRole('heading', { name: 'Review changes' })).toBeVisible()
    await page.getByRole('button', { name: 'Update architecture' }).click()
    const baseRevision = await displayedRevision(page)
    const storePath = onlyArchitectureStore(dataRoot)
    expect(gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])).toBe(baseRevision)

    await addComponent(page, 'Queue', 'Candidate-only queue body.\n')
    await editAccepted(page, 'Gateway')
    await page.getByLabel('Description').fill('Changed gateway body.\n')
    const gatewayRelationships = page.getByRole('group', { name: 'Outgoing relationships' })
    await gatewayRelationships.getByLabel('Target').nth(1).selectOption({ label: 'Queue — New component' })
    await keepChange(page)

    await editAccepted(page, 'Docs')
    await page.getByLabel('Description').fill('Changed docs body.\n')
    await keepChange(page)

    await editAccepted(page, 'Worker')
    await addRelationship(page, 'Queue — New component', '')
    await keepChange(page)

    const componentIndex = page.getByRole('navigation', { name: 'Components' })
    await componentIndex.getByRole('button', { name: 'Gateway', exact: true }).click()
    await expect(page.getByText('Accepted gateway body.', { exact: true })).toBeVisible()
    await expect(componentIndex.getByText('Queue', { exact: true })).toHaveCount(0)
    await expect(page.getByTestId('architecture-map')).toBeVisible()

    await page.getByRole('button', { name: /Changes in progress/ }).click()
    await page.getByRole('button', { name: 'Review changes' }).click()
    const reviewAlert = await page.getByRole('alert')
    await expect(reviewAlert).toContainText('Worker')
    const workerRow = page.locator('.changes-in-progress li').filter({ hasText: 'Worker' })
    await expect(workerRow).toHaveAttribute('aria-invalid', 'true')
    await expect(workerRow.getByText('Needs attention')).toBeVisible()
    await reviewAlert.getByRole('button', { name: 'Fix relationship' }).click()
    const invalidLabel = page.getByLabel('Label').nth(1)
    await expect(invalidLabel).toBeFocused()
    await expect(invalidLabel).toHaveAttribute('aria-invalid', 'true')
    await invalidLabel.fill('observes')
    await keepChange(page)

    await page.getByRole('button', { name: 'Review changes' }).click()
    await expect(page.getByRole('heading', { name: 'Review changes' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'With changes' })).toHaveAttribute('aria-pressed', 'true')
    await expect(page.getByRole('button', { name: 'Added component: Queue' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Content changed: Gateway' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Content changed: Docs' })).toBeVisible()
    await expect(page.getByRole('button', { name: /Added relationship: Gateway — calls — Queue/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /Added relationship: Worker — observes — Queue/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /Removed relationship: Gateway — calls — Worker/ })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Content changed: Worker' })).toHaveCount(0)

    await page.getByRole('button', { name: 'Added component: Queue' }).click()
    await expect(page.getByRole('heading', { name: 'Queue' })).toBeVisible()
    await expect(page.getByText('Candidate-only queue body.', { exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Before changes' }).click()
    await expect(page.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    await expect(componentIndex.getByText('Queue', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Candidate-only queue body.', { exact: true })).toHaveCount(0)
    const reviewContext = page.getByLabel('Review context')
    await expect(reviewContext.getByRole('heading', { name: 'Gateway' })).toBeVisible()
    await expect(reviewContext.getByText('Accepted gateway body.', { exact: true })).toBeVisible()
    await expect(reviewContext).not.toContainText('Queue')
    const beforeGateway = componentIndex.getByRole('button', { name: 'Gateway, Content changed' })
    await expect(beforeGateway).toBeVisible()
    await beforeGateway.click()
    await expect(reviewContext.getByText('Accepted gateway body.', { exact: true })).toBeVisible()
    await expect(reviewContext.getByText('Outgoing relationships (2)')).toBeVisible()

    await page.getByRole('button', { name: 'With changes' }).click()
    await page.getByRole('button', { name: /Removed relationship: Gateway — calls — Worker/ }).click()
    await expect(reviewContext).toContainText('Removed relationship')
    await expect(page.getByTestId('raw-diff')).toContainText('Changed gateway body.')
    const focusedPath = await page.evaluate(() => document.activeElement?.getAttribute('data-diff-path'))
    expect(focusedPath).toMatch(/^components\/.+\.md$/)
    const reviewPane = page.locator('.review-workspace-pane')
    const reviewedBase = await detailValue(reviewPane, 'Base revision')
    const reviewedTree = await detailValue(reviewPane, 'Candidate tree')
    expect(reviewedBase).toBe(baseRevision)

    await page.getByRole('button', { name: 'Update architecture' }).click()
    const acceptedRevision = await displayedRevision(page)
    expect(acceptedRevision).not.toBe(baseRevision)
    expect(gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])).toBe(acceptedRevision)
    expect(gitBare(storePath, ['show', '-s', '--format=%P', acceptedRevision])).toBe(reviewedBase)
    expect(gitBare(storePath, ['show', '-s', '--format=%T', acceptedRevision])).toBe(reviewedTree)

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'restart.log')
    await openProject(page, application.origin, sourceRoot)
    expect(await displayedRevision(page)).toBe(acceptedRevision)
    const restartedIndex = page.getByRole('navigation', { name: 'Components' })
    await expect(restartedIndex.getByRole('button', { name: 'Queue', exact: true })).toBeVisible()
    await restartedIndex.getByRole('button', { name: 'Docs', exact: true }).click()
    await expect(page.getByText('Changed docs body.', { exact: true })).toBeVisible()
    await restartedIndex.getByRole('button', { name: 'Gateway', exact: true }).click()
    await expect(page.getByText('Changed gateway body.', { exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Edit component' }).click()
    const restartedRelationships = page.getByRole('group', { name: 'Outgoing relationships' })
    await expect(restartedRelationships.getByLabel('Label').nth(0)).toHaveValue('calls')
    await expect(restartedRelationships.getByLabel('Label').nth(1)).toHaveValue('calls')
    await expect(restartedRelationships.getByLabel('Target').nth(0).locator('option:checked')).toHaveText('Worker')
    await expect(restartedRelationships.getByLabel('Target').nth(1).locator('option:checked')).toHaveText('Queue')

    await stopWorkBraid(application)
    application = undefined
    expect(sourceEvidence(sourceRoot)).toEqual(sourceBefore)
    expect(sqlite(dataRoot, "SELECT group_concat(name, ',') FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%';")).toBe('source_architecture_associations')
    expect(sqlite(dataRoot, 'SELECT count(*) FROM source_architecture_associations;')).toBe('1')
  } finally {
    if (application) await stopWorkBraid(application)
    rmSync(runtimeRoot, { recursive: true, force: true })
  }
})

async function openProject(page: Page, origin: string, sourceRoot: string) {
  await page.goto(origin)
  await expect(page.getByRole('heading', { name: 'Open a project' })).toBeVisible()
  await page.getByLabel('Project folder').fill(sourceRoot)
  await page.getByRole('button', { name: 'Open', exact: true }).click()
}

async function addComponent(page: Page, title: string, description: string) {
  await page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Add component' }).click()
  await page.getByLabel('Title').fill(title)
  await page.getByLabel('Description').fill(description)
  await keepChange(page)
}

async function editPending(page: Page, title: string) {
  const item = page.locator('.changes-in-progress li').filter({ hasText: title })
  await item.getByRole('button', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit component' })).toBeVisible()
}

async function editAccepted(page: Page, title: string) {
  await page.getByRole('navigation', { name: 'Components' }).getByRole('button', { name: title, exact: true }).click()
  await page.getByRole('button', { name: 'Edit component' }).click()
  await expect(page.getByRole('heading', { name: 'Edit component' })).toBeVisible()
}

async function addRelationship(page: Page, targetLabel: string, label: string) {
  const relationships = page.getByRole('group', { name: 'Outgoing relationships' })
  await relationships.getByRole('button', { name: 'Add relationship' }).click()
  const target = relationships.getByLabel('Target').last()
  await target.selectOption({ label: targetLabel })
  if (label) await relationships.getByLabel('Label').last().fill(label)
}

async function keepChange(page: Page) {
  await page.getByRole('button', { name: 'Keep change' }).click()
  await expect(page.getByRole('heading', { name: 'Changes in progress' })).toBeVisible()
}

async function displayedRevision(page: Page) {
  const details = page.locator('details.technical-details')
  if (!(await details.getAttribute('open'))) await details.locator('summary').click()
  return detailValue(details, 'Revision')
}

async function detailValue(scope: Locator, label: string) {
  const value = scope.locator('dt', { hasText: label }).locator('xpath=following-sibling::dd[1]')
  return (await value.textContent())?.trim() ?? ''
}

function createSourceRepository(sourceRoot: string) {
  mkdirSync(sourceRoot, { recursive: true })
  git(sourceRoot, ['init', '--quiet'])
  writeFileSync(join(sourceRoot, 'README.md'), '# Throwaway project\n')
  writeFileSync(join(sourceRoot, 'settings.txt'), 'mode=phase1-review\n')
  git(sourceRoot, ['add', 'README.md', 'settings.txt'])
  git(sourceRoot, ['-c', 'user.name=WorkBraid Phase 1', '-c', 'user.email=phase1@workbraid.invalid', 'commit', '--quiet', '-m', 'source fixture'])
  writeFileSync(join(sourceRoot, 'local-note.txt'), 'untracked and unchanged\n')
}

function sourceEvidence(sourceRoot: string): SourceEvidence {
  return {
    head: git(sourceRoot, ['rev-parse', 'HEAD']),
    status: git(sourceRoot, ['status', '--short', '--untracked-files=all']),
    index: git(sourceRoot, ['ls-files', '--stage']),
    files: sourceFileEvidence(sourceRoot),
  }
}

function sourceFileEvidence(sourceRoot: string) {
  const evidence: string[] = []
  const visit = (directory: string) => {
    for (const name of readdirSync(directory).sort()) {
      if (directory === sourceRoot && name === '.git') continue
      const path = join(directory, name)
      const stat = lstatSync(path)
      const relativePath = relative(sourceRoot, path)
      if (stat.isDirectory()) {
        evidence.push(`dir ${stat.mode & 0o777} ${relativePath}`)
        visit(path)
      } else {
        const digest = createHash('sha256').update(readFileSync(path)).digest('hex')
        evidence.push(`file ${stat.mode & 0o777} ${digest} ${relativePath}`)
      }
    }
  }
  visit(sourceRoot)
  return evidence.join('\n')
}

function onlyArchitectureStore(dataRoot: string) {
  const storeRoot = join(dataRoot, 'architecture')
  const entries = readdirSync(storeRoot).filter((name) => name.endsWith('.git'))
  expect(entries).toHaveLength(1)
  return join(storeRoot, entries[0])
}

function git(directory: string, arguments_: string[]) {
  return run('git', ['-C', directory, ...arguments_], repositoryRoot)
}

function gitBare(storePath: string, arguments_: string[]) {
  return run('git', ['--git-dir', storePath, ...arguments_], repositoryRoot)
}

function sqlite(dataRoot: string, query: string) {
  return run('sqlite3', ['-batch', join(dataRoot, 'workbraid.db'), query], repositoryRoot)
}

function run(command: string, arguments_: string[], cwd: string) {
  const result = spawnSync(command, arguments_, {
    cwd,
    encoding: 'utf8',
    env: { ...process.env, GIT_CONFIG_NOSYSTEM: '1', GIT_TERMINAL_PROMPT: '0' },
  })
  if (result.status !== 0) {
    throw new Error(`${command} ${arguments_.join(' ')} failed (${result.status}):\n${result.stdout}\n${result.stderr}`)
  }
  return result.stdout.trim()
}

async function unusedLoopbackPort() {
  const server = createServer()
  await new Promise<void>((resolvePromise, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolvePromise)
  })
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('could not allocate a loopback port')
  await new Promise<void>((resolvePromise, reject) => server.close((error) => error ? reject(error) : resolvePromise()))
  return address.port
}

async function startWorkBraid(binary: string, dataRoot: string, port: number, runtimeRoot: string, logName = 'workbraid.log'): Promise<RunningWorkBraid> {
  const origin = `http://127.0.0.1:${port}`
  const logFD = openSync(join(runtimeRoot, logName), 'a')
  const child = spawn(binary, [
    '-listen', `127.0.0.1:${port}`,
    '-data-dir', dataRoot,
    '-ui-dir', join(frontendRoot, 'dist'),
  ], {
    cwd: repositoryRoot,
    detached: true,
    stdio: ['ignore', logFD, logFD],
  })
  try {
    await waitForServer(origin, child)
    return { child, origin, logFD }
  } catch (error) {
    await stopWorkBraid({ child, origin, logFD })
    throw error
  }
}

async function waitForServer(origin: string, child: ChildProcess) {
  const deadline = Date.now() + 15_000
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`WorkBraid exited before becoming ready (${child.exitCode})`)
    try {
      const response = await fetch(origin)
      if (response.ok) return
    } catch {
      // The process has not bound its loopback socket yet.
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 50))
  }
  throw new Error('WorkBraid did not become ready within 15 seconds')
}

async function stopWorkBraid(application: RunningWorkBraid) {
  const pid = application.child.pid
  if (pid && !childExited(application.child)) {
    try {
      process.kill(-pid, 'SIGTERM')
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error
    }
    await waitForChildExit(application.child, 3_000)
    if (!childExited(application.child)) {
      try {
        process.kill(-pid, 'SIGKILL')
      } catch (error) {
        if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error
      }
      await waitForChildExit(application.child, 3_000)
      if (!childExited(application.child)) throw new Error(`WorkBraid process group ${pid} did not terminate`)
    }
  }
  closeSync(application.logFD)
}

function childExited(child: ChildProcess) {
  return child.exitCode !== null || child.signalCode !== null
}

async function waitForChildExit(child: ChildProcess, timeout: number) {
  const deadline = Date.now() + timeout
  while (!childExited(child) && Date.now() < deadline) {
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 25))
  }
}
