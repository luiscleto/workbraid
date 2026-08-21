import { expect, test, type Page } from '@playwright/test'
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

test('Gate 1 production path creates, accepts, and reconstructs Architecture without touching the source repository', async ({ page }) => {
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-gate-smoke-'))
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
    await expect(page.getByRole('heading', { name: 'Not linked' })).toBeVisible()
    await page.getByRole('button', { name: 'Set up architecture' }).click()
    await expect(page.getByRole('heading', { name: 'Set up architecture?' })).toBeVisible()
    await page.getByRole('button', { name: 'Set up', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Components', exact: true })).toBeVisible()
    await expect(page.getByText('No components', { exact: true })).toBeVisible()

    const storePath = onlyArchitectureStore(dataRoot)
    const bootstrapRevision = gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])
    expect(await displayedRevision(page)).toBe(bootstrapRevision)
    expect(gitBare(storePath, ['ls-tree', bootstrapRevision])).toMatch(/^100644 blob [0-9a-f]{40}\tarchitecture\.yaml\n040000 tree [0-9a-f]{40}\tdiagrams$/)

    await addComponent(page, 'Gateway', [
      'Routes requests to the rest of the system.',
      '',
      '| Concern | Owner |',
      '| --- | --- |',
      '| ingress | Gateway |',
      '',
      '- [x] validates requests',
      '',
      '```text',
      'request -> route',
      '```',
    ].join('\n'))
    await addComponent(page, 'Records', 'Stores durable records.\n')
    await addComponent(page, 'Worker', 'Processes queued work.\n')

    const componentIndex = page.getByRole('navigation', { name: 'Diagrams and components' })
    await expect(componentIndex.getByText('No components', { exact: true })).toBeVisible()
    await expect(componentIndex.getByText('Gateway', { exact: true })).toHaveCount(0)
    await expect(page.getByText('This diagram has no components.')).toBeVisible()

    await openPendingEditor(page, 'Gateway')
    await page.getByRole('button', { name: 'Add relationship' }).click()
    const target = page.getByLabel('Target')
    await target.selectOption({ label: 'Records — New component' })
    await page.getByLabel('Label').fill('calls')
    await page.getByRole('button', { name: 'Keep change' }).click()

    await expect(componentIndex.getByText('No components', { exact: true })).toBeVisible()
    await expect(page.getByText('This diagram has no components.')).toBeVisible()
    await page.getByRole('button', { name: 'Review changes' }).click()
    const review = page.locator('.review-workspace-pane')
    await expect(review).toBeVisible()
    const rawDiff = review.getByTestId('raw-diff')
    await expect(rawDiff).toContainText('# Gateway')
    await expect(rawDiff).toContainText('target:')
    await expect(rawDiff).toContainText('label: "calls"')
    const reviewedBase = await detailValue(review, 'Base revision')
    const reviewedTree = await detailValue(review, 'Candidate tree')
    expect(reviewedBase).toBe(bootstrapRevision)

    await page.getByRole('button', { name: 'Update architecture' }).click()
    await expect(page.getByRole('button', { name: /Changes in progress/ })).toHaveCount(0)
    await expect(componentIndex.getByText('Gateway', { exact: true })).toBeVisible()
    await expect(componentIndex.getByText('Records', { exact: true })).toBeVisible()
    await expect(componentIndex.getByText('Worker', { exact: true })).toBeVisible()
    await expect(page.getByTestId('architecture-map')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Fit map' })).toBeVisible()

    const acceptedRevision = await displayedRevision(page)
    expect(acceptedRevision).not.toBe(bootstrapRevision)
    expect(gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])).toBe(acceptedRevision)
    expect(gitBare(storePath, ['show', '-s', '--format=%T', acceptedRevision])).toBe(reviewedTree)
    expect(gitBare(storePath, ['show', '-s', '--format=%P', acceptedRevision])).toBe(reviewedBase)

    await componentIndex.getByText('Gateway', { exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Gateway' })).toBeVisible()
    await expect(page.getByRole('table')).toBeVisible()
    await page.getByRole('button', { name: 'Edit component' }).click()
    await expect(page.getByLabel('Target')).toHaveValue(/.+/)
    await expect(page.getByLabel('Target').locator('option:checked')).toHaveText('Records')
    await expect(page.getByLabel('Label')).toHaveValue('calls')
    await page.getByRole('button', { name: 'Cancel' }).click()

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'restart.log')
    await openProject(page, application.origin, sourceRoot)

    expect(await displayedRevision(page)).toBe(acceptedRevision)
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByText('Gateway', { exact: true })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByText('Records', { exact: true })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByText('Worker', { exact: true })).toBeVisible()
    await expect(page.getByTestId('architecture-map')).toBeVisible()
    await page.getByRole('navigation', { name: 'Diagrams and components' }).getByText('Gateway', { exact: true }).click()
    await expect(page.getByRole('table')).toBeVisible()
    await page.getByRole('button', { name: 'Edit component' }).click()
    await expect(page.getByLabel('Target').locator('option:checked')).toHaveText('Records')
    await expect(page.getByLabel('Label')).toHaveValue('calls')

    expect(sourceEvidence(sourceRoot)).toEqual(sourceBefore)
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
  await page.getByRole('button', { name: 'Add component' }).click()
  await page.getByLabel('Title').fill(title)
  await page.getByLabel('Description').fill(description)
  await page.getByRole('button', { name: 'Keep change' }).click()
  await expect(page.getByRole('heading', { name: 'Changes in progress' })).toBeVisible()
}

async function openPendingEditor(page: Page, title: string) {
  const item = page.locator('.changes-in-progress li').filter({ hasText: title })
  await item.getByRole('button', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit component' })).toBeVisible()
}

async function displayedRevision(page: Page) {
  const details = page.locator('details.technical-details')
  await details.locator('summary').click()
  return detailValue(details, 'Revision')
}

async function detailValue(scope: ReturnType<Page['locator']>, label: string) {
  const value = scope.locator('dt', { hasText: label }).locator('xpath=following-sibling::dd[1]')
  return (await value.textContent())?.trim() ?? ''
}

function createSourceRepository(sourceRoot: string) {
  mkdirSync(sourceRoot, { recursive: true })
  git(sourceRoot, ['init', '--quiet'])
  writeFileSync(join(sourceRoot, 'README.md'), '# Throwaway project\n')
  writeFileSync(join(sourceRoot, 'settings.txt'), 'mode=gate\n')
  git(sourceRoot, ['add', 'README.md', 'settings.txt'])
  git(sourceRoot, ['-c', 'user.name=WorkBraid Gate', '-c', 'user.email=gate@workbraid.invalid', 'commit', '--quiet', '-m', 'source fixture'])
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
