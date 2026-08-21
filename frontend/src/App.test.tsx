import { act, cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { App } from './App'

const graphHarness = vi.hoisted(() => ({
  calls: [] as Array<{
    elements?: unknown[]
    style?: Array<{ selector: string; style: Record<string, unknown> }>
  }>,
  nodeSelect: undefined as undefined | ((event: { target: { id: () => string } }) => void),
  edgeSelect: undefined as undefined | ((event: { target: { data: () => unknown } }) => void),
  selectedIDs: [] as string[],
  fail: false,
}))

vi.mock('cytoscape', () => ({
  default: (options: { elements?: unknown[]; style?: Array<{ selector: string; style: Record<string, unknown> }> }) => {
    if (graphHarness.fail) throw new Error('canvas unavailable')
    graphHarness.calls.push(options)
    return {
      on: (_event: string, selector: string, callback: unknown) => {
        if (selector === 'node') graphHarness.nodeSelect = callback as typeof graphHarness.nodeSelect
        if (selector === 'edge') graphHarness.edgeSelect = callback as typeof graphHarness.edgeSelect
      },
      destroy: () => undefined,
      fit: () => undefined,
      $: () => ({ unselect: () => undefined }),
      getElementById: (id: string) => ({ select: () => graphHarness.selectedIDs.push(id) }),
    }
  },
}))

const unlinkedProject = {
  source_root: '/tmp/example',
  project_name: 'example',
  known: false,
}

type TestComponent = {
  id: string
  title: string
  filename?: string
  description: string
  relationships?: { target_id: string; label: string; projection_key?: string }[]
}

function testReview({
  base,
  candidate,
  diff,
  generation = 1,
  before = [],
  withChanges = [],
  componentChanges = [],
  relationshipChanges = [],
}: {
  base: string
  candidate: string
  diff: string
  generation?: number
  before?: TestComponent[]
  withChanges?: TestComponent[]
  componentChanges?: Array<{ component_id: string; status: 'added' | 'content_changed'; path: string }>
  relationshipChanges?: Array<{ key: string; before_key?: string; source_id: string; target_id: string; label: string; status: 'added' | 'removed'; path: string; occurrence: number }>
}) {
  const snapshot = (revision: string, components: TestComponent[]) => ({
    revision,
    component_count: components.length,
    component_titles: components.map((component) => component.title),
    components: components.map((component) => ({ ...component, filename: component.filename ?? `${component.id}.md`, relationships: component.relationships ?? [] })),
  })
  return {
    diff,
    base_revision: base,
    candidate_tree: candidate,
    generation,
    before: snapshot(base, before),
    with_changes: snapshot(candidate, withChanges),
    comparison: { components: componentChanges, relationships: relationshipChanges },
  }
}

function acceptedV2(overrides: Record<string, unknown> = {}) {
  const root = '11111111-1111-4111-8111-111111111111'
  const detail = '22222222-2222-4222-8222-222222222222'
  const empty = '33333333-3333-4333-8333-333333333333'
  const gateway = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
  const worker = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
  const records = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
  const components = [
    { id: gateway, title: 'Shared', filename: 'gateway.md', description: 'Gateway documentation.\n', relationships: [{ target_id: worker, label: 'calls' }] },
    { id: worker, title: 'Worker', filename: 'worker.md', description: 'Worker documentation.\n', relationships: [{ target_id: records, label: 'writes' }, { target_id: records, label: 'writes' }] },
    { id: records, title: 'Shared', filename: 'records.md', description: 'Records documentation.\n', relationships: [{ target_id: worker, label: 'feeds' }] },
  ]
  return {
    source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '2'.repeat(40), format_version: 2,
    component_count: components.length, component_titles: components.map((component) => component.title), components,
    root_diagram_id: root,
    diagrams: [
      {
        id: root, title: 'System', depth: 0, breadcrumbs: [{ id: root, title: 'System', focus_anchor_component_id: gateway }],
        appearances: [
          { component_id: gateway, role: 'home', detail_diagram_id: detail, detail_diagram_title: 'Detail' },
          { component_id: worker, role: 'reference' },
          { component_id: records, role: 'home', detail_diagram_id: empty, detail_diagram_title: 'Detail' },
        ],
        boundaries: [],
        relationships: [
          { key: `diagram:${root}:gateway:0`, source_node_key: gateway, target_node_key: worker, source_component_id: gateway, target_component_id: worker, label: 'calls' },
          { key: `diagram:${root}:worker:0`, source_node_key: worker, target_node_key: records, source_component_id: worker, target_component_id: records, label: 'writes' },
          { key: `diagram:${root}:worker:1`, source_node_key: worker, target_node_key: records, source_component_id: worker, target_component_id: records, label: 'writes' },
          { key: `diagram:${root}:records:0`, source_node_key: records, target_node_key: worker, source_component_id: records, target_component_id: worker, label: 'feeds' },
        ],
      },
      {
        id: detail, title: 'Detail', depth: 1, context: 'Inside Shared — gateway.md', parent_diagram_id: root, parent_anchor_component_id: gateway,
        breadcrumbs: [{ id: root, title: 'System', focus_anchor_component_id: gateway }, { id: detail, title: 'Detail' }],
        appearances: [{ component_id: worker, role: 'home' }],
        boundaries: [
          { key: `boundary:${gateway}`, component_id: gateway, title: 'Shared', context: 'gateway.md', home_diagram_id: root, home_diagram_title: 'System' },
          { key: `boundary:${records}`, component_id: records, title: 'Shared', context: 'records.md', home_diagram_id: root, home_diagram_title: 'System' },
        ],
        relationships: [
          { key: `diagram:${detail}:gateway:0`, source_node_key: `boundary:${gateway}`, target_node_key: worker, source_component_id: gateway, target_component_id: worker, label: 'calls' },
          { key: `diagram:${detail}:worker:0`, source_node_key: worker, target_node_key: `boundary:${records}`, source_component_id: worker, target_component_id: records, label: 'writes' },
          { key: `diagram:${detail}:worker:1`, source_node_key: worker, target_node_key: `boundary:${records}`, source_component_id: worker, target_component_id: records, label: 'writes' },
          { key: `diagram:${detail}:records:0`, source_node_key: `boundary:${records}`, target_node_key: worker, source_component_id: records, target_component_id: worker, label: 'feeds' },
        ],
      },
      {
        id: empty, title: 'Detail', depth: 1, context: 'Inside Shared — records.md', parent_diagram_id: root, parent_anchor_component_id: records,
        breadcrumbs: [{ id: root, title: 'System', focus_anchor_component_id: records }, { id: empty, title: 'Detail' }],
        appearances: [], boundaries: [], relationships: [],
      },
    ],
    ...overrides,
  }
}

describe('App', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    graphHarness.calls.length = 0
    graphHarness.nodeSelect = undefined
    graphHarness.edgeSelect = undefined
    graphHarness.selectedIDs.length = 0
    graphHarness.fail = false
  })

  it('keeps the idle screen to one sheet without empty result chrome', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: 'Open a project' })).toBeInTheDocument()
    expect(screen.getByText('Paste the full folder path, starting with /.')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Not linked' })).not.toBeInTheDocument()
  })

  it('offers a simple derived-context confirmation only after the user chooses setup', async () => {
    const fetchMock = mockResponses([unlinkedProject])
    render(<App />)

    await submitPath('  /tmp/example  ')
    expect(await screen.findByRole('heading', { name: 'Not linked' })).toBeInTheDocument()
    expect(screen.getByText('WorkBraid has not linked this folder to architecture.')).toBeInTheDocument()
    expect(screen.getByText('/tmp/example')).toBeInTheDocument()
    expect(screen.queryByText(/store (does not|doesn't) exist/i)).not.toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Set up architecture' }))

    expect(screen.getByRole('heading', { name: 'Set up architecture?' })).toBeInTheDocument()
    expect(screen.getByText('example')).toBeInTheDocument()
    expect(screen.getByText('/tmp/example')).toBeInTheDocument()
    expect(screen.getAllByRole('textbox')).toHaveLength(1)
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.getByRole('heading', { name: 'Not linked' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(requestBody(fetchMock, 0)).toEqual({ source_root: '/tmp/example' })
    expect(screen.getByLabelText('Project folder')).toHaveValue('/tmp/example')
  })

  it('initializes explicitly and shows the empty architecture with technical details collapsed', async () => {
    const revision = 'a'.repeat(40)
    const fetchMock = mockResponses([
      unlinkedProject,
      {
        source_root: '/tmp/example',
        project_name: 'example',
        state: 'empty',
        revision,
        component_count: 0,
      },
    ])
    render(<App />)
    await submitPath('/tmp/example')

    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Set up architecture' }))
    await user.click(screen.getByRole('button', { name: 'Set up' }))

    expect(await screen.findByRole('heading', { name: 'Start with a component' })).toBeInTheDocument()
    expect(screen.getByText('The architecture has no components yet.')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Open a project' })).not.toBeInTheDocument()
    const details = screen.getByText('Technical details').closest('details')
    expect(details).not.toHaveAttribute('open')
    expect(within(details as HTMLElement).getByText(revision)).toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/projects/initialize')
    expect(requestBody(fetchMock, 1)).toEqual({ source_root: '/tmp/example' })
  })

  it('opens a known architecture immediately without presenting setup controls', async () => {
    const revision = 'b'.repeat(40)
    const fetchMock = mockResponses([
      { source_root: '/tmp/example', project_name: 'example', state: 'empty', revision, component_count: 0 },
    ])
    render(<App />)
    await submitPath('/tmp/example')

    expect(await screen.findByRole('heading', { name: 'Start with a component' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Set up architecture?' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /set up|retry|repair|reset/i })).not.toBeInTheDocument()
    expect(within(screen.getByText('Technical details').closest('details') as HTMLElement).getByText(revision)).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(requestPath(fetchMock, 0)).toBe('/api/projects/open')
  })

  it('shows a compact read-only component title inventory', async () => {
    const revision = 'e'.repeat(40)
    mockResponses([{
      source_root: '/tmp/example',
      project_name: 'example',
      state: 'ready',
      revision,
      component_count: 3,
      component_titles: ['API', 'Worker', 'API'],
      components: [
        { id: 'api-1', title: 'API', description: '\nAPI body\n' },
        { id: 'worker', title: 'Worker', description: 'Worker body\n' },
        { id: 'api-2', title: 'API', description: '' },
      ],
    }])
    render(<App />)
    await submitPath('/tmp/example')

    const index = await screen.findByRole('navigation', { name: 'Components' })
    expect(within(index).getAllByRole('listitem').map((item) => item.querySelector('span')?.textContent)).toEqual(['API', 'Worker', 'API'])
    expect(screen.queryByText('This project has an empty architecture.')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add component' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit component' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fit map' })).toBeInTheDocument()
    expect(screen.queryByText(/markdown|frontmatter|uuid|filename|relationship/i)).not.toBeInTheDocument()
    const details = screen.getByText('Technical details').closest('details')
    expect(details).not.toHaveAttribute('open')
    expect(within(details as HTMLElement).getByText(revision)).toBeInTheDocument()
    const documentation = screen.getByRole('heading', { name: 'API' }).closest('article') as HTMLElement
    expect(documentation.nextElementSibling).toBe(details)
  })

  it('navigates one accepted-v2 Diagram projection coherently while keeping authoring available', async () => {
    mockResponses([acceptedV2()])
    render(<App />)
    await submitPath('/tmp/example')

    expect(await screen.findByRole('button', { name: 'Add component' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit component' })).toBeInTheDocument()
    expect(screen.queryByText('View only')).not.toBeInTheDocument()
    const navigator = screen.getByRole('navigation', { name: 'Diagrams and components' })
    expect(within(navigator).getByRole('button', { name: 'System' })).toHaveAttribute('aria-current', 'page')
    expect(within(navigator).getByRole('button', { name: 'Shared, gateway.md' })).toBeInTheDocument()
    expect(within(navigator).getByRole('button', { name: 'Shared, records.md' })).toBeInTheDocument()
    expect(within(navigator).getByText('Also shown here')).toBeInTheDocument()
    expect(screen.getByText('Gateway documentation.')).toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Open Detail' }))
    expect(within(navigator).getByRole('button', { name: 'Detail, Inside Shared — gateway.md' })).toHaveAttribute('aria-current', 'page')
    const breadcrumbs = screen.getByRole('navigation', { name: 'Diagram breadcrumbs' })
    expect(within(breadcrumbs).getByRole('button', { name: 'System' })).toBeInTheDocument()
    expect(within(breadcrumbs).getByText('Detail')).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('Worker documentation.')).toBeInTheDocument()

    const elements = graphHarness.calls.at(-1)?.elements ?? []
    expect(elements).toEqual(expect.arrayContaining([
      expect.objectContaining({ data: expect.objectContaining({ id: 'boundary:cccccccc-cccc-4ccc-8ccc-cccccccccccc', displayLabel: 'Shared\nrecords.md\nLives in System' }) }),
      expect.objectContaining({ data: expect.objectContaining({ source: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', target: 'boundary:cccccccc-cccc-4ccc-8ccc-cccccccccccc', label: 'writes' }) }),
      expect.objectContaining({ data: expect.objectContaining({ source: 'boundary:cccccccc-cccc-4ccc-8ccc-cccccccccccc', target: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', label: 'feeds' }) }),
    ]))
    expect(elements.filter((element) => (element as { data?: { id?: string } }).data?.id === 'boundary:cccccccc-cccc-4ccc-8ccc-cccccccccccc')).toHaveLength(1)
    expect(elements.filter((element) => (element as { data?: { label?: string } }).data?.label === 'writes')).toHaveLength(2)

    act(() => graphHarness.nodeSelect?.({ target: { id: () => 'boundary:cccccccc-cccc-4ccc-8ccc-cccccccccccc' } }))
    expect(within(navigator).getByRole('button', { name: 'System' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('Records documentation.')).toBeInTheDocument()

    await user.click(within(navigator).getByRole('button', { name: 'Detail, Inside Shared — gateway.md' }))
    await user.click(within(screen.getByRole('navigation', { name: 'Diagram breadcrumbs' })).getByRole('button', { name: 'System' }))
    expect(screen.getByText('Gateway documentation.')).toBeInTheDocument()

    await user.click(within(navigator).getByRole('button', { name: 'Detail, Inside Shared — records.md' }))
    expect(screen.getByText('This diagram has no components.')).toBeInTheDocument()
    expect(screen.queryByText('The architecture has no components yet.')).not.toBeInTheDocument()
  })

  it.each([
    ['known non-current', { stale: true, action_error: 'refresh_invalid' }, undefined, 'The current architecture could not be loaded. This earlier view is read-only.', true],
    ['indeterminate Refresh', {}, { action_error: 'refresh_failed' }, "WorkBraid couldn't check for architecture changes. Try Refresh again.", false],
    ['already-known stale after indeterminate Refresh', { stale: true, action_error: 'refresh_invalid' }, { stale: true, action_error: 'refresh_failed' }, 'The current architecture could not be loaded. This earlier view is read-only.', true],
  ])('preserves %s authority semantics after writable-v2 authoring lands', async (_case, initialOverrides, refreshOverrides, message, readOnly) => {
    mockResponses(refreshOverrides ? [acceptedV2(initialOverrides), acceptedV2(refreshOverrides)] : [acceptedV2(initialOverrides)])
    render(<App />)
    await submitPath('/tmp/example')
    if (refreshOverrides) {
      const user = userEvent.setup()
      await user.click(screen.getByRole('button', { name: 'Refresh' }))
    }

    expect(await screen.findByText(message)).toBeInTheDocument()
    const mutations = screen.queryAllByRole('button', { name: /add component|edit component|review changes|update architecture/i })
    expect(mutations.length === 0).toBe(readOnly)
  })

  it('offers concise Diagram setup instead of ordinary authoring for readable v1', async () => {
    const v1 = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '1'.repeat(40), format_version: 1,
      component_count: 1, component_titles: ['Gateway'],
      components: [{ id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted.\n', relationships: [] }],
    }
    const setup = { ...v1, changes: { components: [], valid: true, diagram_setup: true } }
    const fetchMock = mockResponses([v1, setup])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    expect(screen.queryByRole('button', { name: 'Edit component' })).not.toBeInTheDocument()
    await user.click(await screen.findByRole('button', { name: 'Set up diagrams' }))
    expect(await screen.findByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
    expect(screen.getByText('Setting up diagrams will make this architecture editable.')).toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/diagrams/setup')
    expect(screen.queryByText(/format|yaml|parser|schema/i)).not.toBeInTheDocument()
  })

  it('keeps accepted components separate while one edit and one addition accumulate as changes in progress', async () => {
    const accepted = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: 'f'.repeat(40),
      component_count: 1, component_titles: ['API'],
      components: [{ id: 'api-id', title: 'API', description: '\nAccepted body\n' }],
    }
    const edited = {
      ...accepted,
      changes: {
        valid: true,
        components: [{ id: 'api-id', title: 'Gateway', description: '\nChanged body\n', new: false }],
      },
    }
    const both = {
      ...accepted,
      changes: {
        valid: true,
        components: [
          ...edited.changes.components,
          { id: 'worker-id', title: 'Worker', description: '\nDoes work\n', new: true },
        ],
      },
    }
    const fetchMock = mockResponses([accepted, edited, both])
    render(<App />)
    await submitPath('/tmp/example')

    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    await user.clear(screen.getByLabelText('Title'))
    await user.type(screen.getByLabelText('Title'), 'Gateway')
    await user.clear(screen.getByLabelText('Description'))
    await user.type(screen.getByLabelText('Description'), 'Changed body\n')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))

    expect(await screen.findByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
    expect(screen.getByText('These changes have not updated the architecture yet.')).toBeInTheDocument()
    expect(within(screen.getByRole('navigation', { name: 'Components' })).getByText('API')).toBeInTheDocument()
    expect(screen.getByText('Gateway')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Add component' }))
    await user.type(screen.getByLabelText('Title'), 'Worker')
    await user.type(screen.getByLabelText('Description'), '\nDoes work\n')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))

    expect(await screen.findByText('Worker')).toBeInTheDocument()
    expect(within(screen.getByRole('heading', { name: 'Changes in progress' }).closest('section') as HTMLElement).getAllByRole('listitem')).toHaveLength(2)
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/components/edit')
    expect(requestBody(fetchMock, 1)).toEqual({
      source_root: '/tmp/example', expected_revision: 'f'.repeat(40), component_id: 'api-id', title: 'Gateway', description: '\nChanged body\n',
      title_changed: true, description_changed: true,
    })
    expect(requestPath(fetchMock, 2)).toBe('/api/architecture/components/add')
    expect(graphHarness.calls).toHaveLength(1)
    expect(graphHarness.calls[0].elements).toEqual([
      expect.objectContaining({ data: expect.objectContaining({ id: 'api-id', label: 'API' }), position: { x: 0, y: 0 } }),
    ])
  })

  it('sends explicit title-only intent without an untouched CRLF description', async () => {
    const accepted = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: 'f'.repeat(40),
      component_count: 1, component_titles: ['API'],
      components: [{ id: 'api-id', title: 'API', description: '\r\nExact body  \r\nSecond exact\r\n' }],
    }
    const edited = {
      ...accepted,
      changes: {
        valid: true,
        components: [{ id: 'api-id', title: 'Gateway', description: accepted.components[0].description, new: false }],
      },
    }
    const fetchMock = mockResponses([accepted, edited])
    render(<App />)
    await submitPath('/tmp/example')

    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    expect(screen.getByLabelText('Description')).toHaveValue('Exact body  \nSecond exact\n')
    await user.clear(screen.getByLabelText('Title'))
    await user.type(screen.getByLabelText('Title'), 'Gateway')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))

    expect(requestBody(fetchMock, 1)).toEqual({
      source_root: '/tmp/example',
      expected_revision: 'f'.repeat(40),
      component_id: 'api-id',
      title: 'Gateway',
      title_changed: true,
      description_changed: false,
    })
  })

  it('hides one structural body separator while preserving it in a description edit', async () => {
    const accepted = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: 'f'.repeat(40),
      component_count: 1, component_titles: ['API'],
      components: [{ id: 'api-id', title: 'API', description: '\nOriginal body\n' }],
    }
    const edited = {
      ...accepted,
      changes: {
        valid: true,
        components: [{ id: 'api-id', title: 'API', description: '\nChanged body\n', new: false }],
      },
    }
    const fetchMock = mockResponses([accepted, edited])
    render(<App />)
    await submitPath('/tmp/example')

    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    expect(screen.getByLabelText('Description')).toHaveValue('Original body\n')
    await user.clear(screen.getByLabelText('Description'))
    await user.type(screen.getByLabelText('Description'), 'Changed body')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))

    expect(requestBody(fetchMock, 1)).toEqual({
      source_root: '/tmp/example',
      expected_revision: 'f'.repeat(40),
      component_id: 'api-id',
      description: '\nChanged body',
      title_changed: false,
      description_changed: true,
    })
  })

  it('retrieves invalid backend-held changes after a browser reload and keeps them correctable', async () => {
    const accepted = {
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '1'.repeat(40),
      component_count: 0, component_titles: [], components: [],
    }
    const invalid = {
      ...accepted,
      changes: {
        valid: false,
        validation_code: 'title_required',
        validation_item: 'new-id',
        components: [{ id: 'new-id', title: '', description: 'Useful description\n', new: true }],
      },
    }
    const invalidWithWorker = {
      ...accepted,
      changes: {
        valid: false,
        validation_code: 'title_required',
        validation_item: 'new-id',
        components: [
          { id: 'new-id', title: '', description: 'Useful description\n', new: true },
          { id: 'worker-id', title: 'Worker', description: 'Does work.\n', new: true },
        ],
      },
    }
    const corrected = {
      ...accepted,
      changes: {
        valid: true,
        components: [
          { id: 'new-id', title: 'Gateway', description: 'Useful description\n', new: true },
          { id: 'worker-id', title: 'Worker', description: 'Does work.\n', new: true },
        ],
      },
    }
    mockResponses([accepted, invalid, invalidWithWorker, invalidWithWorker, corrected])
    const first = render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Add component' }))
    await user.type(screen.getByLabelText('Title'), '   ')
    await user.type(screen.getByLabelText('Description'), 'Useful description')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))
    expect(await screen.findByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
    expect(screen.getByText('Untitled component')).toBeInTheDocument()
    expect(screen.queryByText('Add a title.')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Edit component' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Add component' }))
    await user.type(screen.getByLabelText('Title'), 'Worker')
    await user.type(screen.getByLabelText('Description'), 'Does work.')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))
    expect(await screen.findByText('Worker')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Edit component' })).not.toBeInTheDocument()

    first.unmount()
    render(<App />)
    await submitPath('/tmp/example')
    expect(await screen.findByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
    expect(screen.getByText('Untitled component')).toBeInTheDocument()
    expect(screen.getByText('Worker')).toBeInTheDocument()
    expect(screen.queryByText('Add a title.')).not.toBeInTheDocument()
    const reloadedUser = userEvent.setup()
    const untitledItem = screen.getByText('Untitled component').closest('li') as HTMLElement
    await reloadedUser.click(within(untitledItem).getByRole('button', { name: 'Edit' }))
    expect(screen.getByLabelText('Title')).toHaveValue('')
    await reloadedUser.type(screen.getByLabelText('Title'), 'Gateway')
    await reloadedUser.click(screen.getByRole('button', { name: 'Keep change' }))
    expect(await screen.findByText('Gateway')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('reviews the complete canonical diff before deliberately updating architecture', async () => {
    const base = '1'.repeat(40)
    const candidate = '2'.repeat(40)
    const successor = '3'.repeat(40)
    const beforeComponents = [
      { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted gateway.\n', relationships: [{ target_id: 'worker', label: 'calls', projection_key: 'review:before:gateway:0' }] },
      { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Worker body.\n', relationships: [] },
      { id: 'docs', title: 'Docs', filename: 'docs.md', description: 'Old docs.\n', relationships: [] },
    ]
    const withComponents = [
      { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Changed gateway.\n', relationships: [
        { target_id: 'worker', label: 'routes', projection_key: 'review:with:gateway:0' },
        { target_id: 'queue', label: 'publishes', projection_key: 'review:with:gateway:1' },
        { target_id: 'queue', label: 'publishes', projection_key: 'review:with:gateway:2' },
      ] },
      { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Worker body.\n', relationships: [{ target_id: 'queue', label: 'observes', projection_key: 'review:with:worker:0' }] },
      { id: 'docs', title: 'Docs', filename: 'docs.md', description: 'New docs.\n', relationships: [] },
      { id: 'queue', title: 'Queue', filename: 'queue.md', description: 'New queue.\n', relationships: [] },
    ]
    const diff = 'diff --git a/components/gateway.md b/components/gateway.md\n--- a/components/gateway.md\n+++ b/components/gateway.md\n@@ -1 +1 @@\n-Accepted gateway.\n+Changed gateway.\ndiff --git a/components/queue.md b/components/queue.md\nnew file mode 100644\n--- /dev/null\n+++ b/components/queue.md\n@@ -0,0 +1 @@\n+New queue.\n'
    const pending = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: base,
      component_count: 3, component_titles: ['Gateway', 'Worker', 'Docs'], components: beforeComponents,
      changes: { valid: true, components: [
        { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Changed gateway.\n', relationships: withComponents[0].relationships, new: false },
        { id: 'queue', title: 'Queue', filename: 'queue.md', description: 'New queue.\n', relationships: [], new: true },
        { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Worker body.\n', relationships: withComponents[1].relationships, new: false },
        { id: 'docs', title: 'Docs', filename: 'docs.md', description: 'New docs.\n', relationships: [], new: false },
      ] },
    }
    const reviewed = {
      ...pending,
      changes: {
        ...pending.changes,
        review: testReview({
          base, candidate, diff, before: beforeComponents, withChanges: withComponents,
          componentChanges: [
            { component_id: 'gateway', status: 'content_changed', path: 'components/gateway.md' },
            { component_id: 'docs', status: 'content_changed', path: 'components/docs.md' },
            { component_id: 'queue', status: 'added', path: 'components/queue.md' },
          ],
          relationshipChanges: [
            { key: 'review:removed:gateway:0', before_key: 'review:before:gateway:0', source_id: 'gateway', target_id: 'worker', label: 'calls', status: 'removed', path: 'components/gateway.md', occurrence: 1 },
            { key: 'review:with:gateway:0', source_id: 'gateway', target_id: 'worker', label: 'routes', status: 'added', path: 'components/gateway.md', occurrence: 1 },
            { key: 'review:with:gateway:1', source_id: 'gateway', target_id: 'queue', label: 'publishes', status: 'added', path: 'components/gateway.md', occurrence: 1 },
            { key: 'review:with:gateway:2', source_id: 'gateway', target_id: 'queue', label: 'publishes', status: 'added', path: 'components/gateway.md', occurrence: 2 },
            { key: 'review:with:worker:0', source_id: 'worker', target_id: 'queue', label: 'observes', status: 'added', path: 'components/worker.md', occurrence: 1 },
          ],
        }),
      },
    }
    const accepted = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: successor,
      component_count: 4, component_titles: ['Gateway', 'Worker', 'Docs', 'Queue'], components: withComponents,
      parent_diff: reviewed.changes.review.diff,
    }
    const fetchMock = mockResponses([pending, reviewed, accepted])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()

    expect(await screen.findByRole('button', { name: 'Review changes' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Review changes' }))

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'With changes' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByTestId('raw-diff').textContent).toBe(diff)
    expect(screen.getByRole('button', { name: 'Added component: Queue' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Content changed: Gateway' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Content changed: Docs' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Added relationship, occurrence 1: Gateway — publishes — Queue' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Added relationship, occurrence 2: Gateway — publishes — Queue' })).toBeInTheDocument()
    const reviewElements = graphHarness.calls.at(-1)?.elements as Array<{ data: Record<string, unknown> }>
    const reviewStyles = graphHarness.calls.at(-1)?.style ?? []
    expect(reviewElements.find((element) => element.data.id === 'worker')?.data.reviewStatus).toBe('unchanged')
    expect(reviewElements.find((element) => element.data.id === 'review:with:worker:0')?.data.reviewStatus).toBe('added')
    expect(reviewElements.find((element) => element.data.id === 'gateway')?.data.reviewStatus).toBe('content_changed')
    expect(reviewStyles.find((rule) => rule.selector === 'node:selected')?.style).toEqual({
      'background-color': '#e7dba9', 'border-color': '#18734f', 'border-width': 4, opacity: 1,
    })
    expect(reviewStyles.find((rule) => rule.selector === 'edge:selected')?.style).toEqual({
      width: 4, opacity: 1, 'line-color': '#18734f', 'target-arrow-color': '#18734f',
    })
    expect(reviewStyles.find((rule) => rule.selector === 'node[reviewStatus = "unchanged"]:selected')?.style).toEqual({
      'background-color': '#f8f0dc', 'border-color': '#27251f', 'border-width': 5, 'border-style': 'dotted', opacity: 1,
    })
    expect(reviewStyles.find((rule) => rule.selector === 'node[reviewStatus = "added"]:selected')?.style).toEqual({
      'background-color': '#d8eadf', 'border-color': '#126747', 'border-width': 5, shape: 'hexagon', opacity: 1,
    })
    expect(reviewStyles.find((rule) => rule.selector === 'node[reviewStatus = "content_changed"]:selected')?.style).toEqual({
      'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-width': 5, 'border-style': 'dashed', opacity: 1,
    })
    expect(reviewStyles.find((rule) => rule.selector === 'edge[reviewStatus = "unchanged"]:selected')?.style).toEqual({
      width: 4.5, opacity: 1, 'line-color': '#736c5c', 'target-arrow-color': '#736c5c', 'line-style': 'dotted',
    })
    expect(reviewStyles.find((rule) => rule.selector === 'edge[reviewStatus = "added"]:selected')?.style).toEqual({
      width: 4.5, opacity: 1, 'line-color': '#126747', 'target-arrow-color': '#126747',
    })
    expect(reviewStyles.find((rule) => rule.selector === 'edge[reviewStatus = "removed"]:selected')?.style).toEqual({
      width: 4.5, opacity: 1, 'line-color': '#a04432', 'target-arrow-color': '#a04432', 'line-style': 'dashed',
    })
    expect(screen.getByRole('button', { name: 'Update architecture' })).toBeInTheDocument()
    const reviewDetails = screen.getByText('Review details').closest('details') as HTMLElement
    const technicalDetails = screen.getByText('Technical details').closest('details') as HTMLElement
    expect(within(reviewDetails).getByText(base)).toBeInTheDocument()
    expect(within(reviewDetails).getByText(candidate)).toBeInTheDocument()
    expect(reviewDetails.compareDocumentPosition(technicalDetails) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()

    await user.click(screen.getByRole('button', { name: 'Added component: Queue' }))
    expect(screen.getByRole('heading', { name: 'Queue' })).toBeInTheDocument()
    expect(screen.getByText('New queue.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    const v1Index = screen.getByRole('navigation', { name: 'Components' })
    expect(within(v1Index).queryByText('Queue')).not.toBeInTheDocument()
    expect(within(v1Index).getByRole('button', { name: 'Gateway, Content changed' })).toBeInTheDocument()
    expect(within(v1Index).getByRole('button', { name: 'Worker' })).toBeInTheDocument()
    expect(within(v1Index).getByRole('button', { name: 'Docs, Content changed' })).toBeInTheDocument()
    expect(screen.queryByText('New queue.')).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Select a change' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'With changes' }))
    await user.click(screen.getByRole('button', { name: 'Removed relationship: Gateway — calls — Worker' }))
    expect(document.activeElement).toHaveAttribute('data-diff-path', 'components/gateway.md')
    expect(screen.getByLabelText('Review context')).toHaveTextContent('Removed relationship')
    await user.click(screen.getByRole('button', { name: 'Clear focus' }))
    expect(screen.getByRole('heading', { name: 'Select a change' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Removed relationship: Gateway — calls — Worker' }))
    await user.click(screen.getByRole('button', { name: 'Update architecture' }))

    expect(await screen.findByRole('heading', { name: 'Gateway' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Changes in progress' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    const acceptedTechnicalDetails = screen.getByText('Technical details').closest('details') as HTMLElement
    expect(within(acceptedTechnicalDetails).getByText(successor)).toBeInTheDocument()
    expect(within(acceptedTechnicalDetails).getByRole('heading', { name: 'Parent diff' })).toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/review')
    expect(requestPath(fetchMock, 2)).toBe('/api/architecture/accept')
    expect(requestBody(fetchMock, 2)).toEqual({
      source_root: '/tmp/example', base_revision: base, candidate_tree: candidate, generation: 1,
    })
  })

  it('focuses exact v2 internal and boundary relationship edges without classifying relationship-only content changes', async () => {
    const base = '4'.repeat(40)
    const candidate = '5'.repeat(40)
    const root = '11111111-1111-4111-8111-111111111111'
    const detail = '22222222-2222-4222-8222-222222222222'
    const gateway = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
    const worker = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const records = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
    const fixture = acceptedV2({ revision: base })
    const before = {
      ...fixture,
      diagrams: fixture.diagrams.map((diagram) => ({
        ...diagram,
        relationships: diagram.relationships.map((relationship) => ({
          ...relationship,
          key: `diagram:${diagram.id}:${relationship.source_component_id}:${relationship.key.split(':').at(-1)}`,
        })),
      })),
    }
    const withComponents = before.components.map((component) => component.id === worker ? {
      ...component,
      relationships: [{ target_id: records, label: 'writes' }, { target_id: gateway, label: 'reports' }],
    } : component)
    const withDiagrams = before.diagrams.map((diagram) => ({
      ...diagram,
      relationships: diagram.relationships.map((relationship) => relationship.key === `diagram:${diagram.id}:${worker}:1` ? {
        ...relationship,
        target_node_key: diagram.id === detail ? `boundary:${gateway}` : gateway,
        target_component_id: gateway,
        label: 'reports',
      } : relationship),
    }))
    const relationshipChanges = [
      {
        key: `review:with:${worker}:1`, source_id: worker, target_id: gateway, source_title: 'Worker', target_title: 'Shared',
        label: 'reports', status: 'added' as const, path: 'components/worker.md', occurrence: 1,
        diagram_projections: [
          { side: 'with' as const, diagram_id: root, key: `diagram:${root}:${worker}:1`, source_node_key: worker, target_node_key: gateway },
          { side: 'with' as const, diagram_id: detail, key: `diagram:${detail}:${worker}:1`, source_node_key: worker, target_node_key: `boundary:${gateway}` },
        ],
      },
      {
        key: `review:removed:${worker}:1`, before_key: `review:before:${worker}:1`, source_id: worker, target_id: records, source_title: 'Worker', target_title: 'Shared',
        label: 'writes', status: 'removed' as const, path: 'components/worker.md', occurrence: 2,
        diagram_projections: [
          { side: 'before' as const, diagram_id: root, key: `diagram:${root}:${worker}:1`, source_node_key: worker, target_node_key: records },
          { side: 'before' as const, diagram_id: detail, key: `diagram:${detail}:${worker}:1`, source_node_key: worker, target_node_key: `boundary:${records}` },
        ],
      },
    ]
    const diff = 'diff --git a/components/worker.md b/components/worker.md\n--- a/components/worker.md\n+++ b/components/worker.md\n@@ -4,2 +4,2 @@\n-    label: writes\n+    label: reports\n'
    const reviewed = acceptedV2({
      revision: base,
      changes: {
        valid: true,
        components: [{ ...withComponents.find((component) => component.id === worker), new: false }],
        review: {
          diff, base_revision: base, candidate_tree: candidate, generation: 1,
          before: { ...before, revision: base },
          with_changes: { ...before, revision: candidate, components: withComponents, diagrams: withDiagrams },
          comparison: { components: [], relationships: relationshipChanges },
        },
      },
    })
    mockResponses([reviewed])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()

    expect(await screen.findByRole('button', { name: 'Added relationship: Worker — reports — Shared' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Removed relationship: Worker — writes — Shared' })).toBeInTheDocument()
    const rootElements = graphHarness.calls.at(-1)?.elements as Array<{ data: Record<string, unknown> }>
    expect(rootElements.map((element) => element.data.id)).toContain(`diagram:${root}:${worker}:1`)
    expect(rootElements.find((element) => element.data.id === `diagram:${root}:${worker}:1`)?.data).toMatchObject({
      reviewStatus: 'added', source: worker, target: gateway, source_title: 'Worker', target_title: 'Shared',
    })
    expect(rootElements.find((element) => element.data.id === worker)?.data.reviewStatus).toBe('unchanged')

    await user.click(screen.getByRole('button', { name: 'Added relationship: Worker — reports — Shared' }))
    expect(graphHarness.selectedIDs).toContain(`diagram:${root}:${worker}:1`)
    expect(document.activeElement).toHaveAttribute('data-diff-path', 'components/worker.md')
    expect(screen.getByLabelText('Review context')).toHaveTextContent('WorkerreportsShared')

    await user.click(screen.getByRole('button', { name: 'Removed relationship: Worker — writes — Shared' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    expect(graphHarness.selectedIDs).toContain(`diagram:${root}:${worker}:1`)
    const removedRootElements = graphHarness.calls.at(-1)?.elements as Array<{ data: Record<string, unknown> }>
    expect(removedRootElements.find((element) => element.data.id === `diagram:${root}:${worker}:1`)?.data).toMatchObject({ reviewStatus: 'removed', source: worker, target: records })

    await user.click(screen.getByRole('button', { name: 'With changes' }))
    await user.click(screen.getByRole('button', { name: 'Detail, Inside Shared — gateway.md' }))
    const detailElements = graphHarness.calls.at(-1)?.elements as Array<{ data: Record<string, unknown> }>
    const addedBoundary = detailElements.find((element) => element.data.id === `diagram:${detail}:${worker}:1`)
    expect(addedBoundary?.data).toMatchObject({ reviewStatus: 'added', source: worker, target: `boundary:${gateway}`, source_id: worker, target_id: gateway })
    expect(screen.getByRole('button', { name: 'Added relationship: Worker — reports — Shared' })).toBeInTheDocument()
    expect(screen.queryByText(gateway)).not.toBeInTheDocument()

    await act(async () => {
      graphHarness.edgeSelect?.({ target: { data: () => addedBoundary?.data } })
    })
    expect(graphHarness.selectedIDs).toContain(`diagram:${detail}:${worker}:1`)
    expect(document.activeElement).toHaveAttribute('data-diff-path', 'components/worker.md')
    await user.click(screen.getByRole('button', { name: 'Removed relationship: Worker — writes — Shared' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    const removedBoundaryElements = graphHarness.calls.at(-1)?.elements as Array<{ data: Record<string, unknown> }>
    expect(removedBoundaryElements.find((element) => element.data.id === `diagram:${detail}:${worker}:1`)?.data).toMatchObject({ reviewStatus: 'removed', source: worker, target: `boundary:${records}` })
  })

  it('keeps external-source boundary relationship focus free of unrelated Diagram documentation', async () => {
    const base = '8'.repeat(40)
    const candidate = '9'.repeat(40)
    const root = '11111111-1111-4111-8111-111111111111'
    const detail = '22222222-2222-4222-8222-222222222222'
    const gateway = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
    const worker = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const before = acceptedV2({ revision: base })
    const withComponents = before.components.map((component) => component.id === gateway ? {
      ...component,
      relationships: [{ target_id: worker, label: 'invokes' }],
    } : component)
    const withDiagrams = before.diagrams.map((diagram) => ({
      ...diagram,
      relationships: diagram.relationships.map((relationship) => relationship.source_component_id === gateway ? {
        ...relationship,
        label: 'invokes',
      } : relationship),
    }))
    const diff = 'diff --git a/components/gateway.md b/components/gateway.md\n--- a/components/gateway.md\n+++ b/components/gateway.md\n@@ -4 +4 @@\n-    label: calls\n+    label: invokes\n'
    const reviewed = acceptedV2({
      revision: base,
      changes: {
        valid: true,
        components: [{ ...withComponents.find((component) => component.id === gateway), new: false }],
        review: {
          diff, base_revision: base, candidate_tree: candidate, generation: 1,
          before: { ...before, revision: base },
          with_changes: { ...before, revision: candidate, components: withComponents, diagrams: withDiagrams },
          comparison: {
            components: [],
            relationships: [
              {
                key: `review:with:${gateway}:0`, source_id: gateway, target_id: worker, source_title: 'Shared', target_title: 'Worker',
                label: 'invokes', status: 'added', path: 'components/gateway.md', occurrence: 1,
                diagram_projections: [
                  { side: 'with', diagram_id: root, key: `diagram:${root}:gateway:0`, source_node_key: gateway, target_node_key: worker },
                  { side: 'with', diagram_id: detail, key: `diagram:${detail}:gateway:0`, source_node_key: `boundary:${gateway}`, target_node_key: worker },
                ],
              },
              {
                key: `review:removed:${gateway}:0`, before_key: `review:before:${gateway}:0`, source_id: gateway, target_id: worker, source_title: 'Shared', target_title: 'Worker',
                label: 'calls', status: 'removed', path: 'components/gateway.md', occurrence: 1,
                diagram_projections: [
                  { side: 'before', diagram_id: root, key: `diagram:${root}:gateway:0`, source_node_key: gateway, target_node_key: worker },
                  { side: 'before', diagram_id: detail, key: `diagram:${detail}:gateway:0`, source_node_key: `boundary:${gateway}`, target_node_key: worker },
                ],
              },
            ],
          },
        },
      },
    })
    mockResponses([reviewed])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    const navigator = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(navigator).getByRole('button', { name: 'Detail, Inside Shared — gateway.md' }))

    const addedEdge = (graphHarness.calls.at(-1)?.elements as Array<{ data: Record<string, unknown> }>).find((element) => element.data.id === `diagram:${detail}:gateway:0`)
    expect(addedEdge?.data).toMatchObject({
      reviewStatus: 'added', source: `boundary:${gateway}`, target: worker, source_title: 'Shared', target_title: 'Worker',
    })
    expect(screen.getByRole('button', { name: 'Added relationship: Shared — invokes — Worker' })).toBeInTheDocument()
    await act(async () => {
      graphHarness.edgeSelect?.({ target: { data: () => addedEdge?.data } })
    })

    expect(graphHarness.selectedIDs).toContain(`diagram:${detail}:gateway:0`)
    expect(document.activeElement).toHaveAttribute('data-diff-path', 'components/gateway.md')
    const externalContext = screen.getByLabelText('Review context')
    expect(within(externalContext).getByRole('heading', { name: 'Relationship' })).toBeInTheDocument()
    expect(externalContext).toHaveTextContent('Added relationship')
    expect(externalContext).toHaveTextContent('SharedinvokesWorker')
    expect(screen.queryByText('Gateway documentation.')).not.toBeInTheDocument()
    expect(screen.queryByText('Worker documentation.')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Shared' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Worker' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Removed relationship: Shared — calls — Worker' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    expect(graphHarness.selectedIDs).toContain(`diagram:${detail}:gateway:0`)
    expect(screen.getByLabelText('Review context')).toHaveTextContent('Removed relationship')
    expect(screen.getByLabelText('Review context')).toHaveTextContent('SharedcallsWorker')
    expect(screen.queryByText('Worker documentation.')).not.toBeInTheDocument()

    await user.click(within(navigator).getByRole('button', { name: 'System' }))
    await user.click(screen.getByRole('button', { name: 'Removed relationship: Shared — calls — Worker' }))
    expect(screen.getByLabelText('Review context')).toHaveTextContent('Gateway documentation.')
    expect(within(screen.getByLabelText('Review context')).getByRole('heading', { name: 'Shared' })).toBeInTheDocument()
    expect(document.activeElement).toHaveAttribute('data-diff-path', 'components/gateway.md')
  })

  it('keeps v2 review index and documentation scoped to the selected side and Diagram', async () => {
    const base = '6'.repeat(40)
    const candidate = '7'.repeat(40)
    const detail = '22222222-2222-4222-8222-222222222222'
    const records = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
    const before = acceptedV2({ revision: base })
    const withDiagrams = before.diagrams.map((diagram) => diagram.id === detail ? {
      ...diagram,
      appearances: [...diagram.appearances, { component_id: records, role: 'reference' }],
      boundaries: diagram.boundaries.filter((boundary) => boundary.component_id !== records),
      relationships: diagram.relationships.map((relationship) => ({
        ...relationship,
        source_node_key: relationship.source_node_key === `boundary:${records}` ? records : relationship.source_node_key,
        target_node_key: relationship.target_node_key === `boundary:${records}` ? records : relationship.target_node_key,
      })),
    } : diagram)
    const reviewed = acceptedV2({
      revision: base,
      changes: {
        valid: true,
        components: [],
        review: {
          diff: 'diff --git a/diagrams/detail.yaml b/diagrams/detail.yaml\n+  - component: records\n+    role: reference\n',
          base_revision: base,
          candidate_tree: candidate,
          generation: 1,
          before,
          with_changes: { ...before, revision: candidate, diagrams: withDiagrams },
          comparison: {
            components: [], relationships: [],
            appearances: [{ diagram_id: detail, component_id: records, role: 'reference', status: 'added', path: 'diagrams/detail.yaml' }],
          },
        },
      },
    })
    mockResponses([reviewed])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    const navigator = await screen.findByRole('navigation', { name: 'Diagrams and components' })

    expect(within(navigator).getByRole('button', { name: 'Shared, gateway.md' })).toBeInTheDocument()
    expect(within(navigator).getByRole('button', { name: 'Shared, records.md' })).toBeInTheDocument()
    await user.click(within(navigator).getByRole('button', { name: 'Detail, Inside Shared — gateway.md' }))
    expect(within(navigator).getByRole('button', { name: 'Worker' })).toBeInTheDocument()
    expect(within(navigator).getByRole('button', { name: 'Shared' })).toBeInTheDocument()

    await user.click(within(navigator).getByRole('button', { name: 'Shared' }))
    expect(screen.getByText('Records documentation.')).toBeInTheDocument()
    expect(screen.queryByText('Gateway documentation.')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(within(navigator).getByRole('button', { name: 'Worker' })).toBeInTheDocument()
    expect(within(navigator).queryByRole('button', { name: 'Shared' })).not.toBeInTheDocument()
    expect(screen.queryByText('Records documentation.')).not.toBeInTheDocument()
    expect(await screen.findByText('Worker documentation.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'With changes' }))
    expect(within(navigator).getByRole('button', { name: 'Shared' })).toBeInTheDocument()
    expect(screen.getByText('Worker documentation.')).toBeInTheDocument()
    await user.click(within(navigator).getByRole('button', { name: 'System' }))
    expect(within(navigator).getByRole('button', { name: 'Shared, gateway.md' })).toBeInTheDocument()
    expect(within(navigator).getByRole('button', { name: 'Shared, records.md' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(within(navigator).getByRole('button', { name: 'Shared, gateway.md' })).toBeInTheDocument()
    expect(within(navigator).getByRole('button', { name: 'Shared, records.md' })).toBeInTheDocument()
  })

  it('turns an invalid quiet pending title into actionable guidance only at review', async () => {
    const pending = {
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '1'.repeat(40),
      component_count: 0, component_titles: [], components: [],
      changes: { valid: false, validation_code: 'title_required', validation_item: 'worker-id', components: [{ id: 'worker-id', title: '', description: '', new: true }] },
    }
    const blocked = { ...pending, action_error: 'review_failed', changes: { ...pending.changes, review_blocker: 'title_required' } }
    mockResponses([pending, blocked], [200, 422])
    render(<App />)
    await submitPath('/tmp/example')

    expect(await screen.findByText('Untitled component')).toBeInTheDocument()
    expect(screen.queryByText(/add a title/i)).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Review changes' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Add a title to the untitled component before updating architecture.')
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    expect(screen.getByText('Untitled component')).toBeInTheDocument()
  })

  it('keeps exact review and deliberate confirmation usable when map initialization fails', async () => {
    const base = '4'.repeat(40)
    const candidate = '5'.repeat(40)
    const component = { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Changed.\n', relationships: [] }
    const diff = 'diff --git a/components/gateway.md b/components/gateway.md\n@@ -1 +1 @@\n-Old\n+Changed\n'
    const reviewed = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: base,
      component_count: 1, component_titles: ['Gateway'], components: [{ ...component, description: 'Old.\n' }],
      changes: {
        valid: true,
        components: [{ ...component, new: false }],
        review: testReview({
          base, candidate, diff,
          before: [{ ...component, description: 'Old.\n' }],
          withChanges: [component],
          componentChanges: [{ component_id: 'gateway', status: 'content_changed', path: 'components/gateway.md' }],
        }),
      },
    }
    graphHarness.fail = true
    mockResponses([reviewed])
    render(<App />)
    await submitPath('/tmp/example')

    expect(await screen.findByRole('alert')).toHaveTextContent('The architecture map could not be shown.')
    expect(screen.getByTestId('raw-diff').textContent).toBe(diff)
    expect(screen.getByText('Review details')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Update architecture' })).toBeEnabled()
  })

  it('keeps duplicate review titles bound to stable component identity across the side toggle', async () => {
    const before = [
      { id: 'public', title: 'Gateway', filename: 'public.md', description: 'Public before.\n', relationships: [] },
      { id: 'private', title: 'Gateway', filename: 'private.md', description: 'Private before.\n', relationships: [] },
    ]
    const withChanges = [
      before[0],
      { ...before[1], description: 'Private with changes.\n' },
    ]
    mockResponses([{
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '4'.repeat(40),
      component_count: 2, component_titles: ['Gateway', 'Gateway'], components: before,
      changes: {
        valid: true,
        components: [{ ...withChanges[1], new: false }],
        review: testReview({
          base: '4'.repeat(40), candidate: '5'.repeat(40), before, withChanges,
          diff: 'diff --git a/components/private.md b/components/private.md\n-Private before.\n+Private with changes.\n',
          componentChanges: [{ component_id: 'private', status: 'content_changed', path: 'components/private.md' }],
        }),
      },
    }])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    const index = await screen.findByRole('navigation', { name: 'Components' })

    await user.click(within(index).getByRole('button', { name: 'Gateway, private.md, Content changed' }))
    expect(screen.getByText('Private with changes.')).toBeInTheDocument()
    expect(screen.queryByText('Public before.')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(screen.getByText('Private before.')).toBeInTheDocument()
    expect(screen.queryByText('Private with changes.')).not.toBeInTheDocument()
    expect(within(index).getByRole('button', { name: 'Gateway, private.md, Content changed' })).toHaveAttribute('aria-current', 'page')
  })

  it('marks a stale accepted view read-only while preserving visible changes in progress', async () => {
    const stale = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '1'.repeat(40), stale: true,
      action_error: 'architecture_stale', component_count: 1, component_titles: ['Gateway'],
      components: [{ id: 'gateway-id', title: 'Gateway', description: 'Accepted.\n' }],
      changes: { valid: true, components: [{ id: 'gateway-id', title: 'Public Gateway', description: 'Pending.\n', new: false }] },
    }
    mockResponses([stale])
    render(<App />)
    await submitPath('/tmp/example')

    const alerts = await screen.findAllByRole('alert')
    expect(alerts.some((alert) => alert.textContent?.includes('The current architecture could not be loaded. This earlier view is read-only.'))).toBe(true)
    expect(alerts.some((alert) => alert.textContent?.includes('These changes are out of date because the architecture changed.'))).toBe(true)
    expect(screen.getByText('Public Gateway')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit|add component|review changes|update architecture/i })).not.toBeInTheDocument()
    expect(screen.queryByText(/merge|rebase|overwrite|repair/i)).not.toBeInTheDocument()
  })

  it('refreshes every accepted projection together only after the explicit action', async () => {
    const revisionA = 'a'.repeat(40)
    const revisionB = 'b'.repeat(40)
    const acceptedA = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: revisionA,
      component_count: 2, component_titles: ['Gateway A', 'Worker A'],
      components: [
        { id: 'gateway', title: 'Gateway A', filename: 'gateway.md', description: 'Accepted A.\n', relationships: [{ target_id: 'worker', label: 'calls A' }] },
        { id: 'worker', title: 'Worker A', filename: 'worker.md', description: 'Worker A.\n', relationships: [] },
      ],
    }
    const acceptedB = {
      ...acceptedA, revision: revisionB, component_titles: ['Gateway B', 'Records B'],
      components: [
        { id: 'gateway', title: 'Gateway B', filename: 'gateway.md', description: 'Accepted B.\n', relationships: [{ target_id: 'records', label: 'reads from' }] },
        { id: 'records', title: 'Records B', filename: 'records.md', description: 'Records B.\n', relationships: [] },
      ],
    }
    const fetchMock = mockResponses([acceptedA, acceptedB])
    render(<App />)
    await submitPath('/tmp/example')

    expect(screen.getByText('Accepted A.')).toBeInTheDocument()
    expect(screen.queryByText('Gateway B')).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Refresh' }))

    expect(await screen.findAllByText('Gateway B')).toHaveLength(2)
    expect(screen.getByText('Accepted B.')).toBeInTheDocument()
    expect(screen.queryByText('Gateway A')).not.toBeInTheDocument()
    expect(graphHarness.calls.at(-1)?.elements).toEqual(expect.arrayContaining([
      expect.objectContaining({ data: expect.objectContaining({ id: 'gateway', label: 'Gateway B' }) }),
      expect.objectContaining({ data: expect.objectContaining({ id: 'projection:gateway:records:0', source: 'gateway', target: 'records', label: 'reads from', distance: 0 }) }),
    ]))
    expect(within(screen.getByText('Technical details').closest('details') as HTMLElement).getByText(revisionB)).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/refresh')
    expect(requestBody(fetchMock, 1)).toEqual({ source_root: '/tmp/example' })
  })

  it.each([
    ['Title', async (user: ReturnType<typeof userEvent.setup>) => user.type(screen.getByLabelText('Title'), ' changed')],
    ['Description', async (user: ReturnType<typeof userEvent.setup>) => user.type(screen.getByLabelText('Description'), ' changed')],
    ['relationship', async (user: ReturnType<typeof userEvent.setup>) => {
      await user.click(screen.getByRole('button', { name: 'Add relationship' }))
      await user.selectOptions(screen.getByLabelText('Target'), 'worker')
      await user.type(screen.getByLabelText('Label'), 'calls')
    }],
  ])('guards dirty %s values before Refresh', async (_field, makeDirty) => {
    const workspace = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '9'.repeat(40), component_count: 2,
      component_titles: ['Gateway', 'Worker'],
      components: [
        { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted.\n', relationships: [] },
        { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Works.\n', relationships: [] },
      ],
    }
    const fetchMock = mockResponses([workspace, workspace])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    await makeDirty(user)

    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(screen.getByRole('dialog')).toHaveTextContent('Leave without keeping?')
    await user.click(screen.getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('heading', { name: 'Edit component' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await user.click(screen.getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('heading', { name: 'Gateway' })).toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/refresh')
  })

  it('keeps stale pending work inspectable in its old context while accepted projections show the replacement', async () => {
    const current = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: 'b'.repeat(40),
      component_count: 2, component_titles: ['Gateway B', 'Target B'],
      components: [
        { id: 'gateway', title: 'Gateway B', filename: 'gateway.md', description: 'Current.\n', relationships: [] },
        { id: 'target', title: 'Target B', filename: 'target.md', description: 'Current target.\n', relationships: [] },
      ],
      changes: {
        stale: true, valid: false, validation_code: 'relationship_label_required', validation_item: 'gateway',
        validation_relationship_position: 1, validation_relationship_field: 'label', review_blocker: 'relationship_label_required',
        components: [{ id: 'gateway', title: 'Pending Gateway A', description: 'Pending old body.\n', new: false, relationships: [{ target_id: 'target', label: 'old calls' }] }],
        relationship_targets: [
          { id: 'gateway', title: 'Pending Gateway A' },
          { id: 'target', title: 'Target A' },
        ],
      },
    }
    mockResponses([current])
    render(<App />)
    await submitPath('/tmp/example')

    expect(await screen.findByText('These changes started from an older architecture and are read-only.')).toBeInTheDocument()
    expect(screen.getByRole('navigation', { name: 'Components' })).toHaveTextContent('Gateway B')
    const diagnostic = screen.getByRole('alert')
    expect(diagnostic).toHaveTextContent('Pending Gateway A has a relationship issue in these read-only changes.')
    expect(diagnostic).toHaveTextContent('This relationship has no label.')
    expect(within(diagnostic).queryByRole('button', { name: 'Fix relationship' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /add component|edit|fix relationship|review changes|update architecture/i })).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'View' }))
    expect(screen.getByRole('heading', { name: 'Change details' })).toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toHaveValue('Pending Gateway A')
    expect(screen.getByLabelText('Title')).toHaveAttribute('readonly')
    expect(screen.getByLabelText('Description')).toHaveValue('Pending old body.\n')
    expect(screen.getByLabelText('Target')).toBeDisabled()
    expect(screen.getByRole('option', { name: 'Target A' })).toBeInTheDocument()
    expect(screen.getByLabelText('Label')).toHaveValue('old calls')
    expect(screen.queryByRole('button', { name: 'Keep change' })).not.toBeInTheDocument()
  })

  it('treats legacy pending evidence as inspect and whole-set Discard only even without a stale response flag', async () => {
    const current = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '1'.repeat(40), format_version: 1,
      component_count: 1, component_titles: ['Legacy accepted'],
      components: [{ id: 'accepted', title: 'Legacy accepted', filename: 'accepted.md', description: 'Accepted.\n', relationships: [] }],
      changes: {
        legacy_read_only: true, valid: false, validation_code: 'relationship_label_required', validation_item: 'pending',
        validation_relationship_position: 1, validation_relationship_field: 'label', review_blocker: 'relationship_label_required',
        components: [{ id: 'pending', title: 'Pending evidence', description: 'Earlier pending body.\n', new: false, relationships: [{ target_id: 'target', label: '' }] }],
        relationship_targets: [{ id: 'target', title: 'Earlier target' }],
        review: testReview({
          base: '1'.repeat(40), candidate: '2'.repeat(40), diff: 'retained review must stay hidden',
          before: [{ id: 'accepted', title: 'Legacy accepted', description: 'Accepted.\n' }],
          withChanges: [{ id: 'pending', title: 'Pending evidence', description: 'Earlier pending body.\n' }],
        }),
      },
    }
    mockResponses([current])
    render(<App />)
    await submitPath('/tmp/example')

    expect(await screen.findByText('These changes started from an older architecture and are read-only.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Discard changes' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit|fix relationship|review changes|update architecture/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'With changes' })).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'View' }))
    expect(screen.getByRole('heading', { name: 'Change details' })).toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toHaveAttribute('readonly')
    expect(screen.getByLabelText('Description')).toHaveAttribute('readonly')
    expect(screen.getByLabelText('Target')).toBeDisabled()
    expect(screen.getByLabelText('Label')).toHaveAttribute('readonly')
    expect(screen.queryByRole('button', { name: /keep change|add relationship|remove relationship/i })).not.toBeInTheDocument()
  })

  it.each([
    ['refresh_invalid', 'The current architecture could not be read. This earlier view is read-only.'],
    ['refresh_unsupported', 'The current architecture uses features this version of WorkBraid cannot open.'],
    ['refresh_unavailable', 'The current architecture could not be found. This earlier view is read-only.'],
    ['refresh_changed', 'Architecture changed again while WorkBraid was refreshing. Refresh once more.'],
  ])('presents conclusive %s as a non-current read-only reference', async (actionError, message) => {
    const current = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: 'a'.repeat(40),
      component_count: 1, component_titles: ['Gateway'],
      components: [{ id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Earlier.\n', relationships: [] }],
    }
    mockResponses([current, { ...current, stale: true, action_error: actionError }], [200, 409])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Refresh' }))

    const alerts = await screen.findAllByRole('alert')
    expect(alerts.map((alert) => alert.textContent).join(' ')).toContain(message)
    expect(screen.queryByRole('button', { name: /add component|edit component|review changes|update architecture/i })).not.toBeInTheDocument()
    expect(screen.getByText('Earlier.')).toBeInTheDocument()
  })

  it('reports an indeterminate Refresh failure without claiming the loaded view is stale', async () => {
    const current = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: 'a'.repeat(40),
      component_count: 1, component_titles: ['Gateway'],
      components: [{ id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Current.\n', relationships: [] }],
    }
    mockResponses([current, { ...current, action_error: 'refresh_failed' }], [200, 503])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Refresh' }))

    expect(await screen.findByRole('alert')).toHaveTextContent("WorkBraid couldn't check for architecture changes. Try Refresh again.")
    expect(screen.queryByText(/earlier view is read-only/i)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Edit component' })).toBeInTheDocument()
  })

  it('uses stable identities for accepted index and map projection with collision-only context', async () => {
    mockResponses([{
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '7'.repeat(40),
      component_count: 3, component_titles: ['Gateway', 'Gateway', 'Worker'],
      components: [
        { id: 'gateway-a', title: 'Gateway', filename: 'public.md', description: 'Public body.\n', relationships: [{ target_id: 'worker', label: 'calls' }] },
        { id: 'gateway-b', title: 'Gateway', filename: 'private.md', description: 'Private body.\n', relationships: [{ target_id: 'worker', label: 'reads from' }] },
        { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Worker body.\n', relationships: [] },
      ],
    }])
    render(<App />)
    await submitPath('/tmp/example')

    const index = await screen.findByRole('navigation', { name: 'Components' })
    expect(within(index).getByRole('button', { name: 'Gateway, public.md' })).toBeInTheDocument()
    expect(within(index).getByRole('button', { name: 'Gateway, private.md' })).toBeInTheDocument()
    expect(within(index).getByRole('button', { name: 'Worker' })).toBeInTheDocument()
    expect(within(index).queryByText('worker.md')).not.toBeInTheDocument()
    expect(screen.getByText('Public body.')).toBeInTheDocument()

    const user = userEvent.setup()
    await user.click(within(index).getByRole('button', { name: 'Gateway, private.md' }))
    expect(screen.getByText('Private body.')).toBeInTheDocument()
    expect(graphHarness.calls.at(-1)?.elements).toHaveLength(5)

    act(() => graphHarness.nodeSelect?.({ target: { id: () => 'worker' } }))
    expect(screen.getByRole('heading', { name: 'Worker' })).toBeInTheDocument()
    expect(screen.getByText('Worker body.')).toBeInTheDocument()
  })

  it('clears accepted documentation selection to a neutral contextual pane', async () => {
    mockResponses([{
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '7'.repeat(40),
      component_count: 1, component_titles: ['Gateway'],
      components: [{ id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Gateway body.\n', relationships: [] }],
    }])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Clear selection' }))

    expect(screen.getByRole('heading', { name: 'Select a component' })).toBeInTheDocument()
    expect(screen.queryByText('Gateway body.')).not.toBeInTheDocument()
    expect(within(screen.getByRole('navigation', { name: 'Components' })).getByRole('button', { name: 'Gateway' })).not.toHaveAttribute('aria-current')
  })

  it('authors ordered outgoing relationships with backend-supplied identity choices', async () => {
    const accepted = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '7'.repeat(40),
      component_count: 4, component_titles: ['Gateway', 'Worker', 'Records', 'Records'],
      components: [
        { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Gateway body.\n', relationships: [
          { target_id: 'worker', label: '  calls: primary  ' },
          { target_id: 'records-a', label: 'reads from' },
        ] },
        { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Worker body.\n', relationships: [] },
        { id: 'records-a', title: 'Records', filename: 'records-a.md', description: 'A.\n', relationships: [] },
        { id: 'records-b', title: 'Records', filename: 'records-b.md', description: 'B.\n', relationships: [] },
      ],
      changes: {
        valid: true,
        components: [{ id: 'queue', title: 'Queue', description: 'Pending.\n', relationships: [], new: true }],
        relationship_targets: [
          { id: 'gateway', title: 'Gateway' },
          { id: 'worker', title: 'Worker' },
          { id: 'records-a', title: 'Records', context: 'records-a.md' },
          { id: 'records-b', title: 'Records', context: 'records-b.md' },
          { id: 'queue', title: 'Queue', new: true },
        ],
      },
    }
    const kept = {
      ...accepted,
      changes: {
        ...accepted.changes,
        components: [
          ...accepted.changes.components,
          { id: 'gateway', title: 'Gateway', description: 'Gateway body.\n', new: false, relationships: [
            { target_id: 'worker', label: '  calls: primary  ' },
            { target_id: 'queue', label: 'publishes\n events' },
          ] },
        ],
      },
    }
    const fetchMock = mockResponses([accepted, kept])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Gateway' }))
    await user.click(screen.getByRole('button', { name: 'Edit component' }))
    const relationships = screen.getByRole('group', { name: 'Outgoing relationships' })
	expect(within(relationships).getAllByRole('option', { name: 'Queue — New component' })).toHaveLength(2)
	expect(within(relationships).getAllByRole('option', { name: 'Records — records-a.md' })).toHaveLength(2)
	expect(within(relationships).getAllByRole('option', { name: 'Records — records-b.md' })).toHaveLength(2)
	expect(within(relationships).getAllByRole('option', { name: 'Worker' })).toHaveLength(2)
    expect(within(relationships).queryByText(/gateway\.md|worker\.md/)).not.toBeInTheDocument()

    await user.click(within(relationships).getByRole('button', { name: 'Remove relationship 2' }))
    await user.click(within(relationships).getByRole('button', { name: 'Add relationship' }))
    const targets = within(relationships).getAllByLabelText('Target')
    const labels = within(relationships).getAllByLabelText('Label')
    await user.selectOptions(targets[1], 'queue')
    await user.type(labels[1], 'publishes{enter} events')
    await user.click(screen.getByRole('button', { name: 'Keep change' }))

    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/components/edit')
    expect(requestBody(fetchMock, 1)).toEqual({
      source_root: '/tmp/example', expected_revision: '7'.repeat(40), component_id: 'gateway', relationships_changed: true,
      relationships: [
        { target_id: 'worker', label: '  calls: primary  ' },
        { target_id: 'queue', label: 'publishes\n events' },
      ],
      title_changed: false, description_changed: false,
    })
    expect(graphHarness.calls.at(-1)?.elements).toHaveLength(6)
    expect(graphHarness.calls.at(-1)?.elements).toEqual(expect.arrayContaining([
      expect.objectContaining({ data: expect.objectContaining({ id: 'gateway', label: 'Gateway' }) }),
	  expect.objectContaining({ data: expect.objectContaining({ id: 'projection:gateway:worker:0', source: 'gateway', target: 'worker', label: '  calls: primary  ', distance: 0 }) }),
    ]))
    expect(graphHarness.calls.at(-1)?.elements).not.toEqual(expect.arrayContaining([{ data: expect.objectContaining({ id: 'queue' }) }]))
  })

  it.each([
    {
      code: 'relationship_label_required',
      field: 'label',
      message: 'Add a label to this relationship.',
      sourceID: 'gateway',
      sourceName: 'Gateway',
      components: [{ id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Body.\n', relationships: [] }],
      targets: [{ id: 'gateway', title: 'Gateway' }],
      relationship: { target_id: 'gateway', label: '' },
    },
    {
      code: 'relationship_target_required',
      field: 'target',
      message: 'Choose a component for this relationship.',
      sourceID: 'gateway-b',
      sourceName: 'Gateway — private.md',
      components: [
        { id: 'gateway-a', title: 'Gateway', filename: 'public.md', description: 'Public.\n', relationships: [] },
        { id: 'gateway-b', title: 'Gateway', filename: 'private.md', description: 'Private.\n', relationships: [] },
      ],
      targets: [
        { id: 'gateway-a', title: 'Gateway', context: 'public.md' },
        { id: 'gateway-b', title: 'Gateway', context: 'private.md' },
      ],
      relationship: { target_id: '', label: 'calls' },
    },
  ])('localizes $code to its component and field while retaining relationship work', async ({
    code, field, message, sourceID, sourceName, components, targets, relationship,
  }) => {
    const pending = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '7'.repeat(40),
      component_count: components.length, component_titles: components.map((component) => component.title), components,
      changes: {
        valid: false, validation_code: code, validation_item: sourceID,
        components: [{
          id: sourceID,
          title: 'Gateway',
          description: sourceID === 'gateway-b' ? 'Private.\n' : 'Body.\n',
          relationships: [relationship],
          new: false,
        }],
        relationship_targets: targets,
      },
    }
    const blocked = {
      ...pending,
      action_error: 'review_failed',
      changes: {
        ...pending.changes,
        review_blocker: code,
        validation_relationship_position: 1,
        validation_relationship_field: field,
      },
    }
    mockResponses([pending, blocked], [200, 422])
    render(<App />)
    await submitPath('/tmp/example')

    expect(screen.queryByText(message)).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Review changes' }))
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(sourceName)
    expect(alert).toHaveTextContent(message)
    expect(alert).not.toHaveTextContent('before updating architecture')
    const owningRow = screen.getByText('Needs attention').closest('li') as HTMLElement
    expect(owningRow).toHaveAttribute('aria-invalid', 'true')
    const fixAction = within(alert).getByRole('button', { name: 'Fix relationship' })
    expect(fixAction).toHaveClass('inline-action')
    await user.click(fixAction)

    const fieldControl = screen.getByLabelText(field === 'label' ? 'Label' : 'Target')
    expect(fieldControl).toHaveAttribute('aria-invalid', 'true')
    expect(fieldControl).toHaveFocus()
    const guidanceID = fieldControl.getAttribute('aria-describedby')
    expect(guidanceID).toBeTruthy()
    expect(document.getElementById(guidanceID!)).toHaveTextContent(message)
    expect(screen.getByRole('button', { name: 'Keep change' })).toBeInTheDocument()
    expect(screen.queryByText(/yaml|frontmatter|uuid|parser|candidate|\bref\b/i)).not.toBeInTheDocument()
  })

  it('guards unsent relationship fields before workbench navigation replaces the editor', async () => {
    const workspace = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '9'.repeat(40), component_count: 2,
      component_titles: ['Gateway', 'Worker'],
      components: [
        { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted.\n', relationships: [] },
        { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Works.\n', relationships: [] },
      ],
    }
    mockResponses([workspace])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    await user.click(screen.getByRole('button', { name: 'Add relationship' }))
    await user.selectOptions(screen.getByLabelText('Target'), 'worker')
    await user.type(screen.getByLabelText('Label'), 'calls')

    await user.click(screen.getByRole('button', { name: 'Worker' }))
    expect(screen.getByRole('dialog')).toHaveTextContent('Leave without keeping?')
    await user.click(screen.getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByLabelText('Target')).toHaveValue('worker')
    expect(screen.getByLabelText('Label')).toHaveValue('calls')

    await user.click(screen.getByRole('button', { name: 'Worker' }))
    await user.click(screen.getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('heading', { name: 'Worker' })).toBeInTheDocument()
    expect(screen.queryByDisplayValue('calls')).not.toBeInTheDocument()
  })

  it('requires confirmation and discards the whole backend-held change set through one action', async () => {
    const pending = {
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '8'.repeat(40),
      component_count: 0, component_titles: [], components: [],
      changes: { valid: true, components: [{ id: 'worker', title: 'Worker', description: 'Body.\n', new: true }] },
    }
    const discarded = { ...pending, changes: undefined }
    const fetchMock = mockResponses([pending, discarded])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Discard changes' }))
    const confirmation = screen.getByRole('dialog', { name: 'Discard changes?' })
    expect(confirmation).toHaveTextContent('This clears every change in progress. The accepted architecture will not change.')
    await user.click(within(confirmation).getByRole('button', { name: 'Discard changes' }))

    expect(await screen.findByRole('heading', { name: 'Start with a component' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Changes in progress/ })).not.toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/discard')
    expect(requestBody(fetchMock, 1)).toEqual({ source_root: '/tmp/example' })
  })

  it('leaves the workspace for project opening only after backend eligibility succeeds', async () => {
    const ready = {
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '8'.repeat(40),
      component_count: 0, component_titles: [], components: [],
    }
    let call = 0
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
      if (call++ === 0) return new Response(JSON.stringify(ready), { status: 200, headers: { 'Content-Type': 'application/json' } })
      return new Response(null, { status: 204 })
    })
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Open another project' }))

    expect(await screen.findByRole('heading', { name: 'Open a project' })).toBeInTheDocument()
    expect(screen.queryByRole('navigation', { name: 'Components' })).not.toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/projects/leave')
  })

  it('keeps the current workspace when backend-held changes block project switching', async () => {
    const review = testReview({
      base: '8'.repeat(40), candidate: '9'.repeat(40), generation: 2,
      diff: 'diff --git a/components/worker.md b/components/worker.md\n+# Worker\n',
      withChanges: [{ id: 'worker', title: 'Worker', filename: 'worker.md', description: '', relationships: [] }],
      componentChanges: [{ component_id: 'worker', status: 'added', path: 'components/worker.md' }],
    })
    const pending = {
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '8'.repeat(40),
      component_count: 0, component_titles: [], components: [],
      changes: { valid: true, components: [{ id: 'worker', title: 'Worker', description: '', new: true }], review },
    }
    const blocked = { ...pending, action_error: 'pending_blocks_switch' }
    const fetchMock = mockResponses([pending, blocked, blocked], [200, 409, 409])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Open another project' }))

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    const notice = screen.getByRole('alert')
    expect(notice).toHaveTextContent('Keep working here or discard these changes before opening another project.')
    expect(notice.parentElement).toHaveClass('workspace-shell')
    expect(notice.nextElementSibling).toHaveClass('architecture-workbench')
    expect(screen.getByText('Worker')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Update architecture' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Discard changes' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Open a project' })).not.toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/projects/leave')

    await user.click(screen.getByRole('button', { name: 'Dismiss message' }))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Update architecture' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Discard changes' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(2)

    await user.click(screen.getByRole('button', { name: 'Open another project' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Keep working here or discard these changes before opening another project.')
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('restores backend-held changes when opening another project is blocked after a browser reload', async () => {
    const pending = {
      source_root: '/tmp/project-a', project_name: 'project-a', state: 'empty', revision: '8'.repeat(40),
      component_count: 0, component_titles: [], components: [],
      changes: { valid: true, components: [{ id: 'worker', title: 'Worker', description: '', new: true }] },
      action_error: 'pending_blocks_switch',
    }
    const fetchMock = mockResponses([pending], [409])
    render(<App />)

    await submitPath('/tmp/project-b')

    expect(await screen.findByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('Keep working here or discard these changes before opening another project.')
    expect(screen.getByText('Worker')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Discard changes' })).toBeInTheDocument()
    expect(screen.getByText('/tmp/project-a')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Open a project' })).not.toBeInTheDocument()
    expect(requestPath(fetchMock, 0)).toBe('/api/projects/open')
    expect(requestBody(fetchMock, 0)).toEqual({ source_root: '/tmp/project-b' })
  })

  it('guards dirty editor values before Add component replaces the editor', async () => {
    const workspace = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '9'.repeat(40), component_count: 1,
      component_titles: ['Gateway'],
      components: [{ id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted.\n', relationships: [] }],
    }
    const fetchMock = mockResponses([workspace])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    await user.type(screen.getByLabelText('Title'), ' locally changed')

    await user.click(screen.getByRole('button', { name: 'Add component' }))
    expect(screen.getByRole('dialog')).toHaveTextContent('Leave without keeping?')
    await user.click(screen.getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('heading', { name: 'Edit component' })).toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toHaveValue('Gateway locally changed')

    await user.click(screen.getByRole('button', { name: 'Add component' }))
    await user.click(screen.getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.getByRole('heading', { name: 'Add component' })).toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toHaveValue('')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it.each([
    ['index selection', async (user: ReturnType<typeof userEvent.setup>) => user.click(screen.getByRole('button', { name: 'Worker' }))],
    ['map selection', async () => act(() => graphHarness.nodeSelect?.({ target: { id: () => 'worker' } }))],
    ['Changes in progress', async (user: ReturnType<typeof userEvent.setup>) => user.click(screen.getByRole('button', { name: /Changes in progress/ }))],
    ['Open another project', async (user: ReturnType<typeof userEvent.setup>) => user.click(screen.getByRole('button', { name: 'Open another project' }))],
  ])('guards dirty editor values before %s replaces the task', async (_label, navigate) => {
    const workspace = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: '9'.repeat(40), component_count: 2,
      component_titles: ['Gateway', 'Worker'],
      components: [
        { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted.\n', relationships: [] },
        { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Works.\n', relationships: [] },
      ],
      changes: { valid: true, components: [{ id: 'gateway', title: 'Pending Gateway', description: 'Pending.\n', new: false }] },
    }
    mockResponses([workspace, { ...workspace, action_error: 'pending_blocks_switch' }], [200, 409])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    const pendingItem = screen.getByText('Pending Gateway').closest('li') as HTMLElement
    await user.click(within(pendingItem).getByRole('button', { name: 'Edit' }))
    await user.type(screen.getByLabelText('Title'), ' locally changed')

    await navigate(user)
    expect(screen.getByRole('dialog')).toHaveTextContent('Leave without keeping?')
    await user.click(screen.getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByLabelText('Title')).toHaveValue('Pending Gateway locally changed')

    await navigate(user)
    await user.click(screen.getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue('Pending Gateway locally changed')).not.toBeInTheDocument()
  })

  it('guards unsent editor values when a delayed review response tries to replace the task', async () => {
    const base = '4'.repeat(40)
    const candidate = '5'.repeat(40)
    const before = [
      { id: 'gateway', title: 'Gateway', filename: 'gateway.md', description: 'Accepted.\n', relationships: [] },
      { id: 'worker', title: 'Worker', filename: 'worker.md', description: 'Works.\n', relationships: [] },
    ]
    const pending = {
      source_root: '/tmp/example', project_name: 'example', state: 'ready', revision: base,
      component_count: 2, component_titles: ['Gateway', 'Worker'], components: before,
      changes: {
        valid: true,
        components: [{ id: 'gateway', title: 'Pending Gateway', filename: 'gateway.md', description: 'Pending.\n', relationships: [], new: false }],
      },
    }
    const reviewed = {
      ...pending,
      changes: {
        ...pending.changes,
        review: testReview({
          base, candidate,
          before,
          withChanges: [{ ...before[0], title: 'Pending Gateway', description: 'Pending.\n' }, before[1]],
          diff: 'diff --git a/components/gateway.md b/components/gateway.md\n-# Gateway\n+# Pending Gateway\n',
          componentChanges: [{ component_id: 'gateway', status: 'content_changed', path: 'components/gateway.md' }],
        }),
      },
    }
    let finishReview: ((response: Response) => void) | undefined
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(pending), { status: 200 }))
    fetchMock.mockImplementationOnce(() => new Promise<Response>((resolve) => { finishReview = resolve }))
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Review changes' }))
    expect(screen.getByRole('button', { name: 'Preparing…' })).toBeDisabled()
    const pendingRow = screen.getByText('Pending Gateway').closest('li') as HTMLElement
    await user.click(within(pendingRow).getByRole('button', { name: 'Edit' }))
    await user.type(screen.getByLabelText('Title'), ' locally changed')
    await user.clear(screen.getByLabelText('Description'))
    await user.type(screen.getByLabelText('Description'), 'Unsent body.')
    await user.click(screen.getByRole('button', { name: 'Add relationship' }))
    await user.selectOptions(screen.getByLabelText('Target'), 'worker')
    await user.type(screen.getByLabelText('Label'), 'unsent calls')

    await act(async () => {
      finishReview?.(new Response(JSON.stringify(reviewed), { status: 200 }))
    })
    expect(await screen.findByRole('dialog')).toHaveTextContent('Leave without keeping?')
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Keep editing' }))

    expect(screen.getByLabelText('Title')).toHaveValue('Pending Gateway locally changed')
    expect(screen.getByLabelText('Description')).toHaveValue('Unsent body.')
    expect(screen.getByLabelText('Target')).toHaveValue('worker')
    expect(screen.getByLabelText('Label')).toHaveValue('unsent calls')
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    expect(requestPath(fetchMock, 1)).toBe('/api/architecture/review')
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('requires a fresh review when pending changes mutate after the displayed review', async () => {
    const reviewed = {
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '1'.repeat(40),
      component_count: 0, component_titles: [], components: [],
      changes: {
        valid: true,
        components: [{ id: 'worker-id', title: 'Worker', description: '', new: true }],
        review: testReview({
          base: '1'.repeat(40), candidate: '2'.repeat(40), diff: 'diff --git a/components/worker.md b/components/worker.md',
          withChanges: [{ id: 'worker-id', title: 'Worker', description: '', relationships: [] }],
          componentChanges: [{ component_id: 'worker-id', status: 'added', path: 'components/worker-id.md' }],
        }),
      },
    }
    const changed = { ...reviewed, action_error: 'review_changed', changes: { valid: true, components: reviewed.changes.components } }
    mockResponses([reviewed, changed], [200, 409])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Update architecture' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('The changes were edited after this review. Review them again before updating architecture.')
    expect(screen.getByText('Worker')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Review changes' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
  })

  it.each(['lost response', 'unreadable response body'])(
    'does not offer a duplicate update after an ambiguous %s',
    async (failure) => {
      const reviewed = {
        source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: '1'.repeat(40),
        component_count: 0, component_titles: [], components: [],
        changes: {
          valid: true,
          components: [{ id: 'worker-id', title: 'Worker', description: '', new: true }],
          review: testReview({
            base: '1'.repeat(40), candidate: '2'.repeat(40), diff: 'diff --git a/components/worker.md b/components/worker.md',
            withChanges: [{ id: 'worker-id', title: 'Worker', description: '', relationships: [] }],
            componentChanges: [{ component_id: 'worker-id', status: 'added', path: 'components/worker-id.md' }],
          }),
        },
      }
      let call = 0
      const fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
        if (call++ === 0) {
          return new Response(JSON.stringify(reviewed), { status: 200, headers: { 'Content-Type': 'application/json' } })
        }
        if (failure === 'lost response') throw new Error('response lost after request')
        return new Response('{not-json', { status: 200, headers: { 'Content-Type': 'application/json' } })
      })
      render(<App />)
      await submitPath('/tmp/example')
      const user = userEvent.setup()
      await user.click(await screen.findByRole('button', { name: 'Update architecture' }))

      const alert = await screen.findByRole('alert')
      expect(alert).toHaveTextContent('WorkBraid could not confirm what happened. Open this project again to check its current architecture.')
      expect(alert).not.toHaveTextContent(/try again/i)
      expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Review changes' })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: /add component|edit/i })).not.toBeInTheDocument()
      expect(screen.queryByRole('button', { name: 'Discard changes' })).not.toBeInTheDocument()
      expect(fetchMock).toHaveBeenCalledTimes(2)
      expect(requestBody(fetchMock, 1)).toEqual({
        source_root: '/tmp/example', base_revision: '1'.repeat(40), candidate_tree: '2'.repeat(40), generation: 1,
      })
    },
  )

  it.each([
    ['architecture_unavailable', 409, 'Architecture unavailable', 'could not open the architecture linked to this project'],
    ['architecture_invalid', 409, 'Architecture needs attention', "could not read this project's architecture"],
    ['architecture_unsupported', 422, 'Architecture not supported yet', 'uses features that this version of WorkBraid cannot open yet'],
  ])('shows open failure %s in product language without recovery controls', async (code, status, heading, message) => {
    mockResponses([{ code }], [status])
    render(<App />)
    await submitPath('/tmp/example')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(heading)
    expect(alert).toHaveTextContent(message)
    expect(alert).toHaveTextContent('/tmp/example')
    expect(screen.queryByText('This project has an empty architecture.')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /retry|repair|reset|create|set up/i })).not.toBeInTheDocument()
    expect(alert.textContent ?? '').not.toMatch(/\b(association|store|manifest|accepted ref|canonical|snapshot|uuid|git object)\b/i)
  })

  it('keeps an incomplete setup retryable in the same running application', async () => {
    const fetchMock = mockResponses([
      unlinkedProject,
      { code: 'setup_incomplete' },
      { source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: 'c'.repeat(40), component_count: 0 },
    ], [200, 500, 200])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Set up architecture' }))
    await user.click(screen.getByRole('button', { name: 'Set up' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Setup did not finish')
    expect(alert).toHaveTextContent('WorkBraid could not finish setting up architecture. Try again.')
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByRole('heading', { name: 'Start with a component' })).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(3)
  })

  it('shows concise setup progress after confirmation', async () => {
    let finishSetup: ((response: Response) => void) | undefined
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(unlinkedProject), { status: 200 }))
    fetchMock.mockImplementationOnce(() => new Promise<Response>((resolve) => { finishSetup = resolve }))
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Set up architecture' }))
    await user.click(screen.getByRole('button', { name: 'Set up' }))

    expect(screen.getByText('Setting up architecture…')).toBeInTheDocument()
    finishSetup?.(new Response(JSON.stringify({
      source_root: '/tmp/example', project_name: 'example', state: 'empty', revision: 'd'.repeat(40), component_count: 0,
    }), { status: 200 }))
    expect(await screen.findByRole('heading', { name: 'Start with a component' })).toBeInTheDocument()
  })

  it('keeps implementation terminology out of normal setup copy', async () => {
    mockResponses([unlinkedProject])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Set up architecture' }))

    const sheet = screen.getByRole('article').textContent ?? ''
    expect(sheet).not.toMatch(/\b(store|association|bootstrap|manifest|canonical|uuid|accepted ref|accepted revision)\b/i)
  })

  it.each([
    ['architecture_invalid', 409, 'Architecture needs attention', "could not read this project's architecture"],
    ['architecture_unsupported', 422, 'Architecture not supported yet', 'uses features that this version of WorkBraid cannot open yet'],
  ])('shows %s clearly without pretending an empty architecture loaded', async (code, status, heading, message) => {
    mockResponses([unlinkedProject, { code }], [200, status])
    render(<App />)
    await submitPath('/tmp/example')
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Set up architecture' }))
    await user.click(screen.getByRole('button', { name: 'Set up' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(heading)
    expect(alert).toHaveTextContent(message)
    expect(screen.queryByText('This project has an empty architecture.')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })

  it.each([
    ['path_required', 'Enter a folder path.'],
    ['path_relative', 'Use a full path, starting with /.'],
    ['path_missing', 'That folder is not on this computer.'],
    ['path_not_directory', 'That path is a file. Choose the project folder.'],
    ['origin_mismatch', 'Open WorkBraid at the address printed in the terminal.'],
  ])('maps backend code %s to an operator sentence', async (code, message) => {
    mockResponses([{ code }], [400])
    render(<App />)

    await submitPath(code === 'path_required' ? '' : '/tmp/example')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('That path did not work')
    expect(alert).toHaveTextContent(message)
  })

  it('maps unknown response shapes and network failures to one generic sentence', async () => {
    mockResponses([{ error: 'request body must be valid JSON' }], [400])
    const { unmount } = render(<App />)

    await submitPath('/tmp/example')
    expect(await screen.findByRole('alert')).toHaveTextContent("WorkBraid couldn't look that up. Try again.")

    unmount()
    vi.restoreAllMocks()
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('backend unavailable'))
    render(<App />)

    await submitPath('/tmp/example')
    expect(await screen.findByRole('alert')).toHaveTextContent("WorkBraid couldn't look that up. Try again.")
    expect(screen.queryByText('backend unavailable')).not.toBeInTheDocument()
  })

  it('shows concise progress for lookup and setup', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => new Promise<Response>(() => undefined))
    render(<App />)

    await submitPath('/tmp/example')

    expect(screen.getByRole('button', { name: 'Looking up…' })).toBeDisabled()
    expect(screen.getByText('Looking up this folder…')).toBeInTheDocument()
  })
})

function mockResponses(bodies: unknown[], statuses: number[] = []) {
  let index = 0
  return vi.spyOn(globalThis, 'fetch').mockImplementation(async () => {
    const current = index++
    return new Response(JSON.stringify(bodies[current]), {
      status: statuses[current] ?? 200,
      headers: { 'Content-Type': 'application/json' },
    })
  })
}

function requestPath(fetchMock: ReturnType<typeof mockResponses>, index: number) {
  return String(fetchMock.mock.calls[index]?.[0])
}

function requestBody(fetchMock: ReturnType<typeof mockResponses>, index: number) {
  const options = fetchMock.mock.calls[index]?.[1]
  return JSON.parse(String(options?.body)) as unknown
}

async function submitPath(path: string) {
  const user = userEvent.setup()
  const input = screen.getByLabelText('Project folder')
  if (path) {
    await user.type(input, path)
  }
  await user.click(screen.getByRole('button', { name: 'Open' }))
}
