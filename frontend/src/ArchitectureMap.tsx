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
  boundary_home_title?: string
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

export type ReviewRelationshipSelection = Omit<ReviewMapRelationshipChange, 'status'> & {
  status: 'added' | 'removed' | 'unchanged'
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
  fitPadding?: number
  layoutComponentIDs?: string[]
  reviewSide?: 'with' | 'before'
  reviewComponents?: ReviewMapComponentChange[]
  reviewRelationships?: ReviewMapRelationshipChange[]
  reviewDiagramID?: string
  selectedRelationshipKey?: string
  onSelectRelationship?: (relationship: ReviewRelationshipSelection) => void
  externalReferences?: ReactNode
  annotationOverlay?: ReactNode
  annotationNodes?: Record<string, number>
  annotationRelationships?: Record<string, number>
  annotationAddNodeID?: string
  annotationAddRelationshipKey?: string
  onSelectNodeAnnotation?: (id: string) => void
  onSelectRelationshipAnnotation?: (key: string) => void
}

type ProjectionOptions = Pick<ArchitectureMapProps, 'layoutComponentIDs' | 'reviewSide' | 'reviewComponents' | 'reviewRelationships' | 'reviewDiagramID' | 'annotationNodes' | 'annotationRelationships' | 'annotationAddNodeID' | 'annotationAddRelationshipKey'>

export function ArchitectureMap({
  revision,
  components,
  selectedID,
  onSelect,
  emptyMessage,
  fitPadding = 72,
  layoutComponentIDs,
  reviewSide,
  reviewComponents = [],
  reviewRelationships = [],
  reviewDiagramID,
  selectedRelationshipKey,
  onSelectRelationship,
  externalReferences,
  annotationOverlay,
  annotationNodes = {},
  annotationRelationships = {},
  annotationAddNodeID,
  annotationAddRelationshipKey,
  onSelectNodeAnnotation,
  onSelectRelationshipAnnotation,
}: ArchitectureMapProps) {
  const container = useRef<HTMLDivElement>(null)
  const boundaryCaptionLayer = useRef<HTMLDivElement>(null)
  const annotationLayer = useRef<HTMLDivElement>(null)
  const graph = useRef<Core | null>(null)
  const syncOverlays = useRef<() => void>(() => undefined)
  const selectHandler = useRef(onSelect)
  const relationshipHandler = useRef(onSelectRelationship)
  const nodeAnnotationHandler = useRef(onSelectNodeAnnotation)
  const relationshipAnnotationHandler = useRef(onSelectRelationshipAnnotation)
  const [renderFailed, setRenderFailed] = useState(false)
  const layoutKey = [...(layoutComponentIDs ?? components.map((component) => component.component_id ?? component.id))].sort().join('\u0000')
  const annotationKey = JSON.stringify([annotationNodes, annotationRelationships, annotationAddNodeID, annotationAddRelationshipKey])
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
    nodeAnnotationHandler.current = onSelectNodeAnnotation
    relationshipAnnotationHandler.current = onSelectRelationshipAnnotation
  }, [onSelect, onSelectRelationship, onSelectNodeAnnotation, onSelectRelationshipAnnotation])

  useEffect(() => {
    if (!container.current) return
    setRenderFailed(false)
    let instance: Core | null = null
    try {
      instance = cytoscape({
        container: container.current,
        elements,
        layout: { name: 'preset', animate: false, fit: true, padding: fitPadding },
        minZoom: 0.35,
        maxZoom: 2.5,
        style: mapStyles,
      })
      instance.on('tap', 'node', (event) => {
        const data = typeof event.target.data === 'function' ? event.target.data() as { annotationCount?: number; annotationNodeID?: string; annotationRelationshipKey?: string } : {}
        if (data.annotationNodeID) {
          nodeAnnotationHandler.current?.(data.annotationNodeID)
          return
        }
        if (data.annotationRelationshipKey) {
          relationshipAnnotationHandler.current?.(data.annotationRelationshipKey)
          return
        }
        selectHandler.current(event.target.id())
      })
      instance.on('tap', 'edge', (event) => {
        const data = event.target.data() as ReviewRelationshipSelection & { reviewStatus?: string; annotationCount?: number }
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
      const updateBoundaryCaptions = () => {
        const layer = boundaryCaptionLayer.current
        if (!layer || !instance) return
        const captions = new Map([...layer.querySelectorAll<HTMLElement>('[data-boundary-node-id]')].map((caption) => [caption.dataset.boundaryNodeId, caption]))
        instance.nodes('[nodeKind = "boundary"]').forEach((node) => {
          const caption = captions.get(node.id())
          if (!caption) return
          const position = node.renderedPosition()
          caption.style.left = `${position.x}px`
          caption.style.top = `${position.y + node.renderedHeight() / 2 + 5}px`
        })
      }
      const updateAnnotationCards = () => {
        const layer = annotationLayer.current
        const canvas = container.current
        if (!layer || !canvas || !instance) return
        const currentInstance = instance
        const layerRect = layer.getBoundingClientRect()
        const canvasRect = canvas.getBoundingClientRect()
        const cards = [...layer.querySelectorAll<HTMLElement>('[data-map-annotation-kind][data-map-annotation-id]')]
        const occupied: { left: number; top: number; right: number; bottom: number }[] = []
        cards.forEach((card, index) => {
          const kind = card.dataset.mapAnnotationKind
          const id = card.dataset.mapAnnotationId
          if (!id) return
          const target = currentInstance.getElementById(id)
          card.hidden = target.empty()
          if (target.empty()) return
          const position = kind === 'relationship'
            ? target.renderedMidpoint()
            : target.renderedPosition()
          const width = Math.min(card.offsetWidth || 250, Math.max(180, canvasRect.width - 24))
          const height = Math.min(card.offsetHeight || 190, Math.max(120, canvasRect.height - 24))
          const anchorX = canvasRect.left - layerRect.left + position.x
          const anchorY = canvasRect.top - layerRect.top + position.y
          const canvasLeft = canvasRect.left - layerRect.left + 10
          const canvasTop = canvasRect.top - layerRect.top + 10
          const canvasRight = canvasRect.right - layerRect.left - 10
          const canvasBottom = canvasRect.bottom - layerRect.top - 10
          let left = anchorX + 42
          if (left + width > canvasRight) left = anchorX - width - 42
          left = Math.max(canvasLeft, Math.min(left, canvasRight - width))
          let top = Math.max(canvasTop, Math.min(anchorY - 24, canvasBottom - height))
          // Prefer space beside the item, then the other side or a free vertical
          // slot. All positions are disposable browser presentation.
          const slots = [
            { left, top },
            { left: Math.max(canvasLeft, Math.min(anchorX - width - 42, canvasRight - width)), top },
            ...occupied.flatMap(prior => [
              { left, top: prior.bottom + 10 },
              { left, top: prior.top - height - 10 },
              { left: prior.right + 10, top: canvasTop },
              { left: prior.left - width - 10, top: canvasTop },
            ]),
          ]
          const free = slots.find(slot => slot.left >= canvasLeft && slot.left + width <= canvasRight &&
            slot.top >= canvasTop && slot.top + height <= canvasBottom &&
            occupied.every(prior => slot.left >= prior.right + 8 || slot.left + width <= prior.left - 8 ||
              slot.top >= prior.bottom + 8 || slot.top + height <= prior.top - 8))
          if (free) { left = free.left; top = free.top }
          else top = Math.max(canvasTop, Math.min(top + index * 24, canvasBottom - height))
          card.style.left = `${left}px`
          card.style.top = `${Math.max(canvasTop, top)}px`
          occupied.push({ left, top, right: left + width, bottom: top + height })
        })
      }
      const updateOverlays = () => {
        updateBoundaryCaptions()
        updateAnnotationCards()
      }
      syncOverlays.current = updateOverlays
      instance.on('pan zoom resize render position', updateOverlays)
      updateOverlays()
      const animationFrame = requestAnimationFrame(updateOverlays)
      const resizeObserver = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(() => {
        instance?.resize()
        updateOverlays()
      })
      resizeObserver?.observe(container.current)
      graph.current = instance
      return () => {
        cancelAnimationFrame(animationFrame)
        resizeObserver?.disconnect()
        instance?.off('pan zoom resize render position', updateOverlays)
        syncOverlays.current = () => undefined
        graph.current = null
        instance?.destroy()
      }
    } catch {
      graph.current = null
      setRenderFailed(true)
    }
    return () => {
      syncOverlays.current = () => undefined
      graph.current = null
      instance?.destroy()
    }
  }, [elements, fitPadding])

  useEffect(() => {
    const instance = graph.current
    if (!instance) return
    // Updating comment badges must not reconstruct or re-fit the map.
    instance.batch(() => {
      instance.nodes('[uiAnnotation]').remove()
      const markers = projectionElements(components, {
        reviewSide, reviewComponents, reviewRelationships, reviewDiagramID,
        annotationNodes, annotationRelationships, annotationAddNodeID, annotationAddRelationshipKey,
      }).filter(element => element.data.uiAnnotation)
      instance.add(markers).ungrabify().unselectify()
    })
    const positionMarkers = () => {
      instance.nodes('[uiAnnotation]').forEach(marker => {
        const data = marker.data()
        const target = instance.getElementById(data.annotationNodeID ?? data.annotationRelationshipKey)
        if (target.empty()) return
        const point = data.annotationNodeID ? target.position() : target.midpoint()
        marker.position({ x: point.x + (data.annotationNodeID ? 58 : 0), y: point.y - (data.annotationNodeID ? 34 : 18) })
      })
    }
    positionMarkers()
    instance.on('position', 'node[!uiAnnotation]', positionMarkers)
    return () => { instance.off('position', 'node[!uiAnnotation]', positionMarkers) }
  }, [elements, annotationKey])

  useEffect(() => {
    const instance = graph.current
    if (!instance) return
    instance.$(':selected').unselect()
    if (selectedRelationshipKey) instance.getElementById(selectedRelationshipKey).select()
    else if (selectedID) instance.getElementById(selectedID).select()
  }, [selectedID, selectedRelationshipKey])

  useEffect(() => {
    const animationFrame = requestAnimationFrame(() => syncOverlays.current())
    return () => cancelAnimationFrame(animationFrame)
  })

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
    : dockPane === 'external' && hasExternalReferences
      ? 'external'
      : hasExternalReferences ? 'external' : 'changes'
  useEffect(() => {
    const animationFrame = requestAnimationFrame(() => {
      graph.current?.resize()
      graph.current?.fit(undefined, fitPadding)
      syncOverlays.current()
    })
    return () => cancelAnimationFrame(animationFrame)
  }, [dockCollapsed, visibleDockPane, hasExternalReferences, hasReviewControls, fitPadding])
  const dockPanes = [
    ...(hasReviewControls ? [{ id: 'changes' as const, label: 'Changes' }] : []),
    ...(hasExternalReferences ? [{ id: 'external' as const, label: 'External references' }] : []),
  ]
  const bottomDock = dockPanes.length ? (
    <div className={`map-bottom-dock ${dockCollapsed ? 'collapsed' : ''}`}>
      <div className="map-bottom-dock-header">
        {dockPanes.length > 1 ? (
          <div className="map-bottom-dock-tabs" role="tablist" aria-label="Map information">
            {dockPanes.map((pane) => <button key={pane.id} type="button" role="tab" aria-selected={visibleDockPane === pane.id} aria-controls="map-bottom-dock-panel" onClick={() => setDockPane(pane.id)}>{pane.label}</button>)}
          </div>
        ) : <strong>{dockPanes[0].label}</strong>}
        <button className="map-bottom-dock-collapse" type="button" aria-expanded={!dockCollapsed} aria-controls="map-bottom-dock-panel" onClick={() => setDockCollapsed((collapsed) => !collapsed)}>{dockCollapsed ? 'Expand' : 'Collapse'}</button>
      </div>
      {!dockCollapsed && (
        <div className="map-bottom-dock-body" id="map-bottom-dock-panel" role={dockPanes.length > 1 ? 'tabpanel' : 'region'} aria-label={dockPanes.find((pane) => pane.id === visibleDockPane)?.label}>
          {visibleDockPane === 'changes' ? reviewControls : externalReferences}
        </div>
      )}
    </div>
  ) : null
  const boundaryCaptions = components.filter((component) => component.node_kind === 'boundary' && component.boundary_home_title)

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
      {!renderFailed && boundaryCaptions.length > 0 && (
        <div ref={boundaryCaptionLayer} className="map-boundary-captions" aria-hidden="true">
          {boundaryCaptions.map((component) => <span className="map-boundary-caption" data-boundary-node-id={component.id} key={component.id}>Lives in {component.boundary_home_title}</span>)}
        </div>
      )}
      {!renderFailed && annotationOverlay && <div ref={annotationLayer} className="map-annotation-layer">{annotationOverlay}</div>}
      {!renderFailed && <button className="map-fit" type="button" onClick={() => {
        graph.current?.fit(undefined, fitPadding)
        syncOverlays.current()
      }}>Fit map</button>}
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
      <p className="map-review-key"><span>＋ Added</span><span>△ Content changed</span><span>− Removed relationship</span></p>
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
  const positions = deterministicPositions(options.layoutComponentIDs ?? components.map((component) => component.component_id ?? component.id))
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
    const annotationCount = options.annotationNodes?.[component.id] ?? 0
    return {
      data: {
        id: component.id,
        label: component.title,
        displayLabel: component.title,
        nodeKind: component.node_kind ?? '',
        boundaryHomeTitle: component.boundary_home_title,
        reviewStatus: status,
        annotationCount,
      },
      position: positions[component.component_id ?? component.id],
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
  const occurrences = new Map<string, number>()
  const edges: ElementDefinition[] = []
  for (const source of components) {
    for (let relationshipIndex = 0; relationshipIndex < (source.relationships ?? []).length; relationshipIndex += 1) {
      const relationship = source.relationships[relationshipIndex]
      const pair = `${source.id}\u0000${relationship.target_id}`
      const index = seen.get(pair) ?? 0
      seen.set(pair, index + 1)
      const count = grouped.get(pair) ?? 1
      const key = relationship.projection_key ?? `projection:${source.id}:${relationship.target_id}:${index}`
      const sourceComponentID = source.component_id ?? source.id
      const targetComponentID = components.find((component) => component.id === relationship.target_id)?.component_id ?? relationship.target_id
      const exactFact = `${sourceComponentID}\u0000${targetComponentID}\u0000${relationship.label}`
      const occurrence = (occurrences.get(exactFact) ?? 0) + 1
      occurrences.set(exactFact, occurrence)
      const relationshipChange = relationshipStatus.get(key)
      const change = relationshipChange?.change
      const reviewProjection = relationshipChange?.projection
      const status = change?.status ?? (options.reviewSide ? 'unchanged' : '')
      const annotationCount = options.annotationRelationships?.[key] ?? 0
      edges.push({
        data: {
          id: key,
          key,
          source: reviewProjection?.source_node_key ?? source.id,
          target: reviewProjection?.target_node_key ?? relationship.target_id,
          source_id: change?.source_id ?? sourceComponentID,
          target_id: change?.target_id ?? targetComponentID,
          source_title: change?.source_title ?? source.title,
          target_title: change?.target_title ?? titleByID.get(relationship.target_id) ?? 'Component',
          label: relationship.label,
          displayLabel: status === 'added' ? `Added — ${relationship.label}` : status === 'removed' ? `Removed — ${relationship.label}` : relationship.label,
          distance: count === 1 ? 0 : (index - (count - 1) / 2) * 52,
          reviewStatus: status,
          status,
          path: change?.path ?? (source.filename ? `components/${source.filename}` : ''),
          occurrence: change?.occurrence ?? occurrence,
          before_key: change?.before_key,
          review_side: options.reviewSide,
          annotationCount,
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
  const positionsByNodeID = new Map(nodes.map((node) => [String(node.data.id), node.position ?? { x: 0, y: 0 }]))
  const annotationMarkers: ElementDefinition[] = []
  for (const node of nodes) {
    const count = Number(node.data.annotationCount ?? 0)
    const add = node.data.id === options.annotationAddNodeID && !count
    if (!count && !add) continue
    const position = node.position ?? { x: 0, y: 0 }
    annotationMarkers.push({
      data: { id: `annotation-node:${node.data.id}`, displayLabel: count ? `✎ ${count}` : '✎ +', uiAnnotation: true, ...(add ? { annotationAdd: true } : {}), annotationNodeID: node.data.id },
      position: { x: position.x + 58, y: position.y - 34 },
    })
  }
  for (const edge of edges) {
    const count = Number(edge.data.annotationCount ?? 0)
    const add = edge.data.key === options.annotationAddRelationshipKey && !count
    if (!count && !add) continue
    const source = positionsByNodeID.get(String(edge.data.source))
    const target = positionsByNodeID.get(String(edge.data.target))
    if (!source || !target) continue
    annotationMarkers.push({
      data: { id: `annotation-relationship:${edge.data.id}`, displayLabel: count ? `✎ ${count}` : '✎ +', uiAnnotation: true, ...(add ? { annotationAdd: true } : {}), annotationRelationshipKey: edge.data.key },
      position: { x: (source.x + target.x) / 2, y: (source.y + target.y) / 2 - 18 },
    })
  }
  return [...nodes, ...edges, ...annotationMarkers]
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
      'text-valign': 'center',
      'text-halign': 'center',
      width: 116,
      height: 54,
      shape: 'round-rectangle',
    },
  },
  { selector: 'node[reviewStatus = "unchanged"]', style: { opacity: 0.48, 'border-style': 'dotted' } },
  { selector: 'node[reviewStatus = "added"]', style: { 'background-color': '#d8eadf', 'border-color': '#126747', 'border-width': 3, shape: 'hexagon' } },
  { selector: 'node[reviewStatus = "content_changed"]', style: { 'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-width': 3, 'border-style': 'dashed' } },
  { selector: 'node[nodeKind = "reference"]', style: { 'border-style': 'dashed', 'background-color': '#eee3c8' } },
  { selector: 'node[nodeKind = "boundary"]', style: { shape: 'diamond', 'border-style': 'dotted', 'background-color': '#efe7d3', width: 104, height: 62 } },
  { selector: 'node:selected', style: { 'background-color': '#e7dba9', 'border-color': '#18734f', 'border-width': 4, opacity: 1 } },
  { selector: 'node[reviewStatus = "unchanged"]:selected', style: { 'background-color': '#f8f0dc', 'border-color': '#27251f', 'border-width': 5, 'border-style': 'dotted', opacity: 1 } },
  { selector: 'node[reviewStatus = "added"]:selected', style: { 'background-color': '#d8eadf', 'border-color': '#126747', 'border-width': 5, shape: 'hexagon', opacity: 1 } },
  { selector: 'node[reviewStatus = "content_changed"]:selected', style: { 'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-width': 5, 'border-style': 'dashed', opacity: 1 } },
  { selector: 'node[uiAnnotation]', style: { width: 32, height: 20, shape: 'round-rectangle', label: 'data(displayLabel)', color: '#68470f', 'background-color': '#f2dea0', 'border-color': '#a77b25', 'border-width': 1, 'font-size': 9, 'font-weight': 600, 'text-valign': 'center', 'text-halign': 'center', opacity: 1, 'z-index': 20 } },
  { selector: 'node[uiAnnotation][annotationAdd]', style: { opacity: 0.58, 'background-color': '#f8f0dc', 'border-style': 'dashed' } },
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
      'text-rotation': 'none',
      'text-wrap': 'wrap',
      'text-max-width': '150px',
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
