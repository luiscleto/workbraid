import cytoscape, { Core, ElementDefinition } from 'cytoscape'
import { type ReactNode, useEffect, useMemo, useRef, useState } from 'react'

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
  source_title?: string
  target_title?: string
  label: string
  status: 'added' | 'removed'
  path: string
  occurrence: number
  diagram_projections?: ReviewDiagramRelationshipProjection[]
}

type ReviewDiagramRelationshipProjection = {
  side: 'with' | 'before'
  diagram_id: string
  key: string
  source_node_key: string
  target_node_key: string
}

export type ReviewRelationshipSelection = ReviewMapRelationshipChange & {
  source_title: string
  target_title: string
  review_side?: 'with' | 'before'
}

type ArchitectureMapProps = {
  revision: string
  components: MapComponent[]
  selectedID?: string
  onSelect: (id: string) => void
  emptyMessage?: string
  layoutComponentIDs?: string[]
  reviewSide?: 'with' | 'before'
  reviewComponents?: ReviewMapComponentChange[]
  reviewRelationships?: ReviewMapRelationshipChange[]
  reviewDiagramID?: string
  selectedRelationshipKey?: string
  onSelectRelationship?: (relationship: ReviewRelationshipSelection) => void
  externalReferences?: ReactNode
}

type ProjectionOptions = Pick<ArchitectureMapProps, 'layoutComponentIDs' | 'reviewSide' | 'reviewComponents' | 'reviewRelationships' | 'reviewDiagramID'>

export function ArchitectureMap({
  revision,
  components,
  selectedID,
  onSelect,
  emptyMessage,
  layoutComponentIDs,
  reviewSide,
  reviewComponents = [],
  reviewRelationships = [],
  reviewDiagramID,
  selectedRelationshipKey,
  onSelectRelationship,
  externalReferences,
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
    reviewDiagramID,
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
            review_side: data.review_side,
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
      reviewDiagramID={reviewDiagramID}
      onSelectComponent={onSelect}
      onSelectRelationship={onSelectRelationship}
    />
  ) : null
  const hasExternalReferences = Boolean(externalReferences)
  const hasReviewControls = Boolean(reviewControls)
  const reviewDockIdentity = reviewSide ? revision : ''
  const [dockPane, setDockPane] = useState<'changes' | 'external'>('changes')
  const [dockCollapsed, setDockCollapsed] = useState(false)

  useEffect(() => {
    if (reviewDockIdentity && hasReviewControls) {
      setDockPane('changes')
      return
    }
    if (!hasReviewControls && hasExternalReferences) setDockPane('external')
  }, [reviewDockIdentity, hasReviewControls, hasExternalReferences])

  const visibleDockPane = dockPane === 'changes' && hasReviewControls
    ? 'changes'
    : hasExternalReferences ? 'external' : 'changes'
  const bottomDock = hasReviewControls || hasExternalReferences ? (
    <div className={`map-bottom-dock ${dockCollapsed ? 'collapsed' : ''}`}>
      <div className="map-bottom-dock-header">
        {hasReviewControls && hasExternalReferences ? (
          <div className="map-bottom-dock-tabs" role="tablist" aria-label="Map information">
            <button type="button" role="tab" aria-selected={visibleDockPane === 'changes'} aria-controls="map-bottom-dock-panel" onClick={() => setDockPane('changes')}>Changes</button>
            <button type="button" role="tab" aria-selected={visibleDockPane === 'external'} aria-controls="map-bottom-dock-panel" onClick={() => setDockPane('external')}>External references</button>
          </div>
        ) : <strong>{hasReviewControls ? 'Changes' : 'External references'}</strong>}
        <button className="map-bottom-dock-collapse" type="button" aria-expanded={!dockCollapsed} aria-controls="map-bottom-dock-panel" onClick={() => setDockCollapsed((collapsed) => !collapsed)}>{dockCollapsed ? 'Expand' : 'Collapse'}</button>
      </div>
      {!dockCollapsed && (
        <div className="map-bottom-dock-body" id="map-bottom-dock-panel" role={hasReviewControls && hasExternalReferences ? 'tabpanel' : 'region'} aria-label={visibleDockPane === 'changes' ? 'Changes' : 'External references'}>
          {visibleDockPane === 'changes' ? reviewControls : externalReferences}
        </div>
      )}
    </div>
  ) : null

  if (components.length === 0) {
    return (
      <div className="map-empty">
        {reviewSide === 'before' ? 'Before changes has no components.' : emptyMessage ?? 'The architecture has no components yet.'}
        {bottomDock}
      </div>
    )
  }

  return (
    <section className={`map-surface ${bottomDock ? 'has-bottom-dock' : ''}`.trim()} aria-label={reviewSide ? `${reviewSide === 'with' ? 'With changes' : 'Before changes'} architecture map` : 'Architecture map'}>
      {renderFailed ? (
        <div className="map-failure" role="alert">
          <strong>The architecture map could not be shown.</strong>
          <span>{reviewSide ? 'You can still inspect the complete change and update the architecture.' : 'Use the component list to keep working.'}</span>
        </div>
      ) : <div ref={container} className="map-canvas" data-testid="architecture-map" />}
      {!renderFailed && <button className="map-fit" type="button" onClick={() => graph.current?.fit(undefined, 34)}>Fit map</button>}
      {bottomDock}
    </section>
  )
}

function ReviewChangeControls({
  side,
  components,
  componentChanges,
  relationshipChanges,
  reviewDiagramID,
  onSelectComponent,
  onSelectRelationship,
}: {
  side: 'with' | 'before'
  components: MapComponent[]
  componentChanges: ReviewMapComponentChange[]
  relationshipChanges: ReviewMapRelationshipChange[]
  reviewDiagramID?: string
  onSelectComponent: (id: string) => void
  onSelectRelationship?: (relationship: ReviewRelationshipSelection) => void
}) {
  const titles = new Map(components.map((component) => [component.id, component.title]))
  const visibleComponentChanges = componentChanges.filter((change) => titles.has(change.component_id))
  const visibleRelationshipChanges = relationshipChanges.filter((change) => reviewRelationshipVisible(change, side, reviewDiagramID))
  const facts = new Map<string, number>()
  for (const relationship of visibleRelationshipChanges) {
    const fact = `${relationship.status}\u0000${relationship.source_id}\u0000${relationship.target_id}\u0000${relationship.label}`
    facts.set(fact, (facts.get(fact) ?? 0) + 1)
  }
  if (!visibleComponentChanges.length && !visibleRelationshipChanges.length) {
    return <p className="map-review-empty">No visual changes in this {reviewDiagramID ? 'diagram' : 'view'}.</p>
  }
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
          const sourceTitle = change.source_title ?? titles.get(change.source_id) ?? 'Component'
          const targetTitle = change.target_title ?? titles.get(change.target_id) ?? 'Component'
          const fact = `${change.status}\u0000${change.source_id}\u0000${change.target_id}\u0000${change.label}`
          const duplicateContext = (facts.get(fact) ?? 0) > 1 ? `, occurrence ${change.occurrence}` : ''
          const diagramProjection = reviewDiagramID
            ? change.diagram_projections?.find((projection) => projection.side === side && projection.diagram_id === reviewDiagramID)
              ?? (side === 'with' && change.status === 'removed'
                ? change.diagram_projections?.find((projection) => projection.side === 'before' && projection.diagram_id === reviewDiagramID)
                : undefined)
            : undefined
          const selection = {
            ...change,
            ...(diagramProjection ? { key: diagramProjection.key, review_side: diagramProjection.side } : side === 'before' && change.before_key ? { key: change.before_key } : {}),
            source_title: sourceTitle,
            target_title: targetTitle,
          }
          return <li key={change.key}><button type="button" onClick={() => onSelectRelationship?.(selection)}>{relationshipStatusLabel(change.status)}{duplicateContext}: {sourceTitle} — {change.label} — {targetTitle}</button></li>
        })}
      </ul>
    </div>
  )
}

function reviewRelationshipVisible(change: ReviewMapRelationshipChange, side: 'with' | 'before', reviewDiagramID?: string) {
  if (!reviewDiagramID) return side === 'with' || change.status === 'removed'
  if (change.diagram_projections?.some((projection) => projection.side === side && projection.diagram_id === reviewDiagramID)) return true
  return side === 'with' && change.status === 'removed' && change.diagram_projections?.some((projection) => projection.side === 'before' && projection.diagram_id === reviewDiagramID)
}

export function projectionElements(components: MapComponent[], options: ProjectionOptions = {}): ElementDefinition[] {
  const positions = deterministicPositions(options.layoutComponentIDs ?? components.map((component) => component.id))
  const componentStatus = new Map(options.reviewComponents?.map((change) => [change.component_id, change.status]))
  const relationshipStatus = new Map<string, { change: ReviewMapRelationshipChange; projection?: ReviewDiagramRelationshipProjection }>()
  for (const change of options.reviewRelationships ?? []) {
    if (options.reviewDiagramID && options.reviewSide) {
      const projection = change.diagram_projections?.find((candidate) => candidate.side === options.reviewSide && candidate.diagram_id === options.reviewDiagramID)
      if (projection) relationshipStatus.set(projection.key, { change, projection })
      continue
    }
    if (change.status === 'added') relationshipStatus.set(change.key, { change })
    if (change.status === 'removed' && change.before_key) relationshipStatus.set(change.before_key, { change })
  }
  const titleByID = new Map(components.map((component) => [component.id, component.title]))
  const nodes: ElementDefinition[] = components.map((component) => {
    const status = componentStatus.get(component.id) ?? (options.reviewSide ? 'unchanged' : '')
    return {
      data: {
        id: component.id,
        label: component.title,
        displayLabel: component.title,
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
      const relationshipChange = relationshipStatus.get(key)
      const change = relationshipChange?.change
      const reviewProjection = relationshipChange?.projection
      const status = change?.status ?? (options.reviewSide ? 'unchanged' : '')
      edges.push({
        data: {
          id: key,
          key,
          source: reviewProjection?.source_node_key ?? source.id,
          target: reviewProjection?.target_node_key ?? relationship.target_id,
          source_id: change?.source_id ?? source.component_id ?? source.id,
          target_id: change?.target_id ?? components.find((component) => component.id === relationship.target_id)?.component_id ?? relationship.target_id,
          source_title: change?.source_title ?? source.title,
          target_title: change?.target_title ?? titleByID.get(relationship.target_id) ?? 'Component',
          label: relationship.label,
          displayLabel: status === 'added' ? `Added — ${relationship.label}` : status === 'removed' ? `Removed — ${relationship.label}` : relationship.label,
          distance: count === 1 ? 0 : (index - (count - 1) / 2) * 52,
          reviewStatus: status,
          status,
          path: change?.path,
          occurrence: change?.occurrence,
          before_key: change?.before_key,
          review_side: options.reviewSide,
        },
      })
    }
  }
  if (options.reviewSide === 'with' && !options.reviewDiagramID) {
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
