import { expect, it } from 'vitest'
import { deterministicPositions, displayPositions, projectionElements, roundPosition } from './ArchitectureMap'

it.each([[0.5, 1], [-0.5, -1], [-180.49, -180], [-180.5, -181], [100000, 100000]])('rounds model coordinate %s to %s', (value, expected) => {
  expect(roundPosition(value)).toBe(expected)
})

it('keeps every provided coordinate exact through overlap, boundary roles and label edits', () => {
  const components = [
    {id:'a',title:'A',relationships:[],position:{x:0,y:0}},
    {id:'b',title:'B',relationships:[],position:{x:0,y:0}},
    {id:'c',title:'C',relationships:[],position:{x:100,y:20}},
    {id:'boundary',title:'Elsewhere',node_kind:'boundary' as const,boundary_home_title:'A long nested Diagram title',relationships:[],position:{x:10,y:15}},
  ]
  const positions = displayPositions(components)
  expect(positions.a).toEqual({x:0,y:0})
  expect(positions.b).toEqual(positions.a)
  expect(displayPositions([...components].reverse())).toEqual(positions)
  expect(positions.c).toEqual({x:100,y:20})
  expect(positions.boundary).toEqual({x:10,y:15})
  expect(displayPositions(components.map(c=>({...c,title:'A much longer title',node_kind:'reference'})))).toEqual(positions)
})

it('projects disconnected, cyclic, and parallel accepted relationships with local edge keys', () => {
  const elements = projectionElements([
    { id: 'a', title: 'A', relationships: [{ target_id: 'b', label: 'calls' }, { target_id: 'b', label: 'reads from' }] },
    { id: 'b', title: 'B', relationships: [{ target_id: 'a', label: 'responds to' }] },
    { id: 'c', title: 'Disconnected', relationships: [] },
  ])
  const nodes = elements.filter((element) => !('source' in element.data))
  const edges = elements.filter((element) => 'source' in element.data)

  expect(nodes.map((node) => node.data.id)).toEqual(['a', 'b', 'c'])
  expect(edges).toHaveLength(3)
  expect(edges.map((edge) => edge.data.label)).toEqual(['calls', 'reads from', 'responds to'])
  expect(new Set(edges.map((edge) => edge.data.id)).size).toBe(3)
  expect(edges[0].data.distance).not.toBe(edges[1].data.distance)
})

it('uses one deterministic union-ID position basis across review sides', () => {
  const layoutIDs = ['worker', 'api', 'new-component']
  const before = projectionElements([
    { id: 'api', title: 'API', relationships: [] },
    { id: 'worker', title: 'Worker', relationships: [] },
  ], { reviewSide: 'before', layoutComponentIDs: layoutIDs })
  const withChanges = projectionElements([
    { id: 'api', title: 'API', relationships: [] },
    { id: 'worker', title: 'Worker', relationships: [] },
    { id: 'new-component', title: 'New', relationships: [] },
  ], { reviewSide: 'with', layoutComponentIDs: [...layoutIDs].reverse() })

  const positions = (elements: ReturnType<typeof projectionElements>) => new Map(elements.filter((element) => !('source' in element.data)).map((element) => [element.data.id, element.position]))
  expect(positions(before).get('api')).toEqual(positions(withChanges).get('api'))
  expect(positions(before).get('worker')).toEqual(positions(withChanges).get('worker'))
  expect(deterministicPositions(layoutIDs)).toEqual(deterministicPositions([...layoutIDs].reverse()))
})

it('gives derived boundaries distinct semantic slots that remain stable across review sides', () => {
  const layoutIDs = ['gateway', 'records', 'worker']
  const withChanges = projectionElements([
    { id: 'gateway', component_id: 'gateway', title: 'Gateway', relationships: [] },
    { id: 'boundary:worker', component_id: 'worker', title: 'Worker', node_kind: 'boundary', relationships: [] },
    { id: 'boundary:records', component_id: 'records', title: 'Records', node_kind: 'boundary', relationships: [] },
  ], { reviewSide: 'with', layoutComponentIDs: layoutIDs })
  const before = projectionElements([
    { id: 'gateway', component_id: 'gateway', title: 'Gateway', relationships: [] },
    { id: 'worker', component_id: 'worker', title: 'Worker', node_kind: 'home', relationships: [] },
    { id: 'boundary:records', component_id: 'records', title: 'Records', node_kind: 'boundary', relationships: [] },
  ], { reviewSide: 'before', layoutComponentIDs: [...layoutIDs].reverse() })

  const position = (elements: ReturnType<typeof projectionElements>, id: string) => elements.find((element) => element.data.id === id)?.position
  expect(position(withChanges, 'boundary:worker')).not.toEqual(position(withChanges, 'boundary:records'))
  expect(position(withChanges, 'boundary:worker')).toEqual(position(before, 'worker'))
  expect(position(withChanges, 'boundary:records')).toEqual(position(before, 'boundary:records'))
})

it('keeps active topology separate from exact added and removed occurrence annotations', () => {
  const components = [
    {
      id: 'api', title: 'API', relationships: [
        { target_id: 'worker', label: 'calls', projection_key: 'review:with:api:0' },
        { target_id: 'worker', label: 'calls', projection_key: 'review:with:api:1' },
      ],
    },
    { id: 'worker', title: 'Worker', relationships: [] },
  ]
  const relationships = [
    { key: 'review:with:api:1', source_id: 'api', target_id: 'worker', label: 'calls', status: 'added' as const, path: 'components/api.md', occurrence: 2 },
    { key: 'review:removed:api:2', before_key: 'review:before:api:2', source_id: 'api', target_id: 'worker', label: 'reads', status: 'removed' as const, path: 'components/api.md', occurrence: 1 },
  ]

  const elements = projectionElements(components, {
    reviewSide: 'with',
    layoutComponentIDs: ['api', 'worker'],
    reviewComponents: [{ component_id: 'api', status: 'content_changed', path: 'components/api.md' }],
    reviewRelationships: relationships,
  })
  const edges = elements.filter((element) => 'source' in element.data)

  expect(edges).toHaveLength(3)
  expect(edges.find((edge) => edge.data.id === 'review:with:api:1')?.data.reviewStatus).toBe('added')
  expect(edges.find((edge) => edge.data.id === 'review:removed:api:2')?.data.annotation).toBe(true)
  expect(edges.find((edge) => edge.data.id === 'review:removed:api:2')?.data.displayLabel).toBe('Removed — reads')
  expect(elements.find((element) => element.data.id === 'api')?.data.displayLabel).toBe('API')
})

it('maps v2 relationship deltas to exact selected-Diagram internal and boundary edges', () => {
  const root = 'root'
  const detail = 'detail'
  const changes = [
    {
      key: 'review:with:worker:1', source_id: 'worker', target_id: 'gateway', source_title: 'Worker', target_title: 'Gateway',
      label: 'reports', status: 'added' as const, path: 'components/worker.md', occurrence: 1,
      diagram_projections: [
        { side: 'with' as const, diagram_id: root, key: 'diagram:root:worker:1', source_node_key: 'worker', target_node_key: 'gateway' },
        { side: 'with' as const, diagram_id: detail, key: 'diagram:detail:worker:1', source_node_key: 'worker', target_node_key: 'boundary:gateway' },
      ],
    },
    {
      key: 'review:removed:worker:1', before_key: 'review:before:worker:1', source_id: 'worker', target_id: 'records', source_title: 'Worker', target_title: 'Records',
      label: 'writes', status: 'removed' as const, path: 'components/worker.md', occurrence: 2,
      diagram_projections: [
        { side: 'before' as const, diagram_id: root, key: 'diagram:root:worker:1', source_node_key: 'worker', target_node_key: 'records' },
        { side: 'before' as const, diagram_id: detail, key: 'diagram:detail:worker:1', source_node_key: 'worker', target_node_key: 'boundary:records' },
      ],
    },
  ]
  const candidateDetail = projectionElements([
    { id: 'worker', component_id: 'worker', title: 'Worker', relationships: [{ target_id: 'boundary:gateway', label: 'reports', projection_key: 'diagram:detail:worker:1' }] },
    { id: 'boundary:gateway', component_id: 'gateway', title: 'Gateway', node_kind: 'boundary', boundary_home_title: 'System', relationships: [] },
  ], { reviewSide: 'with', reviewDiagramID: detail, reviewRelationships: changes })
  const addedBoundary = candidateDetail.find((element) => element.data.id === 'diagram:detail:worker:1')
  expect(addedBoundary?.data).toMatchObject({ reviewStatus: 'added', source: 'worker', target: 'boundary:gateway', source_id: 'worker', target_id: 'gateway', source_title: 'Worker', target_title: 'Gateway' })
  expect(candidateDetail.find((element) => element.data.id === 'boundary:gateway')?.data).toMatchObject({ label: 'Gateway', boundaryHomeTitle: 'System' })
  expect(candidateDetail.find((element) => element.data.id === 'boundary:gateway')?.data.displayLabel).toContain('…')
  expect(candidateDetail.filter((element) => 'source' in element.data)).toHaveLength(1)
  expect(candidateDetail.find((element) => element.data.id === 'worker')?.data.reviewStatus).toBe('unchanged')

  const baseDetail = projectionElements([
    { id: 'worker', component_id: 'worker', title: 'Worker', node_kind: 'reference', relationships: [{ target_id: 'boundary:records', label: 'writes', projection_key: 'diagram:detail:worker:1' }] },
    { id: 'boundary:records', component_id: 'records', title: 'Records', node_kind: 'boundary', boundary_home_title: 'Data', relationships: [] },
  ], { reviewSide: 'before', reviewDiagramID: detail, reviewRelationships: changes })
  expect(baseDetail.find((element) => element.data.id === 'diagram:detail:worker:1')?.data).toMatchObject({ reviewStatus: 'removed', source: 'worker', target: 'boundary:records', source_id: 'worker', target_id: 'records' })
  expect(baseDetail.find((element) => element.data.id === 'worker')?.data).toMatchObject({ displayLabel: 'Worker', nodeKind: 'reference' })
  expect(baseDetail.find((element) => element.data.id === 'boundary:records')?.data).toMatchObject({ label: 'Records', boundaryHomeTitle: 'Data', nodeKind: 'boundary' })
  expect(baseDetail.find((element) => element.data.id === 'boundary:records')?.data.displayLabel).toContain('…')

  const candidateRoot = projectionElements([
    { id: 'worker', component_id: 'worker', title: 'Worker', relationships: [{ target_id: 'gateway', label: 'reports', projection_key: 'diagram:root:worker:1' }] },
    { id: 'gateway', component_id: 'gateway', title: 'Gateway', relationships: [] },
  ], { reviewSide: 'with', reviewDiagramID: root, reviewRelationships: changes })
  expect(candidateRoot.find((element) => element.data.id === 'diagram:root:worker:1')?.data).toMatchObject({ reviewStatus: 'added', source: 'worker', target: 'gateway' })

  const baseRoot = projectionElements([
    { id: 'worker', component_id: 'worker', title: 'Worker', relationships: [{ target_id: 'records', label: 'writes', projection_key: 'diagram:root:worker:1' }] },
    { id: 'records', component_id: 'records', title: 'Records', relationships: [] },
  ], { reviewSide: 'before', reviewDiagramID: root, reviewRelationships: changes })
  expect(baseRoot.find((element) => element.data.id === 'diagram:root:worker:1')?.data).toMatchObject({ reviewStatus: 'removed', source: 'worker', target: 'records' })
})
