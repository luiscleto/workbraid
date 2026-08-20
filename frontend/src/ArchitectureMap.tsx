import cytoscape, { Core, ElementDefinition } from 'cytoscape'
import { useEffect, useMemo, useRef, useState } from 'react'

export type MapRelationship = {
  target_id: string
  label: string
  projection_key?: string
}

export type MapComponent = {
  id: string
  component_id?: string
  title: string
  filename?: string
  node_kind?: 'home' | 'reference' | 'boundary'
  subtitle?: string
  relationships: MapRelationship[]
}

export type ReviewMapComponentChange = {
  component_id: string
  status: 'added' | 'content_changed'
  path: string
}

export type ReviewMapRelationshipChange = {
  key: string
  before_key?: string
  source_id: string
  target_id: string
  label: string
  status: 'added' | 'removed'
  path: string
  occurrence: number
}

export type ReviewRelationshipSelection = ReviewMapRelationshipChange & {
  source_title: string
  target_title: string
}

type ArchitectureMapProps = {
  revision: string
  components: MapComponent[]
  selectedID?: string
  onSelect: (id: string) => void
  layoutComponentIDs?: string[]
  reviewSide?: 'with' | 'before'
  reviewComponents?: ReviewMapComponentChange[]
  reviewRelationships?: ReviewMapRelationshipChange[]
  selectedRelationshipKey?: string
  onSelectRelationship?: (relationship: ReviewRelationshipSelection) => void
}

type ProjectionOptions = Pick<ArchitectureMapProps, 'layoutComponentIDs' | 'reviewSide' | 'reviewComponents' | 'reviewRelationships'>

export function ArchitectureMap({
  revision,
  components,
  selectedID,
  onSelect,
  layoutComponentIDs,
  reviewSide,
  reviewComponents = [],
  reviewRelationships = [],
  selectedRelationshipKey,
  onSelectRelationship,
}: ArchitectureMapProps) {
  const container = useRef<HTMLDivElement>(null)
  const graph = useRef<Core | null>(null)
  const selectHandler = useRef(onSelect)
  const relationshipHandler = useRef(onSelectRelationship)
  const [renderFailed, setRenderFailed] = useState(false)
  const layoutKey = [...(layoutComponentIDs ?? components.map((component) => component.id))].sort().join('\u0000')
  // A revision-pinned projection intentionally ignores response-object churn
  // caused by pending edits at the same accepted revision. A review revision is
  // the bound candidate tree or base commit and carries one stable layout basis.
  const elements = useMemo(() => projectionElements(components, {
    layoutComponentIDs,
    reviewSide,
    reviewComponents,
    reviewRelationships,
  }), [revision, reviewSide, layoutKey])

  useEffect(() => {
    selectHandler.current = onSelect
    relationshipHandler.current = onSelectRelationship
  }, [onSelect, onSelectRelationship])

  useEffect(() => {
    if (!container.current) return
    setRenderFailed(false)
    let instance: Core | null = null
    try {
      instance = cytoscape({
        container: container.current,
        elements,
        layout: { name: 'preset', animate: false, fit: true, padding: 34 },
        minZoom: 0.35,
        maxZoom: 2.5,
        style: mapStyles,
      })
      instance.on('tap', 'node', (event) => selectHandler.current(event.target.id()))
      instance.on('tap', 'edge', (event) => {
        const data = event.target.data() as ReviewRelationshipSelection & { reviewStatus?: string }
        if (data.reviewStatus && relationshipHandler.current) {
          relationshipHandler.current({
            key: data.key,
            before_key: data.before_key,
            source_id: data.source_id,
            target_id: data.target_id,
            label: data.label,
            status: data.status,
            path: data.path,
            occurrence: data.occurrence,
            source_title: data.source_title,
            target_title: data.target_title,
          })
        }
      })
      graph.current = instance
    } catch {
      graph.current = null
      setRenderFailed(true)
    }
    return () => {
      graph.current = null
      instance?.destroy()
    }
  }, [elements])

  useEffect(() => {
    const instance = graph.current
    if (!instance) return
    instance.$(':selected').unselect()
    if (selectedRelationshipKey) instance.getElementById(selectedRelationshipKey).select()
    else if (selectedID) instance.getElementById(selectedID).select()
  }, [selectedID, selectedRelationshipKey])

  const reviewControls = reviewSide ? (
    <ReviewChangeControls
      side={reviewSide}
      components={components}
      componentChanges={reviewComponents}
      relationshipChanges={reviewRelationships}
      onSelectComponent={onSelect}
      onSelectRelationship={onSelectRelationship}
    />
  ) : null

  if (components.length === 0) {
    return (
      <div className="map-empty">
        {reviewSide === 'before' ? 'Before changes has no components.' : 'The architecture has no components yet.'}
        {reviewControls}
      </div>
    )
  }

  return (
    <section className="map-surface" aria-label={reviewSide ? `${reviewSide === 'with' ? 'With changes' : 'Before changes'} architecture map` : 'Architecture map'}>
      {renderFailed ? (
        <div className="map-failure" role="alert">
          <strong>The architecture map could not be shown.</strong>
          <span>{reviewSide ? 'You can still inspect the complete change and update the architecture.' : 'Use the component list to keep working.'}</span>
        </div>
      ) : <div ref={container} className="map-canvas" data-testid="architecture-map" />}
      {!renderFailed && <button className="map-fit" type="button" onClick={() => graph.current?.fit(undefined, 34)}>Fit map</button>}
      {reviewControls}
    </section>
  )
}

function ReviewChangeControls({
  side,
  components,
  componentChanges,
  relationshipChanges,
  onSelectComponent,
  onSelectRelationship,
}: {
  side: 'with' | 'before'
  components: MapComponent[]
  componentChanges: ReviewMapComponentChange[]
  relationshipChanges: ReviewMapRelationshipChange[]
  onSelectComponent: (id: string) => void
  onSelectRelationship?: (relationship: ReviewRelationshipSelection) => void
}) {
  const titles = new Map(components.map((component) => [component.id, component.title]))
  const visibleComponentChanges = componentChanges.filter((change) => titles.has(change.component_id))
  const visibleRelationshipChanges = relationshipChanges.filter((change) => side === 'with' || change.status === 'removed')
  const facts = new Map<string, number>()
  for (const relationship of visibleRelationshipChanges) {
    const fact = `${relationship.status}\u0000${relationship.source_id}\u0000${relationship.target_id}\u0000${relationship.label}`
    facts.set(fact, (facts.get(fact) ?? 0) + 1)
  }
  if (!visibleComponentChanges.length && !visibleRelationshipChanges.length) return null
  return (
    <div className="map-review-controls" aria-label="Visual changes">
      <p className="map-review-key"><span>＋ Added</span><span>△ Content changed</span><span>− − Removed relationship</span></p>
      <ul>
        {visibleComponentChanges.map((change) => {
          const title = titles.get(change.component_id)
          if (!title) return null
          return <li key={change.component_id}><button type="button" onClick={() => onSelectComponent(change.component_id)}>{componentStatusLabel(change.status)}: {title}</button></li>
        })}
        {visibleRelationshipChanges.map((change) => {
          const sourceTitle = titles.get(change.source_id) ?? change.source_id
          const targetTitle = titles.get(change.target_id) ?? change.target_id
          const fact = `${change.status}\u0000${change.source_id}\u0000${change.target_id}\u0000${change.label}`
          const duplicateContext = (facts.get(fact) ?? 0) > 1 ? `, occurrence ${change.occurrence}` : ''
          const selection = { ...change, ...(side === 'before' && change.before_key ? { key: change.before_key } : {}), source_title: sourceTitle, target_title: targetTitle }
          return <li key={change.key}><button type="button" onClick={() => onSelectRelationship?.(selection)}>{relationshipStatusLabel(change.status)}{duplicateContext}: {sourceTitle} — {change.label} — {targetTitle}</button></li>
        })}
      </ul>
    </div>
  )
}

export function projectionElements(components: MapComponent[], options: ProjectionOptions = {}): ElementDefinition[] {
  const positions = deterministicPositions(options.layoutComponentIDs ?? components.map((component) => component.id))
  const componentStatus = new Map(options.reviewComponents?.map((change) => [change.component_id, change.status]))
  const relationshipStatus = new Map<string, ReviewMapRelationshipChange>()
  for (const change of options.reviewRelationships ?? []) {
    if (change.status === 'added') relationshipStatus.set(change.key, change)
    if (change.status === 'removed' && change.before_key) relationshipStatus.set(change.before_key, change)
  }
  const titleByID = new Map(components.map((component) => [component.id, component.title]))
  const nodes: ElementDefinition[] = components.map((component) => {
    const status = componentStatus.get(component.id) ?? (options.reviewSide ? 'unchanged' : '')
    return {
      data: {
        id: component.id,
        label: component.title,
        displayLabel: status === 'added' ? `${component.title}\n＋ Added` : status === 'content_changed' ? `${component.title}\n△ Content changed` : [component.title, component.subtitle].filter(Boolean).join('\n'),
        nodeKind: component.node_kind ?? '',
        reviewStatus: status,
      },
      position: positions[component.id],
    }
  })
  const grouped = new Map<string, number>()
  for (const source of components) {
    for (const relationship of source.relationships ?? []) {
      const key = `${source.id}\u0000${relationship.target_id}`
      grouped.set(key, (grouped.get(key) ?? 0) + 1)
    }
  }
  const seen = new Map<string, number>()
  const edges: ElementDefinition[] = []
  for (const source of components) {
    for (let relationshipIndex = 0; relationshipIndex < (source.relationships ?? []).length; relationshipIndex += 1) {
      const relationship = source.relationships[relationshipIndex]
      const pair = `${source.id}\u0000${relationship.target_id}`
      const index = seen.get(pair) ?? 0
      seen.set(pair, index + 1)
      const count = grouped.get(pair) ?? 1
      const key = relationship.projection_key ?? `projection:${source.id}:${relationship.target_id}:${index}`
      const change = relationshipStatus.get(key)
      const status = change?.status ?? (options.reviewSide ? 'unchanged' : '')
      edges.push({
        data: {
          id: key,
          key,
          source: source.id,
          target: relationship.target_id,
          source_id: source.id,
          target_id: relationship.target_id,
          source_title: source.title,
          target_title: titleByID.get(relationship.target_id) ?? relationship.target_id,
          label: relationship.label,
          displayLabel: status === 'added' ? `Added — ${relationship.label}` : status === 'removed' ? `Removed — ${relationship.label}` : relationship.label,
          distance: count === 1 ? 0 : (index - (count - 1) / 2) * 52,
          reviewStatus: status,
          status,
          path: change?.path,
          occurrence: change?.occurrence,
          before_key: change?.before_key,
        },
      })
    }
  }
  if (options.reviewSide === 'with') {
    const removed = (options.reviewRelationships ?? []).filter((change) => change.status === 'removed')
    const removedGrouped = new Map<string, number>()
    for (const change of removed) {
      const pair = `${change.source_id}\u0000${change.target_id}`
      removedGrouped.set(pair, (removedGrouped.get(pair) ?? 0) + 1)
    }
    const removedSeen = new Map<string, number>()
    for (const change of removed) {
      const pair = `${change.source_id}\u0000${change.target_id}`
      const index = removedSeen.get(pair) ?? 0
      removedSeen.set(pair, index + 1)
      const count = removedGrouped.get(pair) ?? 1
      edges.push({
        data: {
          id: change.key,
          key: change.key,
          source: change.source_id,
          target: change.target_id,
          source_id: change.source_id,
          target_id: change.target_id,
          source_title: titleByID.get(change.source_id) ?? change.source_id,
          target_title: titleByID.get(change.target_id) ?? change.target_id,
          label: change.label,
          displayLabel: `Removed — ${change.label}`,
          distance: count === 1 ? 78 : 78 + index * 42,
          reviewStatus: 'removed',
          status: 'removed',
          path: change.path,
          occurrence: change.occurrence,
          before_key: change.before_key,
          annotation: true,
        },
      })
    }
  }
  return [...nodes, ...edges]
}

export function deterministicPositions(componentIDs: string[]): Record<string, { x: number; y: number }> {
  const ids = [...new Set(componentIDs)].sort()
  const positions: Record<string, { x: number; y: number }> = {}
  if (ids.length === 1) {
    positions[ids[0]] = { x: 0, y: 0 }
    return positions
  }
  const radius = Math.max(150, ids.length * 38)
  ids.forEach((id, index) => {
    const angle = -Math.PI / 2 + (index * Math.PI * 2) / ids.length
    positions[id] = {
      x: Math.round(Math.cos(angle) * radius * 1000) / 1000,
      y: Math.round(Math.sin(angle) * radius * 1000) / 1000,
    }
  })
  return positions
}

function componentStatusLabel(status: ReviewMapComponentChange['status']) {
  return status === 'added' ? 'Added component' : 'Content changed'
}

function relationshipStatusLabel(status: ReviewMapRelationshipChange['status']) {
  return status === 'added' ? 'Added relationship' : 'Removed relationship'
}

const mapStyles: cytoscape.StylesheetJson = [
  {
    selector: 'node',
    style: {
      'background-color': '#f8f0dc',
      'border-color': '#27251f',
      'border-width': 1.5,
      color: '#27251f',
      label: 'data(displayLabel)',
      'font-family': 'IBM Plex Sans',
      'font-size': 13,
      'text-wrap': 'wrap',
      'text-max-width': '128px',
      width: 116,
      height: 54,
      shape: 'round-rectangle',
    },
  },
  { selector: 'node[reviewStatus = "unchanged"]', style: { opacity: 0.48, 'border-style': 'dotted' } },
  { selector: 'node[reviewStatus = "added"]', style: { 'background-color': '#d8eadf', 'border-color': '#126747', 'border-width': 3, shape: 'hexagon' } },
  { selector: 'node[reviewStatus = "content_changed"]', style: { 'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-width': 3, 'border-style': 'dashed' } },
  { selector: 'node[nodeKind = "reference"]', style: { 'border-style': 'dashed', 'background-color': '#eee3c8' } },
  { selector: 'node[nodeKind = "boundary"]', style: { shape: 'diamond', 'border-style': 'dotted', 'background-color': '#efe7d3', color: '#5e584b', width: 104, height: 62 } },
  { selector: 'node:selected', style: { 'background-color': '#e7dba9', 'border-color': '#18734f', 'border-width': 4, opacity: 1 } },
  { selector: 'node[reviewStatus = "unchanged"]:selected', style: { 'background-color': '#f8f0dc', 'border-color': '#27251f', 'border-width': 5, 'border-style': 'dotted', opacity: 1 } },
  { selector: 'node[reviewStatus = "added"]:selected', style: { 'background-color': '#d8eadf', 'border-color': '#126747', 'border-width': 5, shape: 'hexagon', opacity: 1 } },
  { selector: 'node[reviewStatus = "content_changed"]:selected', style: { 'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-width': 5, 'border-style': 'dashed', opacity: 1 } },
  {
    selector: 'edge',
    style: {
      width: 1.25,
      'line-color': '#736c5c',
      'target-arrow-color': '#736c5c',
      'target-arrow-shape': 'triangle',
      'curve-style': 'unbundled-bezier',
      'control-point-distances': 'data(distance)',
      'control-point-weights': 0.5,
      label: 'data(displayLabel)',
      color: '#514c41',
      'font-family': 'IBM Plex Sans',
      'font-size': 10,
      'text-background-color': '#f4ecd8',
      'text-background-opacity': 1,
      'text-background-padding': '2px',
      'text-rotation': 'autorotate',
    },
  },
  { selector: 'edge[reviewStatus = "unchanged"]', style: { opacity: 0.38, 'line-style': 'dotted' } },
  { selector: 'edge[reviewStatus = "added"]', style: { width: 3, 'line-color': '#126747', 'target-arrow-color': '#126747' } },
  { selector: 'edge[reviewStatus = "removed"]', style: { width: 2.5, 'line-color': '#a04432', 'target-arrow-color': '#a04432', 'line-style': 'dashed', opacity: 0.82 } },
  { selector: 'edge:selected', style: { width: 4, opacity: 1, 'line-color': '#18734f', 'target-arrow-color': '#18734f' } },
  { selector: 'edge[reviewStatus = "unchanged"]:selected', style: { width: 4.5, opacity: 1, 'line-color': '#736c5c', 'target-arrow-color': '#736c5c', 'line-style': 'dotted' } },
  { selector: 'edge[reviewStatus = "added"]:selected', style: { width: 4.5, opacity: 1, 'line-color': '#126747', 'target-arrow-color': '#126747' } },
  { selector: 'edge[reviewStatus = "removed"]:selected', style: { width: 4.5, opacity: 1, 'line-color': '#a04432', 'target-arrow-color': '#a04432', 'line-style': 'dashed' } },
]
