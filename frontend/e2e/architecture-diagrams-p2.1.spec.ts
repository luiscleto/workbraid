import { expect, test, type Locator, type Page } from '@playwright/test'
import { createHash } from 'node:crypto'
import { spawn, spawnSync, type ChildProcess } from 'node:child_process'
import { basename, dirname, join, relative, resolve } from 'node:path'
import { closeSync, lstatSync, mkdirSync, mkdtempSync, openSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { fileURLToPath } from 'node:url'

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repositoryRoot = resolve(frontendRoot, '..')
const rootID = '11111111-1111-4111-8111-111111111111'
const detailID = '22222222-2222-4222-8222-222222222222'
const emptyID = '33333333-3333-4333-8333-333333333333'
const gatewayID = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const workerID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
const recordsID = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'

type RunningWorkBraid = { child: ChildProcess; origin: string; logFD: number }
type SourceEvidence = { head: string; status: string; index: string; files: string }

test('P2.1 Diagram hierarchy remains navigable and writable on the living v2 product', async ({ page }) => {
  page.setDefaultTimeout(10_000)
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-p21-diagrams-'))
  const sourceRoot = join(runtimeRoot, 'source-project')
  const dataRoot = join(runtimeRoot, 'app-data')
  const binary = join(runtimeRoot, 'workbraid')
  const forbiddenRequests: string[] = []
  page.on('request', (request) => {
    if (request.url().includes('forbidden.workbraid.invalid')) forbiddenRequests.push(request.url())
  })
  let application: RunningWorkBraid | undefined

  try {
    createSourceRepository(sourceRoot)
    mkdirSync(dataRoot, { recursive: true })
    const sourceBefore = sourceEvidence(sourceRoot)
    run('go', ['build', '-buildvcs=false', '-o', binary, './cmd/workbraid'], repositoryRoot)
    const port = await unusedLoopbackPort()

    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'bootstrap.log')
    await openProject(page, application.origin, sourceRoot)
    await page.getByRole('button', { name: 'Set up architecture' }).click()
    await page.getByRole('button', { name: 'Set up', exact: true }).click()
    const bootstrapRevision = await displayedRevision(page)
    const storePath = onlyArchitectureStore(dataRoot)
    const storeID = basename(storePath, '.git')
    await stopWorkBraid(application)
    application = undefined

    const firstRevision = writeAcceptedV2(storePath, bootstrapRevision, bootstrapRevision, storeID, sourceRoot, 'A')
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'accepted-v2.log')
    await openProject(page, application.origin, sourceRoot)
    expect(await displayedRevision(page)).toBe(firstRevision)

    await expect(page.getByText('View only', { exact: true })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Add component' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit component' })).toBeVisible()
    const navigator = page.getByRole('navigation', { name: 'Diagrams and components' })
    await expect(navigator.getByRole('button', { name: 'System A' })).toHaveAttribute('aria-current', 'page')
    await expect(navigator.getByRole('button', { name: 'Detail, Inside Shared — gateway.md' })).toBeVisible()
    await expect(navigator.getByRole('button', { name: 'Detail, Inside Shared — records.md' })).toBeVisible()
    await expect(navigator.getByRole('button', { name: 'Shared, gateway.md' })).toBeVisible()
    await expect(navigator.getByRole('button', { name: 'Shared, records.md' })).toBeVisible()
    await expect(navigator.getByText('Included here · Lives in Detail')).toBeVisible()
    await expect(navigator.getByRole('button', { name: 'Worker, Included here · Lives in Detail' })).toBeVisible()
    await expect(page.getByText('Gateway documentation A.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Fit map' })).toBeVisible()

    await navigator.getByRole('button', { name: 'Worker' }).click()
    await expect(page.getByText('Worker documentation A.')).toBeVisible()
    await navigator.getByRole('button', { name: 'Shared, gateway.md' }).click()
    await page.getByRole('button', { name: 'Open Detail' }).click()
    await expect(navigator.getByRole('button', { name: 'Detail, Inside Shared — gateway.md' })).toHaveAttribute('aria-current', 'page')
    const breadcrumbs = page.getByRole('navigation', { name: 'Diagram breadcrumbs' })
    await expect(breadcrumbs.getByRole('button', { name: 'System A' })).toBeVisible()
    await expect(breadcrumbs.getByText('Detail')).toHaveAttribute('aria-current', 'page')
    await expect(page.getByText('Worker documentation A.')).toBeVisible()

    const mapDock = page.locator('.map-bottom-dock')
    await expect(mapDock).toBeVisible()
    await expect(page.getByRole('tablist', { name: 'Map information' })).toHaveCount(0)
    const external = page.getByRole('navigation', { name: 'Components that live elsewhere' })
    const recordsBoundary = external.getByRole('button', { name: 'Shared, records.md, Lives in System A' })
    await expect(recordsBoundary).toBeVisible()
    const recordsRelationships = external.getByRole('list', { name: 'Relationships for Shared, records.md' })
    await expect(recordsRelationships.getByRole('listitem')).toHaveCount(3)
    await expect(recordsRelationships.getByText('writes', { exact: true })).toHaveCount(2)
    await expect(recordsRelationships.getByText('feeds', { exact: true })).toHaveCount(1)
    await recordsRelationships.getByRole('listitem').first().focus()
    await expect(recordsRelationships.getByRole('listitem').first()).toBeFocused()

    await page.getByRole('button', { name: 'Edit component' }).click()
    await page.getByLabel('Description').fill('Worker documentation A with a review-only change.\n')
    await page.getByRole('button', { name: 'Keep change' }).click()
    await page.getByRole('button', { name: 'Review changes' }).click()
    const reviewControls = page.locator('.map-review-controls')
    const reviewMapRegion = page.locator('.map-region')
    await navigator.getByRole('button', { name: 'Detail, Inside Shared — records.md' }).click()
    await expect(page.getByRole('tablist', { name: 'Map information' })).toHaveCount(0)
    await expect(page.getByRole('region', { name: 'Changes', exact: true })).toContainText('No visual changes in this diagram.')
    await navigator.getByRole('button', { name: 'Detail, Inside Shared — gateway.md' }).click()
    const dockTabs = page.getByRole('tablist', { name: 'Map information' })
    const changesTab = dockTabs.getByRole('tab', { name: 'Changes' })
    const externalTab = dockTabs.getByRole('tab', { name: 'External references' })
    await expect(changesTab).toHaveAttribute('aria-selected', 'true')
    await expect(reviewControls).toBeVisible()
    await expect(external).toHaveCount(0)
    let dockBox = await mapDock.boundingBox()
    let mapBox = await reviewMapRegion.boundingBox()
    expect(dockBox).not.toBeNull()
    expect(mapBox).not.toBeNull()
    expect(Math.abs(dockBox!.x - mapBox!.x - 14)).toBeLessThanOrEqual(1)
    expect(Math.abs(mapBox!.width - dockBox!.width - 28)).toBeLessThanOrEqual(1)
    expect(Math.abs((mapBox!.y + mapBox!.height) - (dockBox!.y + dockBox!.height) - 14)).toBeLessThanOrEqual(1)

    await externalTab.click()
    await expect(externalTab).toHaveAttribute('aria-selected', 'true')
    await expect(external).toBeVisible()
    await expect(reviewControls).toHaveCount(0)
    const expandedDockHeight = (await mapDock.boundingBox())!.height
    await page.getByRole('button', { name: 'Collapse' }).click()
    await expect(page.locator('.map-bottom-dock-body')).toHaveCount(0)
    expect((await mapDock.boundingBox())!.height).toBeLessThan(expandedDockHeight)
    await page.getByRole('button', { name: 'Expand' }).click()
    await expect(external).toBeVisible()

    await page.setViewportSize({ width: 720, height: 720 })
    await expect(mapDock).toBeVisible()
    dockBox = await mapDock.boundingBox()
    mapBox = await reviewMapRegion.boundingBox()
    expect(dockBox).not.toBeNull()
    expect(mapBox).not.toBeNull()
    expect(Math.abs(dockBox!.x - mapBox!.x - 14)).toBeLessThanOrEqual(1)
    expect(Math.abs(mapBox!.width - dockBox!.width - 28)).toBeLessThanOrEqual(1)
    expect(Math.abs((mapBox!.y + mapBox!.height) - (dockBox!.y + dockBox!.height) - 14)).toBeLessThanOrEqual(1)
    await changesTab.click()
    await expect(reviewControls).toBeVisible()
    await expect(external).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Fit map' })).toBeVisible()
    await page.setViewportSize({ width: 1280, height: 720 })

    await page.getByRole('button', { name: 'Discard changes' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Discard changes' }).click()
    await expect(page.getByRole('button', { name: /Changes in progress/ })).toHaveCount(0)
    await expect(mapDock).toBeVisible()
    await expect(page.getByRole('tablist', { name: 'Map information' })).toHaveCount(0)
    await expect(external).toBeVisible()
    await recordsBoundary.click()
    await expect(navigator.getByRole('button', { name: 'System A' })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByText('Records documentation A.')).toBeVisible()

    await navigator.getByRole('button', { name: 'Detail, Inside Shared — records.md' }).click()
    await expect(page.getByRole('heading', { name: 'No components here' })).toBeVisible()
    await expect(page.getByText('This diagram has no components.')).toBeVisible()
    await expect(page.getByText('The architecture has no components yet.')).toHaveCount(0)
    await page.getByRole('navigation', { name: 'Diagram breadcrumbs' }).getByRole('button', { name: 'System A' }).click()
    await expect(page.getByText('Records documentation A.')).toBeVisible()
    await page.getByRole('button', { name: 'Clear selection' }).click()
    await expect(page.getByRole('heading', { name: 'Select a component' })).toBeVisible()
    expect(forbiddenRequests).toEqual([])

    const secondRevision = writeAcceptedV2(storePath, bootstrapRevision, firstRevision, storeID, sourceRoot, 'B')
    expect(gitBare(storePath, ['show', '-s', '--format=%P', secondRevision])).toBe(bootstrapRevision)
    await expect(navigator.getByRole('button', { name: 'System A' })).toBeVisible()
    await expect(page.getByText('Worker documentation B.')).toHaveCount(0)
    await page.getByRole('button', { name: 'Refresh' }).click()
    await expect(navigator.getByRole('button', { name: 'System B' })).toHaveAttribute('aria-current', 'page')
    await navigator.getByRole('button', { name: 'Worker' }).click()
    await expect(page.getByText('Worker documentation B.')).toBeVisible()
    expect(await displayedRevision(page)).toBe(secondRevision)

    await page.getByRole('button', { name: 'Open another project' }).click()
    await expect(page.getByRole('heading', { name: 'Open a project' })).toBeVisible()
    await openProject(page, application.origin, sourceRoot, false)
    expect(await displayedRevision(page)).toBe(secondRevision)

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'restart.log')
    await openProject(page, application.origin, sourceRoot)
    expect(await displayedRevision(page)).toBe(secondRevision)
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'System B' })).toHaveAttribute('aria-current', 'page')
    await page.getByRole('navigation', { name: 'Diagrams and components' }).getByRole('button', { name: 'Detail, Inside Shared — gateway.md' }).click()
    await expect(page.getByRole('navigation', { name: 'Components that live elsewhere' })).toBeVisible()

    await stopWorkBraid(application)
    application = undefined
    expect(sourceEvidence(sourceRoot)).toEqual(sourceBefore)
    expect(gitBare(storePath, ['rev-parse', 'refs/heads/accepted'])).toBe(secondRevision)
    expect(sqlite(dataRoot, "SELECT group_concat(name, ',') FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%';")).toBe('source_architecture_associations')
    expect(sqlite(dataRoot, 'SELECT count(*) FROM source_architecture_associations;')).toBe('1')
  } finally {
    if (application) await stopWorkBraid(application)
    rmSync(runtimeRoot, { recursive: true, force: true })
  }
})

test('P2.2 deliberately sets up diagrams for one readable legacy Architecture', async ({ page }) => {
  page.setDefaultTimeout(10_000)
  const runtimeRoot = mkdtempSync(join(tmpdir(), 'workbraid-p22-legacy-'))
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
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'legacy-bootstrap.log')
    await openProject(page, application.origin, sourceRoot)
    await page.getByRole('button', { name: 'Set up architecture' }).click()
    await page.getByRole('button', { name: 'Set up', exact: true }).click()
    const bootstrap = await displayedRevision(page)
    const storePath = onlyArchitectureStore(dataRoot)
    const storeID = basename(storePath, '.git')
    await stopWorkBraid(application)
    application = undefined

    const legacy = writeAcceptedV1(storePath, bootstrap, storeID, sourceRoot)
    const gatewayEntry = gitBare(storePath, ['ls-tree', legacy, 'components/gateway.md'])
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'legacy-open.log')
    await openProject(page, application.origin, sourceRoot)
    expect(await displayedRevision(page)).toBe(legacy)
    await expect(page.getByText('Gateway legacy documentation.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit component' })).toHaveCount(0)
    await page.getByRole('button', { name: 'Set up diagrams' }).click()
    await expect(page.getByRole('heading', { name: 'Changes in progress' })).toBeVisible()
    await expect(page.getByText('Setting up diagrams will make this architecture editable.')).toBeVisible()
    await page.getByRole('button', { name: 'Review changes' }).click()
    const review = page.locator('.review-workspace-pane')
    await expect(review.getByText('Diagram changes')).toBeVisible()
    await expect(review.getByText('Gateway placed in diagram')).toBeVisible()
    await review.getByRole('button', { name: 'Before changes' }).click()
    await expect(page.getByText('That diagram exists only with the changes. Before changes shows the earlier architecture map.')).toBeVisible()
    await expect(review.getByTestId('raw-diff')).toContainText('version: 2')
    const candidateTree = await detailValue(review, 'Candidate tree')
    expect(gitBare(storePath, ['ls-tree', candidateTree, 'components/gateway.md'])).toBe(gatewayEntry)
    await page.getByRole('button', { name: 'Update architecture' }).click()
    const accepted = await displayedRevision(page)
    expect(accepted).not.toBe(legacy)
    await expect(page.getByRole('button', { name: 'Add component' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit component' })).toBeVisible()

    await stopWorkBraid(application)
    application = undefined
    application = await startWorkBraid(binary, dataRoot, port, runtimeRoot, 'legacy-restart.log')
    await openProject(page, application.origin, sourceRoot)
    expect(await displayedRevision(page)).toBe(accepted)
    await expect(page.getByRole('navigation', { name: 'Diagrams and components' })).toBeVisible()
    await expect(page.getByText('Gateway legacy documentation.')).toBeVisible()
    expect(sourceEvidence(sourceRoot)).toEqual(sourceBefore)
    expect(sqlite(dataRoot, "SELECT group_concat(name, ',') FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%';")).toBe('source_architecture_associations')
  } finally {
    if (application) await stopWorkBraid(application)
    rmSync(runtimeRoot, { recursive: true, force: true })
  }
})

function writeAcceptedV1(storePath: string, parent: string, storeID: string, sourceRoot: string) {
  const gateway = `---\nid: "${gatewayID}"\nrelationships:\n  - target: "${workerID}"\n    label: calls\n---\n# Gateway\nGateway legacy documentation.\n`
  const worker = `---\nid: "${workerID}"\n---\n# Worker\nWorker legacy documentation.\n`
  const componentTree = gitBareInput(storePath, ['mktree'], [
    `100755 blob ${gitBareInput(storePath, ['hash-object', '-w', '--stdin'], gateway)}\tgateway.md`,
    `100644 blob ${gitBareInput(storePath, ['hash-object', '-w', '--stdin'], worker)}\tworker.md`,
  ].join('\n') + '\n')
  const manifest = `format: workbraid-architecture\nversion: 1\nstore_id: "${storeID}"\nproject:\n  name: "Source project"\n  source_hint: "${sourceRoot}"\n`
  const manifestBlob = gitBareInput(storePath, ['hash-object', '-w', '--stdin'], manifest)
  const tree = gitBareInput(storePath, ['mktree'], `100644 blob ${manifestBlob}\tarchitecture.yaml\n040000 tree ${componentTree}\tcomponents\n`)
  const commit = gitBareInput(storePath, ['-c', 'user.name=WorkBraid P2.2', '-c', 'user.email=p22@workbraid.invalid', 'commit-tree', tree, '-p', parent], 'legacy fixture\n')
  gitBare(storePath, ['update-ref', 'refs/heads/accepted', commit, parent])
  return commit
}

function writeAcceptedV2(storePath: string, parent: string, expected: string, storeID: string, sourceRoot: string, suffix: string) {
  const componentSources = {
    'gateway.md': `---\nid: "${gatewayID}"\nrelationships:\n  - target: "${workerID}"\n    label: calls\n---\n# Shared\nGateway documentation ${suffix}.\n\n![remote](https://forbidden.workbraid.invalid/image.png)\n\n<img src="https://forbidden.workbraid.invalid/raw.png">\n`,
    'records.md': `---\nid: "${recordsID}"\nrelationships:\n  - target: "${workerID}"\n    label: feeds\n---\n# Shared\nRecords documentation ${suffix}.\n`,
    'worker.md': `---\nid: "${workerID}"\nrelationships:\n  - target: "${recordsID}"\n    label: writes\n  - target: "${recordsID}"\n    label: writes\n---\n# Worker\nWorker documentation ${suffix}.\n`,
  }
  const componentEntries = Object.entries(componentSources).map(([name, source]) => `${name === 'worker.md' ? '100755' : '100644'} blob ${gitBareInput(storePath, ['hash-object', '-w', '--stdin'], source)}\t${name}`)
  const componentTree = gitBareInput(storePath, ['mktree'], componentEntries.join('\n') + '\n')
  const diagramSources = {
    'detail.yaml': `id: "${detailID}"\ntitle: "Detail"\nappearances:\n  - component: "${workerID}"\n    role: home\n`,
    'empty.yaml': `id: "${emptyID}"\ntitle: "Detail"\nappearances: []\n`,
    'root.yaml': `id: "${rootID}"\ntitle: "System ${suffix}"\nappearances:\n  - component: "${gatewayID}"\n    role: home\n    detail_diagram: "${detailID}"\n  - component: "${workerID}"\n    role: reference\n  - component: "${recordsID}"\n    role: home\n    detail_diagram: "${emptyID}"\n`,
  }
  const diagramEntries = Object.entries(diagramSources).map(([name, source]) => `100644 blob ${gitBareInput(storePath, ['hash-object', '-w', '--stdin'], source)}\t${name}`)
  const diagramTree = gitBareInput(storePath, ['mktree'], diagramEntries.join('\n') + '\n')
  const manifest = `format: workbraid-architecture\nversion: 2\nstore_id: "${storeID}"\nproject:\n  name: "Source project"\n  source_hint: "${sourceRoot}"\nroot_diagram: "${rootID}"\n`
  const manifestBlob = gitBareInput(storePath, ['hash-object', '-w', '--stdin'], manifest)
  const rootTree = gitBareInput(storePath, ['mktree'], `100644 blob ${manifestBlob}\tarchitecture.yaml\n040000 tree ${componentTree}\tcomponents\n040000 tree ${diagramTree}\tdiagrams\n`)
  const commit = gitBareInput(storePath, ['-c', 'user.name=WorkBraid P2.1', '-c', 'user.email=p21@workbraid.invalid', 'commit-tree', rootTree, '-p', parent], `accepted v2 ${suffix}\n`)
  gitBare(storePath, ['update-ref', 'refs/heads/accepted', commit, expected])
  return commit
}

async function openProject(page: Page, origin: string, sourceRoot: string, navigate = true) {
  if (navigate) await page.goto(origin)
  await expect(page.getByRole('heading', { name: 'Open a project' })).toBeVisible()
  await page.getByLabel('Project folder').fill(sourceRoot)
  await page.getByRole('button', { name: 'Open', exact: true }).click()
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
  writeFileSync(join(sourceRoot, 'settings.txt'), 'mode=p2.1\n')
  git(sourceRoot, ['add', 'README.md', 'settings.txt'])
  git(sourceRoot, ['-c', 'user.name=WorkBraid P2.1', '-c', 'user.email=p21@workbraid.invalid', 'commit', '--quiet', '-m', 'source fixture'])
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
  const stores = readdirSync(join(dataRoot, 'architecture')).filter((name) => name.endsWith('.git'))
  expect(stores).toHaveLength(1)
  return join(dataRoot, 'architecture', stores[0])
}

function git(directory: string, args: string[]) { return run('git', ['-C', directory, ...args], repositoryRoot) }
function gitBare(storePath: string, args: string[]) { return run('git', ['--git-dir', storePath, ...args], repositoryRoot) }
function gitBareInput(storePath: string, args: string[], input: string) { return run('git', ['--git-dir', storePath, ...args], repositoryRoot, input) }
function sqlite(dataRoot: string, query: string) { return run('sqlite3', ['-batch', join(dataRoot, 'workbraid.db'), query], repositoryRoot) }

function run(command: string, args: string[], cwd: string, input?: string) {
  const result = spawnSync(command, args, { cwd, input, encoding: 'utf8', env: { ...process.env, GIT_CONFIG_NOSYSTEM: '1', GIT_TERMINAL_PROMPT: '0' } })
  if (result.status !== 0) throw new Error(`${command} ${args.join(' ')} failed (${result.status}):\n${result.stdout}\n${result.stderr}`)
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

async function startWorkBraid(binary: string, dataRoot: string, port: number, runtimeRoot: string, logName: string): Promise<RunningWorkBraid> {
  const origin = `http://127.0.0.1:${port}`
  const logFD = openSync(join(runtimeRoot, logName), 'a')
  const child = spawn(binary, ['-listen', `127.0.0.1:${port}`, '-data-dir', dataRoot, '-ui-dir', join(frontendRoot, 'dist')], {
    cwd: repositoryRoot, detached: true, stdio: ['ignore', logFD, logFD],
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
      // Process has not bound its loopback socket yet.
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 50))
  }
  throw new Error('WorkBraid did not become ready within 15 seconds')
}

async function stopWorkBraid(application: RunningWorkBraid) {
  const pid = application.child.pid
  if (pid && !childExited(application.child)) {
    try { process.kill(-pid, 'SIGTERM') } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error
    }
    await waitForChildExit(application.child, 3_000)
    if (!childExited(application.child)) {
      try { process.kill(-pid, 'SIGKILL') } catch (error) {
        if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error
      }
      await waitForChildExit(application.child, 3_000)
      if (!childExited(application.child)) throw new Error(`WorkBraid process group ${pid} did not terminate`)
    }
  }
  closeSync(application.logFD)
}

function childExited(child: ChildProcess) { return child.exitCode !== null || child.signalCode !== null }
async function waitForChildExit(child: ChildProcess, timeout: number) {
  const deadline = Date.now() + timeout
  while (!childExited(child) && Date.now() < deadline) await new Promise((resolvePromise) => setTimeout(resolvePromise, 25))
}
