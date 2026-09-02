import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { App } from './App'

const graphHarness = vi.hoisted(() => ({
  calls: [] as Array<{ elements?: Array<{ data: Record<string, unknown> }> }>,
  nodeSelect: undefined as undefined | ((event: { target: { id: () => string } }) => void),
  edgeSelect: undefined as undefined | ((event: { target: { data: () => unknown } }) => void),
  fail: false,
}))

vi.mock('cytoscape', () => ({
  default: (options: { elements?: Array<{ data: Record<string, unknown> }> }) => {
    if (graphHarness.fail) throw new Error('canvas unavailable')
    graphHarness.calls.push(options)
    return {
      on: (_event: string, selector: string | (() => void), callback?: unknown) => {
        if (selector === 'node') graphHarness.nodeSelect = callback as typeof graphHarness.nodeSelect
        if (selector === 'edge') graphHarness.edgeSelect = callback as typeof graphHarness.edgeSelect
      },
      off: () => undefined,
      nodes: () => [],
      resize: () => undefined,
      destroy: () => undefined,
      fit: () => undefined,
      $: () => ({ unselect: () => undefined }),
      getElementById: () => ({ select: () => undefined }),
    }
  },
}))

const root = '11111111-1111-4111-8111-111111111111'
const worker = '22222222-2222-4222-8222-222222222222'
const external = '33333333-3333-4333-8333-333333333333'
const detail = '44444444-4444-4444-8444-444444444444'

function response(value: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } }))
}

function architecture(overrides: Record<string, unknown> = {}) {
  return {
    project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', project_name: 'Example Project',
    state: 'ready', revision: 'a'.repeat(40), format_version: 2, component_count: 2,
    component_titles: ['Worker', 'External'], root_diagram_id: root,
    components: [
      { id: worker, title: 'Worker', description: 'Does work.\n', filename: 'worker.md', relationships: [{ target_id: external, label: 'calls' }] },
      { id: external, title: 'External', description: 'Elsewhere.\n', filename: 'external.md', relationships: [] },
    ],
    diagrams: [{
      id: root, title: 'System', filename: 'root.yaml', depth: 0, breadcrumbs: [{ id: root, title: 'System' }],
      appearances: [{ component_id: worker, role: 'home', detail_diagram_id: detail, detail_diagram_title: 'Detail' }],
      boundaries: [{ key: `boundary:${external}`, component_id: external, title: 'External', home_diagram_id: detail, home_diagram_title: 'Detail' }],
      relationships: [{ key: 'edge', source_node_key: worker, target_node_key: `boundary:${external}`, source_component_id: worker, target_component_id: external, label: 'calls' }],
    }, {
      id: detail, title: 'Detail', filename: 'detail.yaml', depth: 1, parent_diagram_id: root, parent_anchor_component_id: worker,
      breadcrumbs: [{ id: root, title: 'System' }, { id: detail, title: 'Detail' }],
      appearances: [{ component_id: external, role: 'home' }], boundaries: [], relationships: [],
    }],
    home_move_destinations: [{ component_id: worker, current_home_id: root, diagram_ids: [] }, { component_id: external, current_home_id: detail, diagram_ids: [root] }],
    reference_choices: [{ diagram_id: root, component_id: external, title: 'External', home_diagram: 'Detail' }],
    ...overrides,
  }
}

function requestBody(mock: ReturnType<typeof vi.fn>, index: number) {
  return JSON.parse(String(mock.mock.calls[index][1]?.body))
}

function reviewedArchitecture(overrides: Record<string, unknown> = {}) {
  const accepted = architecture()
  const beforeComponents = (accepted.components as Array<Record<string, unknown>>).map((component) => ({ ...component }))
  const withComponents = beforeComponents.map((component) => component.id === worker
    ? { ...component, title: 'Worker updated', description: 'Candidate documentation.\n', relationships: [{ target_id: external, label: 'invokes', projection_key: 'edge-with' }] }
    : { ...component })
  const beforeDiagrams = accepted.diagrams as Array<Record<string, unknown>>
  const withDiagrams = beforeDiagrams.map((diagram) => diagram.id === root ? {
    ...diagram,
    appearances: [{ component_id: worker, role: 'home', detail_diagram_id: detail, detail_diagram_title: 'Detail' }, { component_id: external, role: 'reference' }],
    boundaries: [],
    relationships: [{ key: 'edge-with', source_node_key: worker, target_node_key: external, source_component_id: worker, target_component_id: external, label: 'invokes' }],
  } : diagram)
  const diff = [
    'diff --git a/components/worker.md b/components/worker.md',
    '--- a/components/worker.md',
    '+++ b/components/worker.md',
    '@@ -1 +1 @@',
    '-Accepted documentation.',
    '+Candidate documentation.',
  ].join('\n') + '\n'
  return architecture({
    changes: {
      components: [{ id: worker, title: 'Worker updated', description: 'Candidate documentation.\n', new: false, relationships: [{ target_id: external, label: 'invokes' }] }],
      valid: true,
      candidate: { revision: 'b'.repeat(40), format_version: 2, component_count: 2, component_titles: ['Worker updated', 'External'], components: withComponents, root_diagram_id: root, diagrams: withDiagrams },
      review: {
        diff, base_revision: 'a'.repeat(40), candidate_tree: 'b'.repeat(40), generation: 4,
        before: { revision: 'a'.repeat(40), format_version: 2, component_count: 2, component_titles: ['Worker', 'External'], components: beforeComponents, root_diagram_id: root, diagrams: beforeDiagrams },
        with_changes: { revision: 'b'.repeat(40), format_version: 2, component_count: 2, component_titles: ['Worker updated', 'External'], components: withComponents, root_diagram_id: root, diagrams: withDiagrams },
        comparison: {
          components: [{ component_id: worker, status: 'content_changed', path: 'components/worker.md' }],
          relationships: [
            { key: 'removed-edge', before_key: 'edge', source_id: worker, target_id: external, label: 'calls', status: 'removed', path: 'components/worker.md', occurrence: 1, diagram_projections: [{ side: 'before', diagram_id: root, key: 'edge', source_node_key: worker, target_node_key: `boundary:${external}` }] },
            { key: 'edge-with', source_id: worker, target_id: external, label: 'invokes', status: 'added', path: 'components/worker.md', occurrence: 1, diagram_projections: [{ side: 'with', diagram_id: root, key: 'edge-with', source_node_key: worker, target_node_key: external }] },
          ],
          diagrams: [], appearances: [{ diagram_id: root, component_id: external, role: 'reference', status: 'added', path: 'diagrams/root.yaml' }],
        },
      },
    },
    ...overrides,
  })
}

beforeEach(() => {
  window.history.replaceState({}, '', '/')
  vi.restoreAllMocks()
})

afterEach(() => {
  cleanup()
  graphHarness.calls.length = 0
  graphHarness.nodeSelect = undefined
  graphHarness.edgeSelect = undefined
  graphHarness.fail = false
})

describe('slug-native project entry', () => {
  it('lists private projects and creates a project by name', async () => {
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response({ projects: [] }))
      .mockImplementationOnce(() => response(architecture({ state: 'empty', component_count: 0, component_titles: [], components: [], reference_choices: [], diagrams: [{ id: root, title: 'My Project', filename: 'root.yaml', depth: 0, breadcrumbs: [{ id: root, title: 'My Project' }], appearances: [], boundaries: [], relationships: [] }] }), 201))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    const emptyCatalog = await screen.findByText('No projects yet.')
    const createForm = screen.getByLabelText('New project').closest('form')
    expect(emptyCatalog).toBeInTheDocument()
    expect(createForm).not.toBeNull()
    expect(createForm!.compareDocumentPosition(emptyCatalog) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(screen.queryByText(/folder/i)).not.toBeInTheDocument()
    await user.type(screen.getByLabelText('New project'), 'My Project')
    await user.click(screen.getByRole('button', { name: 'Create project' }))
    expect(await screen.findByRole('button', { name: 'My Project' })).toBeInTheDocument()
    expect(requestBody(fetchMock, 1)).toEqual({ name: 'My Project' })
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('shows collision context and opens by slug', async () => {
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response({ projects: [
        { name: 'Example', slug: 'example', store_id: 'a' },
        { name: 'Example', slug: 'example-2', store_id: 'b' },
      ] }))
      .mockImplementationOnce(() => response(architecture({ project_name: 'Example', project_slug: 'example-2' })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    const slugContext = await screen.findByText('example-2')
    const createForm = screen.getByLabelText('New project').closest('form')
    const catalog = slugContext.closest('nav')
    expect(createForm).not.toBeNull()
    expect(catalog).not.toBeNull()
    expect(createForm!.compareDocumentPosition(catalog!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    await user.click(screen.getByRole('button', { name: 'Example, example-2' }))
    await screen.findByText('Architecture')
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-2' })
    expect(window.location.pathname).toBe('/projects/example-2')
  })

  it('restores a direct slug route and shows normal not found state', async () => {
    window.history.replaceState({}, '', '/projects/missing')
    const fetchMock = vi.fn(() => response({ code: 'project_not_found' }, 404))
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Project not found' })).toBeInTheDocument()
    expect(requestBody(fetchMock, 0)).toEqual({ project_slug: 'missing' })
  })

  it.each([
    [{ name: 'Broken', store_id: 'a', unavailable: true }, 'Project unavailable'],
    [{ name: 'Conflict', slug: 'same', store_id: 'b', conflict: true }, 'Project address conflict'],
  ])('renders catalog authority state %#', async (project, expected) => {
    vi.stubGlobal('fetch', vi.fn(() => response({ projects: [project] })))
    render(<App />)
    expect(await screen.findByText(expected)).toBeInTheDocument()
  })
})

describe('slug workspace and reusable references', () => {
  it('adopts an externally accepted slug on Refresh and replaces the route', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response(architecture({ project_slug: 'new-locator', revision: 'b'.repeat(40) })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(window.location.pathname).toBe('/projects/new-locator'))
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa' })
  })

  it('shows a candidate-relative component through structured controls', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const pending = architecture({
      changes: { components: [], valid: true, candidate: architecture(), references: [{ diagram_id: root, component_id: external, present: true }] },
      diagrams: (architecture().diagrams as unknown[]),
    })
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response(pending))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    const picker = (await screen.findAllByLabelText('Show component here'))[0]
    await user.selectOptions(picker, external)
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', expected_revision: 'a'.repeat(40), diagram_id: root, component_id: external })
    expect(await screen.findByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
  })

  it('offers Stop showing here only for a canonical reference', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const value = architecture({
      diagrams: [{
        id: root, title: 'System', filename: 'root.yaml', depth: 0, breadcrumbs: [{ id: root, title: 'System' }],
        appearances: [{ component_id: worker, role: 'home', detail_diagram_id: detail, detail_diagram_title: 'Detail' }, { component_id: external, role: 'reference' }], boundaries: [],
        relationships: [{ key: 'edge', source_node_key: worker, target_node_key: external, source_component_id: worker, target_component_id: external, label: 'calls' }],
      }, {
        id: detail, title: 'Detail', filename: 'detail.yaml', depth: 1, parent_diagram_id: root, parent_anchor_component_id: worker,
        breadcrumbs: [{ id: root, title: 'System' }, { id: detail, title: 'Detail' }],
        appearances: [{ component_id: external, role: 'home' }], boundaries: [], relationships: [],
      }], reference_choices: [],
    })
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(value))
      .mockImplementationOnce(() => response(architecture({ changes: { components: [], valid: true, candidate: architecture() } })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    const index = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(index).getByRole('button', { name: /External, Included here/ }))
    await user.click(screen.getByRole('button', { name: 'Stop showing here' }))
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', expected_revision: 'a'.repeat(40), diagram_id: root, component_id: external })
  })

  it('keeps folder and setup language out of a writable workspace', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture())))
    render(<App />)
    expect(await screen.findByText('Example Project', { selector: '.workspace-context strong' })).toBeInTheDocument()
    expect(screen.queryByText(/Set up diagrams/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Project folder/i)).not.toBeInTheDocument()
    expect(screen.getByText('Project slug')).toBeInTheDocument()
  })

  it('protects browser-local dirty edits before returning to the catalog', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture())))
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    await user.type(screen.getByLabelText('Description'), ' unsent')
    await user.click(screen.getByRole('button', { name: 'Open another project' }))
    expect(screen.getByRole('dialog', { name: 'Leave without keeping?' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Keep editing' })).toBeInTheDocument()
  })

  it('keeps backend-held pending work visible when project leave is blocked', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const pending = architecture({ changes: { components: [{ id: worker, title: 'Worker', description: 'Changed.\n', new: false, relationships: [] }], valid: true, candidate: architecture() } })
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(pending))
      .mockImplementationOnce(() => response({ ...pending, action_error: 'pending_blocks_switch' }, 409))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: 'Open another project' }))
    expect(await screen.findByText('Keep working here or discard these changes before opening another project.')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Changes in progress' })).toBeInTheDocument()
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa' })
  })

  it('restores Back and Forward routes through backend-owned project transitions', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })))
      .mockImplementationOnce(() => response({ projects: [{ name: 'Example Project', slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa' }] }))
      .mockImplementationOnce(() => response(architecture()))
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)
    await screen.findByText('Example Project', { selector: '.workspace-context strong' })

    window.history.pushState({}, '', '/')
    window.dispatchEvent(new PopStateEvent('popstate'))
    expect(await screen.findByRole('heading', { name: 'Projects' })).toBeInTheDocument()
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa' })

    window.history.pushState({}, '', '/projects/example-project')
    window.dispatchEvent(new PopStateEvent('popstate'))
    expect(await screen.findByText('Example Project', { selector: '.workspace-context strong' })).toBeInTheDocument()
    expect(requestBody(fetchMock, 3)).toEqual({ project_slug: 'example-project' })
  })

  it('protects dirty browser values when Back would leave the project', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture())))
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: 'Edit component' }))
    await user.type(screen.getByLabelText('Description'), ' unsent')
    window.history.pushState({}, '', '/')
    window.dispatchEvent(new PopStateEvent('popstate'))
    expect(await screen.findByRole('dialog', { name: 'Leave without keeping?' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('uses server-owned current-home wording and offers only approved move destinations', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture())))
    const user = userEvent.setup()
    render(<App />)
    const navigation = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    expect(screen.queryByRole('button', { name: 'Change where Worker lives' })).not.toBeInTheDocument()
    await user.click(within(navigation).getByRole('button', { name: 'Detail' }))
    await user.click(within(navigation).getByRole('button', { name: 'External' }))
    await user.click(screen.getByRole('button', { name: 'Change where External lives' }))
    expect(screen.getByRole('heading', { name: 'Change where External lives' })).toBeInTheDocument()
    expect(screen.getByText('Currently lives in Detail.')).toBeInTheDocument()
    const picker = screen.getByLabelText('Diagram')
    expect(within(picker).queryByRole('option', { name: 'Detail' })).not.toBeInTheDocument()
    expect(within(picker).getByRole('option', { name: 'System' })).toBeInTheDocument()
  })

  it('disambiguates a move task only when component titles collide', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const duplicateID = '55555555-5555-4555-8555-555555555555'
    const result = architecture()
    vi.stubGlobal('fetch', vi.fn(() => response({
      ...result,
      component_count: 3,
      component_titles: ['Worker', 'External', 'External'],
      components: [
        ...result.components,
        { id: duplicateID, title: 'External', description: 'Another one.\n', filename: 'external-copy.md', relationships: [] },
      ],
    })))
    const user = userEvent.setup()
    render(<App />)
    const navigation = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(navigation).getByRole('button', { name: 'Detail' }))
    await user.click(within(navigation).getByRole('button', { name: 'External' }))
    await user.click(screen.getByRole('button', { name: 'Change where External — external.md lives' }))
    expect(screen.getByRole('heading', { name: 'Change where External — external.md lives' })).toBeInTheDocument()
    expect(screen.getByText('Currently lives in Detail.')).toBeInTheDocument()
  })

  it('keeps the move editor intact when a server-approved destination becomes unavailable', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response({ code: 'home_move_unavailable' }, 409))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    const navigation = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(navigation).getByRole('button', { name: 'Detail' }))
    await user.click(within(navigation).getByRole('button', { name: 'External' }))
    await user.click(screen.getByRole('button', { name: 'Change where External lives' }))
    await user.selectOptions(screen.getByLabelText('Diagram'), root)
    await user.click(screen.getByRole('button', { name: 'Keep change' }))

    expect(requestBody(fetchMock, 1)).toMatchObject({ component_id: external, diagram_id: root })
    expect(await screen.findByRole('alert')).toHaveTextContent('That destination is no longer available. Choose another diagram.')
    expect(screen.getByRole('heading', { name: 'Change where External lives' })).toBeInTheDocument()
  })
})

describe('candidate review regressions', () => {
  it('names each changed Diagram and distinguishes home from reusable appearances', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const value = reviewedArchitecture()
    const review = (value as unknown as { changes: { review: {
      with_changes: { diagrams: Array<Record<string, unknown>> }
      comparison: { appearances: Array<Record<string, unknown>> }
    } } }).changes.review
    review.with_changes.diagrams = review.with_changes.diagrams.map((diagram) => diagram.id === root
      ? { ...diagram, appearances: [{ component_id: worker, role: 'home', detail_diagram_id: detail, detail_diagram_title: 'Detail' }, { component_id: external, role: 'home' }] }
      : { ...diagram, appearances: [{ component_id: worker, role: 'reference' }] })
    review.comparison.appearances = [
      { diagram_id: root, component_id: external, role: 'reference', status: 'removed', path: 'diagrams/root.yaml' },
      { diagram_id: root, component_id: external, role: 'home', status: 'added', path: 'diagrams/root.yaml' },
      { diagram_id: detail, component_id: external, role: 'home', status: 'removed', path: 'diagrams/detail.yaml' },
      { diagram_id: detail, component_id: worker, role: 'reference', status: 'added', path: 'diagrams/detail.yaml' },
      { diagram_id: root, component_id: worker, role: 'home', status: 'detail_changed', path: 'diagrams/root.yaml' },
    ]
    vi.stubGlobal('fetch', vi.fn(() => response(value)))
    const user = userEvent.setup()
    render(<App />)

    const diagramChanges = await screen.findByRole('region', { name: 'Diagram changes' })
    expect(within(diagramChanges).getByRole('button', { name: 'External no longer shown in System' })).toBeInTheDocument()
    const homeAdded = within(diagramChanges).getByRole('button', { name: 'External now lives in System' })
    expect(homeAdded).toBeInTheDocument()
    expect(within(diagramChanges).getByRole('button', { name: 'External no longer lives in Detail' })).toBeInTheDocument()
    expect(within(diagramChanges).getByRole('button', { name: 'Worker updated shown in Detail' })).toBeInTheDocument()
    expect(within(diagramChanges).getByRole('button', { name: 'Worker updated detail diagram link changed in System' })).toBeInTheDocument()

    await user.click(homeAdded)
    expect(within(screen.getByRole('region', { name: 'Review context' })).getByRole('heading', { name: 'System' })).toBeInTheDocument()
  })

  it('keeps map, index, documentation, and topology on one selected snapshot and focuses the exact diff', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)
    expect(await screen.findByRole('button', { name: 'With changes' })).toHaveAttribute('aria-pressed', 'true')
    const navigation = screen.getByRole('navigation', { name: 'Diagrams and components' })
    expect(within(navigation).getByRole('button', { name: /Worker updated/ })).toBeInTheDocument()
    expect(screen.getByText('Candidate documentation.')).toBeInTheDocument()
    const withElements = graphHarness.calls.at(-1)?.elements ?? []
    expect(withElements.find((element) => element.data.id === 'edge-with')?.data.reviewStatus).toBe('added')

    act(() => graphHarness.nodeSelect?.({ target: { id: () => worker } }))
    await waitFor(() => expect(document.activeElement).toHaveAttribute('data-diff-path', 'components/worker.md'))
    expect(screen.getAllByText('Content changed').length).toBeGreaterThan(0)

    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    expect(within(navigation).getByRole('button', { name: /^Worker/ })).toBeInTheDocument()
    expect(screen.getByText('Does work.')).toBeInTheDocument()
    const beforeElements = graphHarness.calls.at(-1)?.elements ?? []
    expect(beforeElements.find((element) => element.data.id === 'edge')?.data.reviewStatus).toBe('removed')
    expect(beforeElements.some((element) => element.data.id === 'edge-with')).toBe(false)
  })

  it('keeps the canonical diff and acceptance path usable when visual map rendering fails', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    graphHarness.fail = true
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    render(<App />)
    expect(await screen.findByText('The architecture map could not be shown.')).toBeInTheDocument()
    expect(screen.getByTestId('raw-diff')).toHaveTextContent('components/worker.md')
    expect(screen.getByRole('button', { name: 'Update architecture' })).toBeEnabled()
  })

  it('removes stale review controls while preserving read-only changes', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const reviewed = reviewedArchitecture()
    const reviewedChanges = (reviewed as unknown as { changes: Record<string, unknown> }).changes
    const stale = architecture({
      stale: true, action_error: 'refresh_invalid',
      changes: { ...reviewedChanges, stale: true },
    })
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(reviewed))
      .mockImplementationOnce(() => response(stale, 409))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: 'Refresh' }))
    expect(await screen.findByText('These changes started from an older architecture and are read-only.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'With changes' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
  })
})
