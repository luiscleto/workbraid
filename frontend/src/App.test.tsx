import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { App } from './App'

const graphHarness = vi.hoisted(() => ({
  calls: [] as Array<{ elements?: Array<{ data: Record<string, unknown> }> }>,
  annotationMarkers: [] as Array<{ data: Record<string, unknown> }>,
  nodeSelect: undefined as undefined | ((event: { target: { id: () => string; data?: () => unknown } }) => void),
  edgeSelect: undefined as undefined | ((event: { target: { data: () => unknown } }) => void),
  fail: false,
}))

vi.mock('cytoscape', () => ({
  default: (options: { elements?: Array<{ data: Record<string, unknown> }> }) => {
    if (graphHarness.fail) throw new Error('canvas unavailable')
    graphHarness.calls.push(options)
    return {
      batch: (apply: () => void) => apply(),
      add: (elements: Array<{ data: Record<string, unknown> }>) => {
        graphHarness.annotationMarkers = elements
        return { ungrabify: () => ({ unselectify: () => undefined }) }
      },
      on: (_event: string, selector: string | (() => void), callback?: unknown) => {
        if (selector === 'node') graphHarness.nodeSelect = callback as typeof graphHarness.nodeSelect
        if (selector === 'edge') graphHarness.edgeSelect = callback as typeof graphHarness.edgeSelect
      },
      off: () => undefined,
      nodes: () => Object.assign([], { remove: () => undefined }),
      resize: () => undefined,
      destroy: () => undefined,
      fit: () => undefined,
      $: () => ({ unselect: () => undefined }),
      getElementById: () => ({ select: () => undefined, empty: () => false, renderedPosition: () => ({ x: 200, y: 200 }), renderedMidpoint: () => ({ x: 200, y: 200 }) }),
    }
  },
}))

const root = '11111111-1111-4111-8111-111111111111'
const worker = '22222222-2222-4222-8222-222222222222'
const external = '33333333-3333-4333-8333-333333333333'
const detail = '44444444-4444-4444-8444-444444444444'
const candidateDetail = '55555555-5555-4555-8555-555555555555'
const candidateComponent = '66666666-6666-4666-8666-666666666666'

function response(value: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } }))
}

function architecture(overrides: Record<string, unknown> = {}) {
  const result: Record<string, any> = {
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
    change_sets: [],
    ...overrides,
  }
  if (result.changes) {
    result.changes = {
      id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', name: 'Steady lantern', lifecycle: 'active', proposal_markdown: '',
      base_revision: result.revision, generation: 1, components: [], valid: true,
      ...(result.changes as Record<string, unknown>),
    }
    if (!Object.hasOwn(overrides, 'change_sets')) result.change_sets = [result.changes]
  }
  return result
}

function requestBody(mock: ReturnType<typeof vi.fn>, index: number) {
  return JSON.parse(String(mock.mock.calls[index][1]?.body))
}

async function selectShowing(user: ReturnType<typeof userEvent.setup>, optionName: string) {
  const trigger = await screen.findByRole('button', { name: /^Showing / })
  await user.click(trigger)
  expect(trigger).toHaveAttribute('aria-expanded', 'true')
  await user.click(screen.getByRole('option', { name: optionName }))
  expect(trigger).toHaveAttribute('aria-expanded', 'false')
}

async function selectProposal(user: ReturnType<typeof userEvent.setup>) {
  await selectShowing(user, 'Steady lantern')
  await user.click(await screen.findByRole('button', { name: 'Return to review' }))
}

function changeSet(id: string, name: string, overrides: Record<string, unknown> = {}) {
  return {
    id, name, lifecycle: 'active', proposal_markdown: '', base_revision: 'a'.repeat(40), generation: 0,
    components: [], valid: true, candidate: architecture(),
    ...overrides,
  }
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
        diff, reviewed_state: 'c'.repeat(40), base_revision: 'a'.repeat(40), candidate_tree: 'b'.repeat(40), generation: 4,
        before: { revision: 'a'.repeat(40), format_version: 2, component_count: 2, component_titles: ['Worker', 'External'], components: beforeComponents, root_diagram_id: root, diagrams: beforeDiagrams },
        with_changes: { revision: 'b'.repeat(40), format_version: 2, component_count: 2, component_titles: ['Worker updated', 'External'], components: withComponents, root_diagram_id: root, diagrams: withDiagrams },
        comparison: {
          components: [{ component_id: worker, status: 'content_changed', path: 'components/worker.md' }],
          relationships: [
            { key: 'removed-edge', before_key: 'edge', source_id: worker, target_id: external, label: 'calls', status: 'removed', path: 'components/worker.md', occurrence: 1, diagram_projections: [{ side: 'before', diagram_id: root, key: 'edge', source_node_key: worker, target_node_key: `boundary:${external}` }] },
            { key: 'edge-with', source_id: worker, target_id: external, label: 'invokes', status: 'added', path: 'components/worker.md', occurrence: 1, diagram_projections: [{ side: 'with', diagram_id: root, key: 'edge-with', source_node_key: worker, target_node_key: external }] },
          ],
          diagrams: [], appearances: [{ diagram_id: root, component_id: external, role: 'reference', status: 'added', side: 'with_changes', path: 'diagrams/root.yaml' }],
        },
      },
    },
    ...overrides,
  })
}

function submittedReviewArchitecture(overrides: Record<string, unknown> = {}) {
  const value = reviewedArchitecture()
  const current = value as unknown as { changes: Record<string, any>; review_submissions?: unknown[]; submitted_review?: unknown }
  const reviewID = '77777777-7777-4777-8777-777777777777'
  const submission = {
    id: reviewID,
    change_set_id: current.changes.id,
    reviewed_state: current.changes.review.reviewed_state,
    binding: { base_revision: current.changes.review.base_revision, candidate_tree: current.changes.review.candidate_tree, generation: current.changes.review.generation },
    verdict: 'request_changes',
    author: 'Review agent',
    submitted_at: '2026-09-04T14:30:00Z',
    comment_count: 1,
    lifecycle: 'active',
    current_generation: true,
    body: 'Please clarify the worker responsibility.',
    comments: [{
      id: '88888888-8888-4888-8888-888888888888',
      body: 'This wording is ambiguous.',
      anchor: { kind: 'component_markdown', side: 'with_changes', component_id: worker, start_line: 2, end_line: 2 },
    }],
    proposal_markdown: current.changes.proposal_markdown,
    review: current.changes.review,
    ...overrides,
  }
  current.review_submissions = [submission]
  current.submitted_review = submission
  return value
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
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', expected_revision: 'a'.repeat(40) })
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
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', expected_revision: 'a'.repeat(40), pending_generation_observed: true, expected_pending_generation: null, diagram_id: root, component_id: external })
    expect(await screen.findByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb')
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
    expect(requestBody(fetchMock, 1)).toEqual({ project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', expected_revision: 'a'.repeat(40), pending_generation_observed: true, expected_pending_generation: null, diagram_id: root, component_id: external })
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

  it('leaves durable proposals intact when switching projects', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const pending = architecture({ changes: { components: [{ id: worker, title: 'Worker', description: 'Changed.\n', new: false, relationships: [] }], valid: true, candidate: architecture() } })
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(pending))
      .mockImplementationOnce(() => Promise.resolve(new Response(null, { status: 204 })))
      .mockImplementationOnce(() => response({ projects: [] }))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)
    await user.click(await screen.findByRole('button', { name: 'Open another project' }))
    expect(await screen.findByRole('heading', { name: 'Projects' })).toBeInTheDocument()
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
  it('submits immutable anchored feedback and opens its exact review route', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const current = reviewedArchitecture()
    const currentChanges = (current as unknown as { changes: Record<string, any> }).changes
    currentChanges.proposal_markdown = '# Direction\n\nReview this exact proposal.\n'
    currentChanges.review.with_changes.components[0].markdown_source = '# Worker updated\nCandidate documentation.\n'
    const submitted = submittedReviewArchitecture()
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(current))
      .mockImplementationOnce(() => response(architecture({ action_review_id: reviewID }), 201))
      .mockImplementationOnce(() => response(submitted))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Human reviewer')
    await user.selectOptions(screen.getByRole('combobox', { name: 'Conclusion' }), 'request_changes')
    await user.type(screen.getByRole('textbox', { name: /Review summary/ }), 'Please revise this.')
    await user.click(within(screen.getByRole('region', { name: 'Review context' })).getByRole('button', { name: 'Comment on lines' }))
    await user.click(within(screen.getByRole('listbox', { name: 'Worker updated lines source lines' })).getByRole('option', { name: /2Candidate documentation/ }))
    const commentEditor = screen.getByRole('region', { name: /^Comment on/ })
    await user.type(within(commentEditor).getByRole('textbox', { name: 'Comment' }), 'Clarify this line.')
    await user.click(within(commentEditor).getByRole('button', { name: 'Add comment' }))
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    expect(requestBody(fetchMock, 1)).toMatchObject({
      change_set_id: changeSetID,
      reviewed_state: 'c'.repeat(40),
      base_revision: 'a'.repeat(40),
      candidate_tree: 'b'.repeat(40),
      generation: 4,
      verdict: 'request_changes',
      author: 'Human reviewer',
      body: 'Please revise this.',
      comments: [{ body: 'Clarify this line.', anchor: { kind: 'component_markdown', side: 'with_changes', component_id: worker, start_line: 2, end_line: 2 } }],
    })
    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
    expect(screen.getByText('Submitted review')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Showing / })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
  })

  it('restores discarded-proposal feedback from its exact direct route without a selector record', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
    const catalogState = architecture({ change_sets: [] })
    const submitted = submittedReviewArchitecture({ lifecycle: 'no_longer_active', current_generation: false })
    ;(submitted as unknown as { change_sets: unknown[] }).change_sets = []
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(catalogState))
      .mockImplementationOnce(() => response(submitted))
    vi.stubGlobal('fetch', fetchMock)
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    expect(screen.getByText('This proposal is no longer active.')).toBeInTheDocument()
    expect(screen.getByText('Please clarify the worker responsibility.')).toBeInTheDocument()
    expect(screen.getByText('This wording is ambiguous.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Showing / })).not.toBeInTheDocument()
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
  })

  it('keeps an earlier submitted review on its exact historical projection when the active proposal advanced', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
    const submitted = submittedReviewArchitecture({ current_generation: false }) as Record<string, any>
    submitted.change_sets = [{
      ...submitted.changes,
      generation: 5,
      proposal_markdown: '# New proposal version\n',
      review: undefined,
      candidate: {
        ...submitted.changes.candidate,
        components: submitted.changes.candidate.components.map((component: Record<string, any>) => component.id === worker ? { ...component, title: 'Later Worker' } : component),
      },
    }]
    vi.stubGlobal('fetch', vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response(submitted)))
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    expect(screen.getByText('Feedback on an earlier proposal version.')).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Proposal' })).not.toHaveTextContent('New proposal version')
    expect(screen.getByRole('navigation', { name: 'Diagrams and components' })).toHaveTextContent('Worker updated')
    expect(screen.getByRole('navigation', { name: 'Diagrams and components' })).not.toHaveTextContent('Later Worker')
  })

  it('offers clear routes out of an exact submitted review while keeping its history compact', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
    vi.stubGlobal('fetch', vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response(submittedReviewArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    const navigation = screen.getByRole('navigation', { name: 'Review navigation' })
    expect(within(navigation).getByRole('button', { name: 'Current review' })).toBeInTheDocument()
    expect(within(navigation).getByRole('button', { name: 'Proposal workspace' })).toBeInTheDocument()
    expect(within(navigation).getByRole('button', { name: 'Accepted workspace' })).toBeInTheDocument()
    const history = screen.getByRole('region', { name: 'Submitted reviews' })
    expect(within(history).getByText(/Review history/).closest('details')).not.toHaveAttribute('open')

    await user.click(within(navigation).getByRole('button', { name: 'Current review' }))
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}/review`)
    expect(screen.getByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
  })

  it('marks exact submitted comments in navigation and opens independent notes on the map', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
    const submitted = submittedReviewArchitecture() as Record<string, any>
    const review = submitted.submitted_review
    review.comments = [
      { id: '10000000-0000-4000-8000-000000000001', body: 'Diagram note.', anchor: { kind: 'diagram', side: 'with_changes', diagram_id: root } },
      { id: '10000000-0000-4000-8000-000000000002', body: 'Component note.', anchor: { kind: 'component', side: 'with_changes', component_id: worker } },
      { id: '10000000-0000-4000-8000-000000000003', body: 'Placement note.', anchor: { kind: 'composition', side: 'with_changes', diagram_id: root, component_id: worker, aspect: 'home' } },
      { id: '10000000-0000-4000-8000-000000000004', body: 'Relationship note.', anchor: { kind: 'relationship', side: 'with_changes', source_component_id: worker, target_component_id: external, label: 'invokes', occurrence: 1 } },
    ]
    review.comment_count = review.comments.length
    submitted.review_submissions = [review]
    vi.stubGlobal('fetch', vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response(submitted)))
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    const navigator = screen.getByRole('navigation', { name: 'Diagrams and components' })
    const diagramMarker = within(navigator).getByRole('button', { name: '1 review comment on System' })
    expect(within(navigator).getByRole('button', { name: '2 review comments on Worker updated' })).toBeInTheDocument()
    await user.click(diagramMarker)
    expect(screen.queryByRole('tab', { name: 'Comments' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Comments on System')).toHaveTextContent('Diagram note.')
    await user.click(within(navigator).getByRole('button', { name: 'Detail' }))
    expect(screen.queryByLabelText('Comments on System')).not.toBeInTheDocument()
    await user.click(diagramMarker)
    expect(screen.getByLabelText('Comments on System')).toHaveTextContent('Diagram note.')
    await user.click(screen.getByRole('button', { name: 'Close comments on System' }))

    await waitFor(() => expect(graphHarness.calls.length).toBeGreaterThan(0))
    const workerMarker = graphHarness.annotationMarkers.find(element => element.data.annotationNodeID === worker)!
    expect(workerMarker.data.displayLabel).toBe('✎ 2')
    const graphCount = graphHarness.calls.length
    act(() => graphHarness.nodeSelect?.({ target: { id: () => String(workerMarker.data.id), data: () => workerMarker.data } }))
    expect(within(screen.getByLabelText('Open review comments')).getAllByRole('article')).toHaveLength(1)
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('Component note.')
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('Placement note.')

    const relationshipMarker = graphHarness.annotationMarkers.find(element => element.data.annotationRelationshipKey === 'edge-with')!
    expect(relationshipMarker.data.displayLabel).toBe('✎ 1')
    act(() => graphHarness.nodeSelect?.({ target: { id: () => String(relationshipMarker.data.id), data: () => relationshipMarker.data } }))
    expect(within(screen.getByLabelText('Open review comments')).getAllByRole('article')).toHaveLength(2)
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('Relationship note.')
    expect(graphHarness.calls).toHaveLength(graphCount)

    await user.click(screen.getByRole('button', { name: 'Close comments on Worker updated' }))
    expect(within(screen.getByLabelText('Open review comments')).getAllByRole('article')).toHaveLength(1)

    await user.click(within(navigator).getByRole('button', { name: 'Detail' }))
    expect(within(navigator).getByRole('heading', { name: 'Comments elsewhere' })).toBeInTheDocument()
    const elsewhereMarker = within(navigator).getByRole('button', { name: '2 review comments on Worker updated' })
    await user.click(elsewhereMarker)
    expect(within(navigator).getByRole('button', { name: 'System, Changed' })).toHaveAttribute('aria-current', 'page')
    expect(within(navigator).getByRole('button', { name: 'Worker updated, Content changed' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('Component note.')

    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(screen.queryByLabelText('Open review comments')).not.toBeInTheDocument()
  })

  it('starts a Diagram comment without existing feedback and submits its exact anchor', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    const fetchMock = vi.fn(() => response(reviewedArchitecture()))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    const navigator = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(navigator).getByRole('button', { name: 'Comment on System' }))
    const editor = screen.getByRole('region', { name: 'Comment on System' })
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'Clarify this diagram.')
    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Reviewer')
    await user.selectOptions(screen.getByRole('combobox', { name: 'Conclusion' }), 'request_changes')
    expect(screen.getByRole('button', { name: 'Submit review' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Submit review' })).toHaveAccessibleDescription('Add or cancel the open comment before submitting.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))
    expect(screen.getByLabelText('Comments on System')).toHaveTextContent('Clarify this diagram.')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole('button', { name: 'Submit review' }))
    expect(requestBody(fetchMock, 1)).toMatchObject({ comments: [{
      body: 'Clarify this diagram.', anchor: { kind: 'diagram', side: 'with_changes', diagram_id: root },
    }] })
  })

  it('keeps pinned comments live and edits by local identity when another comment is removed', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    const fetchMock = vi.fn(() => response(reviewedArchitecture()))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    const context = await screen.findByRole('region', { name: 'Review context' })
    await user.click(within(context).getByRole('button', { name: 'Comment on component' }))
    let editor = screen.getByRole('region', { name: 'Comment on Worker updated' })
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'First note.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))
    const marker = graphHarness.annotationMarkers.find(element => element.data.annotationNodeID === worker)!
    act(() => graphHarness.nodeSelect?.({ target: { id: () => String(marker.data.id), data: () => marker.data } }))
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('First note.')

    await user.click(within(context).getByRole('button', { name: 'Comment on component' }))
    editor = screen.getByRole('region', { name: 'Comment on Worker updated' })
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'Second note.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('Second note.')
    const summary = screen.getByRole('region', { name: 'Comments to submit' })
    const second = within(summary).getByText('Second note.').closest('li')!
    await user.click(within(second).getByRole('button', { name: 'Edit' }))
    await user.clear(within(second).getByRole('textbox', { name: 'Comment' }))
    await user.type(within(second).getByRole('textbox', { name: 'Comment' }), 'Updated second note.')
    const note = within(screen.getByLabelText('Open review comments')).getByText('First note.').closest('section')!
    await user.click(within(note).getByRole('button', { name: 'Remove' }))
    await user.click(within(summary).getByRole('button', { name: 'Save comment' }))
    expect(screen.getByLabelText('Open review comments')).toHaveTextContent('Updated second note.')
    expect(screen.getByLabelText('Open review comments')).not.toHaveTextContent('First note.')

    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Reviewer')
    const openNote = screen.getByLabelText('Open review comments')
    await user.click(within(openNote).getByRole('button', { name: 'Edit' }))
    editor = screen.getByRole('region', { name: 'Comment on Worker updated' })
    await user.clear(within(editor).getByRole('textbox', { name: 'Comment' }))
    expect(screen.getByRole('button', { name: 'Submit review' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Keep editing' }))
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'Final second note.')
    await user.click(within(editor).getByRole('button', { name: 'Save comment' }))
    await user.click(screen.getByRole('button', { name: 'Submit review' }))
    expect(requestBody(fetchMock, 1)).toMatchObject({ comments: [{
      body: 'Final second note.', anchor: { kind: 'component', side: 'with_changes', component_id: worker },
    }] })
  })

  it('keeps Before and With Diagram notes separate when a summary edit happens on the other side', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)
    const navigator = await screen.findByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(navigator).getByRole('button', { name: 'Comment on System' }))
    let editor = screen.getByRole('region', { name: 'Comment on System' })
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'With note.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    await user.click(within(navigator).getByRole('button', { name: 'Comment on System' }))
    editor = screen.getByRole('region', { name: 'Comment on System' })
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'Before note.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))
    await user.click(screen.getByRole('button', { name: 'With changes' }))
    const summary = screen.getByRole('region', { name: 'Comments to submit' })
    const beforeNote = within(summary).getByText('Before note.').closest('li')!
    await user.click(within(beforeNote).getByRole('button', { name: 'Edit' }))
    await user.clear(within(beforeNote).getByRole('textbox', { name: 'Comment' }))
    await user.type(within(beforeNote).getByRole('textbox', { name: 'Comment' }), 'Edited Before note.')
    await user.click(within(beforeNote).getByRole('button', { name: 'Save comment' }))
    expect(screen.getByLabelText('Comments on System')).toHaveTextContent('With note.')
    expect(screen.getByLabelText('Comments on System')).not.toHaveTextContent('Edited Before note.')
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    expect(screen.getByLabelText('Comments on System')).toHaveTextContent('Edited Before note.')
    expect(screen.getByLabelText('Comments on System')).not.toHaveTextContent('With note.')
  })

  it('resets the same comment target after explicit discard and guards further edits again', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    const context = await screen.findByRole('region', { name: 'Review context' })
    const add = within(context).getByRole('button', { name: 'Comment on component' })
    await user.click(add)
    await user.type(screen.getByRole('textbox', { name: 'Comment' }), 'Do not retain this.')
    await user.click(add)
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.getByRole('textbox', { name: 'Comment' })).toHaveValue('')
    await user.type(screen.getByRole('textbox', { name: 'Comment' }), 'Keep this instead.')
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('textbox', { name: 'Comment' })).toHaveValue('Keep this instead.')
  })

  it('guards an existing line comment when only its exact range changes', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    const current = reviewedArchitecture() as Record<string, any>
    current.changes.review.with_changes.components[0].markdown_source = '# Worker\n\nDoes work.\n'
    const fetchMock = vi.fn(() => response(current))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    const context = await screen.findByRole('region', { name: 'Review context' })
    await user.click(within(context).getByRole('button', { name: 'Comment on lines' }))
    let editor = screen.getByRole('region', { name: 'Comment on Worker updated lines' })
    await user.click(within(editor).getAllByRole('option')[0])
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'Explain this.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))
    const marker = graphHarness.annotationMarkers.find(element => element.data.annotationNodeID === worker)!
    act(() => graphHarness.nodeSelect?.({ target: { id: () => String(marker.data.id), data: () => marker.data } }))
    await user.click(within(screen.getByLabelText('Open review comments')).getByRole('button', { name: 'Edit' }))
    editor = screen.getByRole('region', { name: 'Comment on Worker updated' })
    await user.click(within(editor).getAllByRole('option')[2])
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Keep editing' }))
    expect(within(editor).getByRole('textbox', { name: 'Comment' })).toHaveValue('Explain this.')
    await user.click(within(editor).getByRole('button', { name: 'Save comment' }))
    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Reviewer')
    await user.click(screen.getByRole('button', { name: 'Submit review' }))
    expect(requestBody(fetchMock, 1)).toMatchObject({ comments: [{
      body: 'Explain this.', anchor: { kind: 'component_markdown', side: 'with_changes', component_id: worker, start_line: 1, end_line: 3 },
    }] })
  })

  it('selects, edits and removes an exact proposal Markdown line-range comment', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const current = reviewedArchitecture() as Record<string, any>
    current.changes.proposal_markdown = '# Direction\n\nReview this exact proposal.\n'
    vi.stubGlobal('fetch', vi.fn(() => response(current)))
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    await user.click(within(screen.getByRole('group', { name: 'Proposal' })).getByRole('button', { name: 'Comment on lines' }))
    const source = screen.getByRole('listbox', { name: 'Proposal lines source lines' })
    const lines = within(source).getAllByRole('option')
    await user.click(lines[0])
    await user.keyboard('{Shift>}')
    await user.click(lines[2])
    await user.keyboard('{/Shift}')
    expect(lines[0]).toHaveAttribute('aria-selected', 'true')
    expect(lines[1]).toHaveAttribute('aria-selected', 'true')
    expect(lines[2]).toHaveAttribute('aria-selected', 'true')
    const editor = screen.getByRole('region', { name: 'Comment on Proposal lines' })
    await user.type(within(editor).getByRole('textbox', { name: 'Comment' }), 'Clarify the direction.')
    await user.click(within(editor).getByRole('button', { name: 'Add comment' }))

    const draft = screen.getByText('Clarify the direction.').closest('li')
    expect(draft).not.toBeNull()
    expect(draft).toHaveTextContent('Proposal lines 1–3')
    await user.click(within(draft!).getByRole('button', { name: 'Edit' }))
    const editComment = within(draft!).getByRole('textbox', { name: 'Comment' })
    await user.clear(editComment)
    await user.type(editComment, 'Clarify all three lines.')
    await user.click(within(draft!).getByRole('button', { name: 'Save comment' }))
    const updatedDraft = screen.getByText('Clarify all three lines.').closest('li')
    expect(updatedDraft).not.toBeNull()
    await user.click(within(updatedDraft!).getByRole('button', { name: 'Remove' }))
    expect(screen.queryByText('Clarify all three lines.')).not.toBeInTheDocument()
  })

  it('keeps earlier submitted feedback discoverable after proposal iteration invalidates Review changes', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}`)
    const current = submittedReviewArchitecture({ current_generation: false }) as Record<string, any>
    delete current.submitted_review
    delete current.changes.review
    current.changes.generation = 5
    current.change_sets = [current.changes]
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(current))
      .mockImplementationOnce(() => response(submittedReviewArchitecture({ current_generation: false })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Submitted reviews' })).toHaveTextContent('Changes requested')
    expect(screen.getByRole('region', { name: 'Submitted reviews' })).toHaveTextContent('earlier version')
    const submittedReviews = screen.getByRole('region', { name: 'Submitted reviews' })
    await user.click(within(submittedReviews).getByText(/Review history/))
    await user.click(within(submittedReviews).getByRole('button', { name: /Changes requested/ }))
    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
  })

  it('keeps a removed composition fact anchored to its exact Before side while toggling snapshots', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const current = reviewedArchitecture() as Record<string, any>
    const beforeRoot = current.changes.review.before.diagrams.find((diagram: Record<string, any>) => diagram.id === root)
    const withRoot = current.changes.review.with_changes.diagrams.find((diagram: Record<string, any>) => diagram.id === root)
    beforeRoot.appearances.push({ component_id: external, role: 'reference' })
    beforeRoot.boundaries = []
    withRoot.appearances = withRoot.appearances.filter((appearance: Record<string, any>) => appearance.component_id !== external)
    current.changes.review.comparison.appearances = [{
      diagram_id: root, component_id: external, role: 'reference', status: 'removed', side: 'before', path: 'diagrams/root.yaml',
    }]
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(current))
      .mockImplementationOnce(() => response({ code: 'review_changed' }, 409))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'Composition: External no longer shown in System' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Composition reviewer')
    await user.selectOptions(screen.getByRole('combobox', { name: 'Conclusion' }), 'request_changes')
    await user.click(within(screen.getByRole('region', { name: 'Review context' })).getByRole('button', { name: 'Comment on this change' }))
    const commentEditor = screen.getByRole('region', { name: /^Comment on/ })
    expect(commentEditor).toBeInTheDocument()
    await user.type(within(commentEditor).getByRole('textbox', { name: 'Comment' }), 'Keep this placement.')
    await user.click(screen.getByRole('button', { name: 'With changes' }))
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Keep editing' }))
    await user.click(within(commentEditor).getByRole('button', { name: 'Add comment' }))
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    expect(requestBody(fetchMock, 1)).toMatchObject({ comments: [{
      body: 'Keep this placement.',
      anchor: { kind: 'composition', side: 'before', diagram_id: root, component_id: external, aspect: 'reference' },
    }] })
  })

  it('carries the exact detail child identity into a detail-link review comment', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const current = reviewedArchitecture() as Record<string, any>
    const withRoot = current.changes.review.with_changes.diagrams.find((diagram: Record<string, any>) => diagram.id === root)
    withRoot.appearances = withRoot.appearances.map((appearance: Record<string, any>) => appearance.component_id === worker
      ? { component_id: worker, role: 'home' }
      : appearance)
    current.changes.review.comparison.appearances = [{
      diagram_id: root, component_id: worker, role: 'home', status: 'detail_changed', side: 'before',
      detail_diagram_id: detail, path: 'diagrams/root.yaml',
    }]
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(current))
      .mockImplementationOnce(() => response({ code: 'review_changed' }, 409))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.click(within(await screen.findByRole('region', { name: 'Diagram changes' })).getByRole('button', { name: 'Worker detail diagram link changed in System' }))
    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Hierarchy reviewer')
    await user.selectOptions(screen.getByRole('combobox', { name: 'Conclusion' }), 'request_changes')
    await user.click(within(screen.getByRole('region', { name: 'Review context' })).getByRole('button', { name: 'Comment on this change' }))
    const commentEditor = screen.getByRole('region', { name: /^Comment on/ })
    await user.type(within(commentEditor).getByRole('textbox', { name: 'Comment' }), 'Keep the detail link.')
    await user.click(within(commentEditor).getByRole('button', { name: 'Add comment' }))
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    expect(requestBody(fetchMock, 1)).toMatchObject({ comments: [{ anchor: {
      kind: 'composition', side: 'before', diagram_id: root, component_id: worker,
      aspect: 'detail', detail_diagram_id: detail,
    } }] })
  })

  it('keeps an exact bound Review available for feedback while Accepted is known non-current', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const current = reviewedArchitecture({ stale: true }) as Record<string, any>
    current.changes.stale = true
    vi.stubGlobal('fetch', vi.fn(() => response(current)))
    render(<App />)

    expect(await screen.findByRole('alert')).toHaveTextContent('This earlier view is read-only')
    expect(screen.getByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Reviewer name' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
  })

  it('keeps exact feedback submission available while Accepted observation is indeterminate', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture({ action_error: 'refresh_failed' }))))
    render(<App />)

    expect(await screen.findByRole('alert')).toHaveTextContent("couldn't check for architecture changes")
    expect(screen.getByRole('textbox', { name: 'Reviewer name' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'New changes' })).toBeDisabled()
  })

  it('keeps the exact submitted-review route and snapshot through Refresh', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    const path = `/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`
    window.history.replaceState({}, '', path)
    const submitted = submittedReviewArchitecture()
    const refreshed = submittedReviewArchitecture() as Record<string, unknown>
    delete refreshed.submitted_review
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(architecture()))
      .mockImplementationOnce(() => response(submitted))
      .mockImplementationOnce(() => response(refreshed))
      .mockImplementationOnce(() => response(submitted))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(await screen.findByText('Please clarify the worker responsibility.')).toBeInTheDocument()
    expect(window.location.pathname).toBe(path)
    expect(fetchMock).toHaveBeenCalledTimes(4)
  })

  it('protects unsent review feedback when leaving its exact review', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    const author = screen.getByRole('textbox', { name: 'Reviewer name' })
    await user.type(author, 'Local reviewer')
    await selectShowing(user, 'Accepted')
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(author).toHaveValue('Local reviewer')
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}/review`)

    await selectShowing(user, 'Accepted')
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('protects unsent review feedback before continuing to proposal editing', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewPath = `/projects/example-project/proposals/${changeSetID}/review`
    window.history.replaceState({}, '', reviewPath)
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    const author = await screen.findByRole('textbox', { name: 'Reviewer name' })
    await user.type(author, 'Local reviewer')
    await user.click(screen.getByRole('button', { name: 'Back to proposal' }))
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(author).toHaveValue('Local reviewer')
    expect(window.location.pathname).toBe(reviewPath)

    await user.click(screen.getByRole('button', { name: 'Back to proposal' }))
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Proposal' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}`)
  })

  it('protects a partially typed relationship comment before switching review sides', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'Added relationship: Worker updated — invokes — External' }))
    const context = screen.getByRole('region', { name: 'Review context' })
    await user.click(within(context).getByRole('button', { name: 'Add comment' }))
    const editor = screen.getByRole('region', { name: /Comment on Worker updated/ })
    const comment = within(editor).getByRole('textbox', { name: 'Comment' })
    await user.type(comment, 'Unsent relationship feedback')
    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(comment).toHaveValue('Unsent relationship feedback')
    expect(screen.getByRole('button', { name: 'With changes' })).toHaveAttribute('aria-pressed', 'true')

    await user.click(screen.getByRole('button', { name: 'Before changes' }))
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.getByRole('button', { name: 'Before changes' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByText('Unsent relationship feedback')).not.toBeInTheDocument()
  })

  it('protects a partially typed component comment before changing selection', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    const context = await screen.findByRole('region', { name: 'Review context' })
    await user.click(within(context).getByRole('button', { name: 'Comment on component' }))
    const editor = screen.getByRole('region', { name: /Comment on Worker updated/ })
    const comment = within(editor).getByRole('textbox', { name: 'Comment' })
    await user.type(comment, 'Unsent component feedback')
    await user.click(screen.getByRole('button', { name: 'External, Included here · Lives in Detail' }))
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(comment).toHaveValue('Unsent component feedback')
    expect(screen.getByRole('heading', { name: 'Worker updated' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'External, Included here · Lives in Detail' }))
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.getByRole('heading', { name: 'External' })).toBeInTheDocument()
    expect(screen.queryByText('Unsent component feedback')).not.toBeInTheDocument()
  })

  it('protects unsent feedback before opening an earlier submitted review', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const reviewID = '77777777-7777-4777-8777-777777777777'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const current = submittedReviewArchitecture() as Record<string, unknown>
    delete current.submitted_review
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(current))
      .mockImplementationOnce(() => response(submittedReviewArchitecture()))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.type(await screen.findByRole('textbox', { name: 'Reviewer name' }), 'Local reviewer')
    await user.click(screen.getByRole('button', { name: /Changes requested.*Review agent/ }))
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    expect(fetchMock).toHaveBeenCalledTimes(1)
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('textbox', { name: 'Reviewer name' })).toHaveValue('Local reviewer')

    await user.click(screen.getByRole('button', { name: /Changes requested.*Review agent/ }))
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('heading', { name: 'Changes requested' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}/reviews/${reviewID}`)
  })

  it('restores a UUID-addressed review with safe proposal context and returns to its proposal task', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const value = reviewedArchitecture()
    const changes = (value as unknown as { changes: Record<string, unknown> }).changes
    changes.proposal_markdown = '# Review direction\n\nKeep the boundary.\n\n<script>alert(1)</script>'
    vi.stubGlobal('fetch', vi.fn(() => response(value)))
    const user = userEvent.setup()
    const { container } = render(<App />)

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}/review`)
    const proposal = screen.getByRole('group', { name: 'Proposal' })
    expect(proposal).toHaveAttribute('open')
    expect(within(proposal).getByRole('heading', { name: 'Review direction' })).toBeInTheDocument()
    expect(within(proposal).getByText('Keep the boundary.')).toBeInTheDocument()
    expect(container.querySelector('script')).toBeNull()

    await user.click(screen.getByRole('button', { name: 'Back to proposal' }))
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${changeSetID}`)
    expect(await screen.findByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()
  })

  it('restores review and proposal tasks across same-project history navigation', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const proposalPath = `/projects/example-project/proposals/${changeSetID}`
    const reviewPath = `/projects/example-project/proposals/${changeSetID}/review`
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)
    await selectProposal(user)
    expect(window.location.pathname).toBe(reviewPath)

    window.history.pushState({}, '', proposalPath)
    window.dispatchEvent(new PopStateEvent('popstate'))
    expect(await screen.findByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()

    window.history.pushState({}, '', reviewPath)
    window.dispatchEvent(new PopStateEvent('popstate'))
    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(reviewPath)
  })

  it.each([
    { kind: 'invalidated', value: architecture({ change_sets: [changeSet('bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', 'Steady lantern')] }), notice: 'This proposal has changed and needs to be reviewed again.', expectedHeading: 'Steady lantern', expectedPath: '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb' },
    { kind: 'absent', value: architecture({ change_sets: [], unavailable_change_sets: [{ id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', reason: 'invalid metadata' }] }), notice: 'That review is no longer available.', expectedHeading: 'Worker', expectedPath: '/projects/example-project' },
  ])('normalizes an $kind direct review without manufacturing a replacement', async ({ value, notice, expectedHeading, expectedPath }) => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    vi.stubGlobal('fetch', vi.fn(() => response(value)))
    render(<App />)

    expect((await screen.findByText(notice)).closest('[role="alert"]')).toHaveTextContent(notice)
    expect(screen.getByRole('heading', { name: expectedHeading })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()
    expect(window.location.pathname).toBe(expectedPath)
  })

  it('protects an unsaved proposal when history requests its review route', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const proposalPath = `/projects/example-project/proposals/${changeSetID}`
    const reviewPath = `/projects/example-project/proposals/${changeSetID}/review`
    window.history.replaceState({}, '', '/projects/example-project')
    window.history.pushState({}, '', reviewPath)
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Back to proposal' }))
    expect(window.location.pathname).toBe(proposalPath)
    await user.type(screen.getByRole('textbox', { name: 'Proposal' }), 'Local direction')

    act(() => window.history.back())
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    expect(window.location.pathname).toBe(proposalPath)
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('textbox', { name: 'Proposal' })).toHaveValue('Local direction')
    expect(window.location.pathname).toBe(proposalPath)

    act(() => window.history.back())
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(reviewPath)

    act(() => window.history.back())
    expect(await screen.findByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('opens an applied review read-only without an update action', async () => {
    const changeSetID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    window.history.replaceState({}, '', `/projects/example-project/proposals/${changeSetID}/review`)
    const value = reviewedArchitecture()
    const changes = (value as unknown as { changes: Record<string, unknown> }).changes
    changes.lifecycle = 'applied'
    changes.applied_revision = 'c'.repeat(40)
    vi.stubGlobal('fetch', vi.fn(() => response(value)))
    render(<App />)

    expect(await screen.findByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    expect(screen.getByText('Accepted proposal')).toBeInTheDocument()
    expect(screen.getByText('No proposal document.')).toBeInTheDocument()
    expect(screen.getByText('Review the proposed architecture and its complete file changes.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back to proposal' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Write review' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
  })

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
    await selectProposal(user)

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
    await selectProposal(user)
    expect(await screen.findByRole('button', { name: 'With changes' })).toHaveAttribute('aria-pressed', 'true')
    const navigation = screen.getByRole('navigation', { name: 'Diagrams and components' })
    expect(within(navigation).getByRole('button', { name: 'Worker updated, Content changed' })).toBeInTheDocument()
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
    const user = userEvent.setup()
    render(<App />)
    await selectProposal(user)
    expect(await screen.findByText('The architecture map could not be shown.')).toBeInTheDocument()
    expect(screen.getByTestId('raw-diff')).toHaveTextContent('components/worker.md')
    expect(screen.getByRole('button', { name: 'Update architecture' })).toBeEnabled()
  })

  it('preserves exact feedback controls but blocks Architecture actions when review authority becomes stale', async () => {
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
    await selectProposal(user)
    await user.click(await screen.findByRole('button', { name: 'Refresh' }))
    expect(await screen.findByText('The current architecture could not be loaded. This earlier view is read-only.')).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review')
    expect(screen.getByRole('button', { name: 'With changes' })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Reviewer name' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'New changes' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Delete proposal' })).not.toBeInTheDocument()
  })
})

describe('proposal workspace contexts', () => {
  it.each([
    { lifecycle: 'active', name: 'Open route', extra: {} },
    { lifecycle: 'applied', name: 'Accepted route', extra: { lifecycle: 'applied', read_only: true, applied_revision: 'b'.repeat(40) } },
  ])('restores a direct $lifecycle proposal task by UUID', async ({ name, extra }) => {
    const proposalID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const proposalPath = `/projects/example-project/proposals/${proposalID}`
    window.history.replaceState({}, '', proposalPath)
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [changeSet(proposalID, name, extra)] }))))
    render(<App />)

    expect(await screen.findByRole('heading', { name })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: `Showing ${name}` })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()
    expect(window.location.pathname).toBe(proposalPath)
  })

  it('normalizes a missing direct proposal to Accepted', async () => {
    window.history.replaceState({}, '', '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({
      change_sets: [],
      unavailable_change_sets: [{ id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', reason: 'invalid metadata' }],
    }))))
    render(<App />)

    expect(await screen.findByText('That proposal is no longer available.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('pushes Showing context routes and restores them with Back and Forward', async () => {
    const proposalID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const proposalPath = `/projects/example-project/proposals/${proposalID}`
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [changeSet(proposalID, 'Steady lantern')] }))))
    const user = userEvent.setup()
    render(<App />)

    await selectShowing(user, 'Steady lantern')
    expect(window.location.pathname).toBe(proposalPath)
    act(() => window.history.back())
    expect(await screen.findByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
    act(() => window.history.forward())
    expect(await screen.findByRole('button', { name: 'Showing Steady lantern' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(proposalPath)
  })

  it('preserves a dirty proposal route so Back can be retried deliberately', async () => {
    const proposalID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const proposalPath = `/projects/example-project/proposals/${proposalID}`
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [changeSet(proposalID, 'Steady lantern')] }))))
    const user = userEvent.setup()
    render(<App />)
    await selectShowing(user, 'Steady lantern')
    await user.type(screen.getByRole('textbox', { name: 'Proposal' }), 'Local direction')

    act(() => window.history.back())
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    expect(window.location.pathname).toBe(proposalPath)
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('textbox', { name: 'Proposal' })).toHaveValue('Local direction')
    act(() => window.history.back())
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('creates a generated proposal from the right-pane New changes task', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const generatedID = 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'
    const generated = changeSet(generatedID, 'quiet-harbor')
    const fetchMock = vi.fn()
      .mockImplementationOnce(() => response(architecture({ change_sets: [] })))
      .mockImplementationOnce(() => response(architecture({ change_sets: [generated], action_change_set_id: generatedID }), 201))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'New changes' }))
    const task = screen.getByRole('heading', { name: 'New changes' }).closest('form')
    expect(task).not.toBeNull()
    expect(task!.closest('.working-pane')).not.toBeNull()
    expect(within(task!).getByText('Starts from current Accepted.')).toBeInTheDocument()
    expect(within(task!).getByText('Leave blank to generate a name.')).toBeInTheDocument()
    expect(within(task!).getByRole('button', { name: 'Cancel' })).toBeEnabled()
    await user.click(within(task!).getByRole('button', { name: 'Create' }))
    expect(requestBody(fetchMock, 1)).toEqual({
      project_slug: 'example-project', store_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', accepted_revision: 'a'.repeat(40),
    })
    expect(await screen.findByRole('heading', { name: 'quiet-harbor' })).toBeInTheDocument()
    expect(screen.getByText('These changes have not updated Architecture yet.')).toBeInTheDocument()
    expect(window.location.pathname).toBe(`/projects/example-project/proposals/${generatedID}`)
  })

  it('cancels New changes back to the previous task and resets pane scroll', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const fetchMock = vi.fn(() => response(architecture({ change_sets: [] })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const { container } = render(<App />)

    expect(await screen.findByText('Does work.')).toBeInTheDocument()
    const pane = container.querySelector<HTMLElement>('.working-pane')!
    pane.scrollTop = 140
    await user.click(screen.getByRole('button', { name: 'New changes' }))
    expect(pane.scrollTop).toBe(0)
    await user.type(screen.getByLabelText('Name'), 'Not created')
    pane.scrollTop = 90
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.getByText('Does work.')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'New changes' })).not.toBeInTheDocument()
    expect(pane.scrollTop).toBe(0)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it.each([
    { origin: 'proposal', review: false, returnPath: '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', nextPath: '/projects/example-project' },
    { origin: 'review', review: true, returnPath: '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/review', nextPath: '/projects/example-project/proposals/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb' },
  ])('cancels New changes through history from a $origin without leaving a duplicate entry', async ({ review, returnPath, nextPath }) => {
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(reviewedArchitecture())))
    const user = userEvent.setup()
    render(<App />)

    await selectShowing(user, 'Steady lantern')
    if (review) await user.click(screen.getByRole('button', { name: 'Return to review' }))
    expect(window.location.pathname).toBe(returnPath)
    await user.click(screen.getByRole('button', { name: 'New changes' }))
    await user.type(screen.getByLabelText('Name'), 'Not created')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    await waitFor(() => expect(window.location.pathname).toBe(returnPath))
    if (review) expect(screen.getByRole('heading', { name: 'Review changes' })).toBeInTheDocument()
    else expect(screen.getByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()

    act(() => window.history.back())
    await waitFor(() => expect(window.location.pathname).toBe(nextPath))
    if (review) {
      expect(screen.getByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
      expect(screen.queryByRole('heading', { name: 'Review changes' })).not.toBeInTheDocument()
    } else {
      expect(screen.getByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    }
  })

  it('guards a typed New changes name and drops it only after confirmed navigation', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const proposalID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const fetchMock = vi.fn(() => response(architecture({ change_sets: [changeSet(proposalID, 'Steady lantern')] })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    render(<App />)

    await user.click(await screen.findByRole('button', { name: 'New changes' }))
    await user.type(screen.getByLabelText('Name'), 'Local direction')
    await selectShowing(user, 'Steady lantern')
    let guard = screen.getByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByRole('heading', { name: 'New changes' })).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveValue('Local direction')
    expect(screen.getByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()

    await selectShowing(user, 'Steady lantern')
    guard = screen.getByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(screen.getByRole('button', { name: 'Showing Steady lantern' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'New changes' }))
    expect(screen.getByLabelText('Name')).toHaveValue('')
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('keeps a New changes name when Back to the prior proposal is canceled and permits the retry', async () => {
    const proposalID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const proposalPath = `/projects/example-project/proposals/${proposalID}`
    window.history.replaceState({}, '', '/projects/example-project')
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [changeSet(proposalID, 'Steady lantern')] }))))
    const user = userEvent.setup()
    render(<App />)

    await selectShowing(user, 'Steady lantern')
    await user.click(screen.getByRole('button', { name: 'New changes' }))
    expect(window.location.pathname).toBe('/projects/example-project')
    expect(screen.getByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    await user.type(screen.getByLabelText('Name'), 'Local direction')

    act(() => window.history.back())
    let guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    expect(window.location.pathname).toBe('/projects/example-project')
    await user.click(within(guard).getByRole('button', { name: 'Keep editing' }))
    expect(screen.getByLabelText('Name')).toHaveValue('Local direction')
    expect(window.location.pathname).toBe('/projects/example-project')

    act(() => window.history.back())
    guard = await screen.findByRole('dialog', { name: 'Leave without keeping?' })
    await user.click(within(guard).getByRole('button', { name: 'Leave without keeping' }))
    expect(await screen.findByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    expect(window.location.pathname).toBe(proposalPath)

    act(() => window.history.back())
    expect(await screen.findByRole('button', { name: 'Showing Accepted' })).toBeInTheDocument()
    expect(window.location.pathname).toBe('/projects/example-project')
  })

  it('switches Accepted, valid, and invalid contexts locally without overlaying snapshots', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const validID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const invalidID = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
    const candidate = architecture({
      component_titles: ['Proposed worker', 'External'],
      components: [
        { id: worker, title: 'Proposed worker', description: 'Only in the proposal.\n', filename: 'worker.md', relationships: [] },
        { id: external, title: 'External', description: 'Elsewhere.\n', filename: 'external.md', relationships: [] },
      ],
    })
    const fetchMock = vi.fn(() => response(architecture({ change_sets: [
      changeSet(validID, 'Steady lantern', { candidate }),
      changeSet(invalidID, 'Broken compass', {
        valid: false, candidate: undefined, validation_code: 'relationship_target_invalid', validation_item: worker,
        validation_relationship_position: 1, validation_relationship_field: 'target',
        components: [{ id: worker, title: 'Worker', description: 'Exact facts.\n', new: false, relationships: [{ target_id: 'not-a-uuid', label: 'calls' }] }],
      }),
    ] })))
    vi.stubGlobal('fetch', fetchMock)
    const user = userEvent.setup()
    const { container } = render(<App />)

    expect(await screen.findByText('Does work.')).toBeInTheDocument()
    const pane = container.querySelector<HTMLElement>('.working-pane')!
    pane.scrollTop = 120
    await selectShowing(user, 'Steady lantern')
    expect(screen.getByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
    expect(screen.getByText('These changes have not updated Architecture yet.')).toBeInTheDocument()
    expect(pane.scrollTop).toBe(0)
    await user.click(within(screen.getByRole('navigation', { name: 'Diagrams and components' })).getByRole('button', { name: 'Proposed worker' }))
    expect(await screen.findByText('Only in the proposal.')).toBeInTheDocument()
    expect(screen.queryByText('Does work.')).not.toBeInTheDocument()

    await selectShowing(user, 'Broken compass')
    expect(await screen.findByRole('heading', { name: 'Needs correction' })).toBeInTheDocument()
    expect(screen.queryByTestId('architecture-map')).not.toBeInTheDocument()
    expect(screen.queryByText('Does work.')).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('navigates candidate-only and modified Diagrams from one proposal snapshot', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const validID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const accepted = architecture()
    const acceptedDiagrams = accepted.diagrams as Array<Record<string, any>>
    const candidate = architecture({
      component_count: 3,
      component_titles: ['Worker', 'External', 'Candidate service'],
      components: [
        ...(accepted.components as Array<Record<string, unknown>>),
        { id: candidateComponent, title: 'Candidate service', description: 'Exists only in the proposal.\n', filename: 'candidate-service.md', relationships: [] },
      ],
      diagrams: [
        acceptedDiagrams[0],
        {
          ...acceptedDiagrams[1],
          appearances: [{ component_id: worker, role: 'reference' }],
          boundaries: [], relationships: [],
        },
        {
          id: candidateDetail, title: 'Candidate nested', filename: 'candidate-nested.yaml', depth: 1,
          parent_diagram_id: root, parent_anchor_component_id: worker,
          breadcrumbs: [{ id: root, title: 'System' }, { id: candidateDetail, title: 'Candidate nested' }],
          appearances: [{ component_id: candidateComponent, role: 'home' }], boundaries: [], relationships: [],
        },
      ],
    })
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [changeSet(validID, 'Unified proposal', { candidate })] }))))
    const user = userEvent.setup()
    render(<App />)

    await selectShowing(user, 'Unified proposal')
    const navigation = screen.getByRole('navigation', { name: 'Diagrams and components' })
    await user.click(within(navigation).getByRole('button', { name: 'Candidate nested' }))
    expect(await screen.findByText('Exists only in the proposal.')).toBeInTheDocument()
    expect(within(navigation).getByRole('button', { name: 'Candidate service' })).toHaveAttribute('aria-current', 'page')
    await waitFor(() => expect(graphHarness.calls.at(-1)?.elements?.some((element) => element.data.id === candidateComponent)).toBe(true))

    await user.click(within(navigation).getByRole('button', { name: 'Detail' }))
    expect(await screen.findByText('Does work.')).toBeInTheDocument()
    expect(within(navigation).getByRole('button', { name: 'Worker, Included here · Lives in System' })).toHaveAttribute('aria-current', 'page')
    expect(within(navigation).queryByRole('button', { name: 'External' })).not.toBeInTheDocument()
    await waitFor(() => {
      const elements = graphHarness.calls.at(-1)?.elements ?? []
      expect(elements.some((element) => element.data.id === worker)).toBe(true)
      expect(elements.some((element) => element.data.id === external)).toBe(false)
    })
  })

  it('protects unsaved proposal Markdown before switching contexts and renders it inertly', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const firstID = 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
    const secondID = 'cccccccc-cccc-4ccc-8ccc-cccccccccccc'
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [
      changeSet(firstID, 'Steady lantern', { proposal_markdown: '# Plan\n\n<script>alert(1)</script>' }),
      changeSet(secondID, 'Quiet harbor'),
    ] }))))
    const user = userEvent.setup()
    const { container } = render(<App />)

    await selectShowing(user, 'Steady lantern')
    expect(screen.getByRole('heading', { name: 'Plan' })).toBeInTheDocument()
    expect(container.querySelector('script')).toBeNull()
    await user.type(screen.getByRole('textbox', { name: 'Proposal' }), '\nUnsent')
    await selectShowing(user, 'Quiet harbor')
    expect(screen.getByRole('dialog', { name: 'Leave without keeping?' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Steady lantern' })).toBeInTheDocument()
  })

  it('keeps an out-of-date proposal editable and reviewable but unable to update Accepted', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const reviewed = reviewedArchitecture()
    const proposal = (reviewed as unknown as { changes: Record<string, unknown> }).changes
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [{ ...proposal, out_of_date: true }] }))))
    const user = userEvent.setup()
    const { container } = render(<App />)

    await selectShowing(user, 'Steady lantern · Out of date')
    expect(await screen.findByText('Out of date with Accepted. You can still edit and review this proposal, but it cannot update Architecture until it matches Accepted.')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'With changes' })).not.toBeInTheDocument()
    const pane = container.querySelector<HTMLElement>('.working-pane')!
    pane.scrollTop = 100
    await user.click(screen.getByRole('button', { name: 'Return to review' }))
    expect(pane.scrollTop).toBe(0)
    expect(screen.getByRole('button', { name: 'With changes' })).toBeInTheDocument()
    expect(screen.getByText('Out of date with Accepted. You can inspect this review, but it cannot update Architecture until the proposal matches Accepted.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back to proposal' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
    const submit = screen.getByRole('button', { name: 'Submit review' })
    expect(submit).toBeDisabled()
    expect(submit).toHaveAccessibleDescription('Enter a reviewer name.')
    await user.click(screen.getByRole('button', { name: 'Write review' }))
    expect(screen.getByRole('heading', { name: 'Submit feedback' })).toHaveFocus()
    await user.type(screen.getByRole('textbox', { name: 'Reviewer name' }), 'Reviewer')
    expect(submit).toBeDisabled()
    expect(submit).toHaveAccessibleDescription('Add a review summary or a comment to submit a Comment review.')
    await user.selectOptions(screen.getByRole('combobox', { name: 'Conclusion' }), 'approve')
    expect(submit).toBeEnabled()
    expect(submit).not.toHaveAttribute('aria-describedby')
  })

  it('shows an accepted proposal read-only when selected', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const appliedID = 'dddddddd-dddd-4ddd-8ddd-dddddddddddd'
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [changeSet(appliedID, 'Shipped bridge', {
      lifecycle: 'applied', read_only: true, proposal_markdown: '## Why\n\nDurable rationale.', applied_revision: 'b'.repeat(40),
    })] }))))
    const user = userEvent.setup()
    render(<App />)

    await selectShowing(user, 'Shipped bridge')
    expect(screen.getByText('Accepted proposal')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Shipped bridge' })).toBeInTheDocument()
    expect(screen.getByText('This is the proposal that updated Architecture. It cannot be changed.')).toBeInTheDocument()
    expect(screen.getByText('Durable rationale.')).toBeInTheDocument()
    expect(screen.queryByLabelText('Name')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Review changes' })).not.toBeInTheDocument()
    expect(screen.getByText('Technical details').closest('details')).not.toHaveAttribute('open')
  })

  it('groups Showing choices and disambiguates duplicate visible names by stable identity', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const firstID = 'dddddddd-dddd-4ddd-8ddd-dddddddddddd'
    const secondID = 'eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee'
    const activeID = 'ffffffff-ffff-4fff-8fff-ffffffffffff'
    const staleID = '99999999-9999-4999-8999-999999999999'
    vi.stubGlobal('fetch', vi.fn(() => response(architecture({ change_sets: [
      changeSet(activeID, 'Open work'),
      changeSet(staleID, 'Older work', { out_of_date: true }),
      changeSet(firstID, 'Reusable', { lifecycle: 'applied', read_only: true, applied_revision: 'b'.repeat(40) }),
      changeSet(secondID, 'Reusable', { lifecycle: 'applied', read_only: true, applied_revision: 'c'.repeat(40) }),
    ] }))))
    const user = userEvent.setup()
    render(<App />)

    const trigger = await screen.findByRole('button', { name: 'Showing Accepted' })
    expect(trigger).toHaveAttribute('aria-haspopup', 'listbox')
    await user.click(trigger)
    const listbox = screen.getByRole('listbox', { name: 'Showing' })
    expect(listbox).toHaveAttribute('aria-activedescendant')
    expect(within(listbox).getByRole('option', { name: 'Accepted' })).toHaveAttribute('aria-selected', 'true')
    const open = within(listbox).getByRole('group', { name: 'Open proposals' })
    expect(within(open).getByRole('option', { name: 'Open work' })).toBeInTheDocument()
    expect(within(open).getByRole('option', { name: 'Older work · Out of date' })).toBeInTheDocument()
    const accepted = within(listbox).getByRole('group', { name: 'Accepted proposals' })
    expect(within(accepted).getByRole('option', { name: `Reusable · ${firstID.slice(0, 8)}` })).toBeInTheDocument()
    expect(within(accepted).getByRole('option', { name: `Reusable · ${secondID.slice(0, 8)}` })).toBeInTheDocument()
    await user.keyboard('{ArrowDown}{Enter}')
    expect(trigger).toHaveAccessibleName('Showing Open work')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByRole('heading', { name: 'Open work' })).toBeInTheDocument()
    expect(trigger).toHaveFocus()
    await user.keyboard('{ArrowUp}')
    const reopened = screen.getByRole('listbox', { name: 'Showing' })
    const preceding = within(reopened).getByRole('option', { name: 'Accepted' })
    expect(reopened).toHaveAttribute('aria-activedescendant', preceding.id)
    await user.keyboard('{Enter}')
    expect(trigger).toHaveAccessibleName('Showing Accepted')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })

  it('explains an empty review without offering an Accepted update', async () => {
    window.history.replaceState({}, '', '/projects/example-project')
    const value = reviewedArchitecture()
    const review = (value as unknown as { changes: { review: { diff: string } } }).changes.review
    review.diff = ''
    vi.stubGlobal('fetch', vi.fn(() => response(value)))
    const user = userEvent.setup()
    render(<App />)

    await selectProposal(user)
    expect(await screen.findByText('There is no Architecture change to accept.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Update architecture' })).not.toBeInTheDocument()
  })
})
