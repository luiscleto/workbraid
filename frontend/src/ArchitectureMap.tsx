import cytoscape, { Core, ElementDefinition } from 'cytoscape'
import { type ReactNode, useEffect, useMemo, useRef, useState } from 'react'

export type MapRelationship = {
  routing?: RouteProjection
  target_id: string
  label: string
  projection_key?: string
}

export type RouteProjection = {
  diagram_id: string; source_id: string; target_id: string; label: string; occurrence: number
  count: number; route: {bend:number}|null; display_bend: number; eligible: boolean; reason?: string
}

export type MapComponent = {
	size?: {width:number;height:number} | null
	position?: {x:number;y:number} | null
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
  onRoute?: (route:RouteProjection,bend:number)=>Promise<boolean>
	onResize?: (id:string,size:{width:number;height:number})=>Promise<boolean>
	viewKey?: string
	onPlace?: (id:string,position:{x:number;y:number})=>Promise<boolean>
  revision: string
  components: MapComponent[]
  reviewOtherComponents?: MapComponent[]
  selectedID?: string
  onSelect: (id: string) => void
  emptyMessage?: string
  fitPadding?: number
  layoutComponentIDs?: string[]
  reviewSide?: 'with' | 'before'
  reviewComponents?: ReviewMapComponentChange[]
  reviewPositionIDs?: string[]
  reviewSizeIDs?: string[]
  reviewRelationships?: ReviewMapRelationshipChange[]
  reviewComposition?: ReactNode
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

type ProjectionOptions = Pick<ArchitectureMapProps, 'layoutComponentIDs' | 'reviewSide' | 'reviewComponents' | 'reviewPositionIDs' | 'reviewSizeIDs' | 'reviewRelationships' | 'reviewDiagramID' | 'annotationNodes' | 'annotationRelationships' | 'annotationAddNodeID' | 'annotationAddRelationshipKey'>

export function ArchitectureMap({
  onRoute,
	onResize,
	viewKey,
	onPlace,
  revision,
  components,
  reviewOtherComponents,
  selectedID,
  onSelect,
  emptyMessage,
  fitPadding = 72,
  layoutComponentIDs,
  reviewSide,
  reviewComponents = [],
  reviewPositionIDs = [],
  reviewSizeIDs = [],
  reviewRelationships = [],
  reviewComposition,
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
  const reviewBounds = useRef<cytoscape.BoundingBox12 | undefined>(undefined)
  const routeHandle=useRef<HTMLButtonElement>(null)
  const routeGuide=useRef<SVGPathElement>(null)
  const routeGesture=useRef<{edge:cytoscape.EdgeSingular;route:RouteProjection;start:number;x:number;y:number;zoom:number;normal:{x:number;y:number};submit:NonNullable<typeof onRoute>}|null>(null)
  const routeCancel=useRef<()=>void>(()=>undefined)
  routeCancel.current=()=>{const g=routeGesture.current;routeGesture.current=null;if(g&&!g.edge.cy().destroyed())g.edge.data('distance',g.start);syncOverlays.current()}
  const resizeHandle = useRef<HTMLButtonElement>(null)
  const resizeGesture = useRef<{node:cytoscape.NodeSingular; start:{width:number;height:number}; x:number;y:number;zoom:number;submit:NonNullable<typeof onResize>}|null>(null)
  const resizeCancel = useRef<()=>void>(()=>undefined)
  resizeCancel.current = () => {
    const gesture=resizeGesture.current;resizeGesture.current=null
    if(gesture&&!gesture.node.cy().destroyed())applyDisplaySize(gesture.node,gesture.start)
    syncOverlays.current()
  }
	const placementHandler=useRef(onPlace)
	placementHandler.current=onPlace
	const viewport=useRef<{key:string|undefined;zoom:number;pan:{x:number;y:number}}|null>(null)
	const placementPending=useRef(false)
  const syncOverlays = useRef<() => void>(() => undefined)
  const selectHandler = useRef(onSelect)
  const relationshipHandler = useRef(onSelectRelationship)
  const nodeAnnotationHandler = useRef(onSelectNodeAnnotation)
  const relationshipAnnotationHandler = useRef(onSelectRelationshipAnnotation)
  const [renderFailed, setRenderFailed] = useState(false)
  const [routeFallbacks, setRouteFallbacks] = useState<string[]>([])
  const layoutKey = [...(layoutComponentIDs ?? components.map((component) => component.component_id ?? component.id))].sort().join('\u0000')
  const annotationKey = JSON.stringify([annotationNodes, annotationRelationships, annotationAddNodeID, annotationAddRelationshipKey])
  // A revision-pinned projection intentionally ignores response-object churn
  // caused by pending edits at the same accepted revision. A review revision is
  // the bound candidate tree or base commit and carries one stable layout basis.
  const elements = useMemo(() => projectionElements(components, {
    layoutComponentIDs,
    reviewSide,
    reviewComponents,
    reviewPositionIDs,
    reviewSizeIDs,
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
    let otherInstance: Core | null = null
    let otherContainer: HTMLDivElement | undefined
    try {
      instance = cytoscape({
        container: container.current,
        elements,
        layout: { name: 'preset', animate: false, fit: true, padding: fitPadding },
        minZoom: 0.0001,
        maxZoom: 2.5,
        style: mapStyles,
      })
      reviewBounds.current = undefined
      if (reviewOtherComponents && reviewSide) {
        otherContainer = document.createElement('div')
        Object.assign(otherContainer.style, {position:'fixed',left:'-10000px',width:`${container.current.clientWidth}px`,height:`${container.current.clientHeight}px`,visibility:'hidden'})
        otherContainer.setAttribute('aria-hidden','true')
        document.body.appendChild(otherContainer)
        otherInstance = cytoscape({container:otherContainer,elements:projectionElements(reviewOtherComponents, {
          layoutComponentIDs,reviewSide:reviewSide==='before'?'with':'before',reviewComponents,reviewPositionIDs,reviewSizeIDs,reviewRelationships,reviewDiagramID,
        }),layout:{name:'preset',fit:false},style:mapStyles})
        updateRouteFallbacks(otherInstance)
        reviewBounds.current = diagramBounds(otherInstance)
        // Measure the active exact side without selection/annotation styling as
        // well. Both review sides then share one immutable renderer frame.
        otherInstance.destroy()
        otherInstance = cytoscape({container:otherContainer,elements,layout:{name:'preset',fit:false},style:mapStyles})
        updateRouteFallbacks(otherInstance)
        const activeBounds=diagramBounds(otherInstance),otherBounds=reviewBounds.current
        reviewBounds.current={x1:Math.min(activeBounds.x1,otherBounds.x1),x2:Math.max(activeBounds.x2,otherBounds.x2),y1:Math.min(activeBounds.y1,otherBounds.y1),y2:Math.max(activeBounds.y2,otherBounds.y2)}
      }
	  if(viewport.current&&viewport.current.key===viewKey){instance.viewport({zoom:viewport.current.zoom,pan:viewport.current.pan})}
	  instance.nodes().ungrabify()
	  if(placementHandler.current&&!placementPending.current)instance.nodes('[!uiAnnotation]').grabify()
	  let grabbed: {id:string;start:{x:number;y:number};submit:NonNullable<typeof onPlace>;cancelled:boolean}|null=null
	  let suppressClickUntil=0
	  const cancel=()=>{if(!grabbed||!instance)return;grabbed.cancelled=true;instance.getElementById(grabbed.id).position(grabbed.start)}
	  const cancelResize=()=>{resizeCancel.current();routeCancel.current()}
	  const escape=(event:KeyboardEvent)=>{if(event.key==='Escape'){cancel();cancelResize()}}
	  window.addEventListener('keydown',escape)
	  window.addEventListener('pointercancel',cancel)
	  window.addEventListener('blur',cancel)
	  window.addEventListener('blur',cancelResize)
	  instance.on('grab','node',(event)=>{
	    if(!placementHandler.current||placementPending.current||event.target.data('uiAnnotation'))return
	    grabbed={id:event.target.id(),start:{...event.target.position()},submit:placementHandler.current,cancelled:false}
	    event.target.addClass('placement-grabbed')
	  })
	  instance.on('free','node',async(event)=>{
	    const gesture=grabbed;grabbed=null;event.target.removeClass('placement-grabbed')
	    if(!gesture||!instance)return
	    const point=event.target.position()
	    if(gesture.cancelled){event.target.position(gesture.start);suppressClickUntil=Date.now()+250;return}
	    const distance=Math.hypot(point.x-gesture.start.x,point.y-gesture.start.y)*instance.zoom()
	    if(distance<4){event.target.position(gesture.start);return}
	    suppressClickUntil=Date.now()+250
	    placementPending.current=true;instance.nodes().ungrabify()
	    try {
	      const kept=await gesture.submit(event.target.data('componentID') ?? gesture.id,{x:roundPosition(point.x),y:roundPosition(point.y)})
	      if(!kept&&!instance.destroyed())event.target.position(gesture.start)
	    } finally {
	      placementPending.current=false
	      if(graph.current&&!graph.current.destroyed()&&placementHandler.current)graph.current.nodes('[!uiAnnotation]').grabify()
	    }
	  })
      instance.on('tap', 'node', (event) => {
	    if(Date.now()<suppressClickUntil)return
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
        if (relationshipHandler.current) {
          relationshipHandler.current({
            key: data.key,
            before_key: data.before_key,
            source_id: data.source_id,
            target_id: data.target_id,
            label: data.label,
            status: data.status || 'unchanged',
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
          const zoom=node.cy().zoom()
          caption.style.top = `${position.y + (Number(node.data('height')) / 2 + 6)*zoom}px`
          caption.style.width = `${Number(node.data('width'))}px`
          caption.style.transform = `translateX(-50%) scale(${zoom})`
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
        if(instance){
          const next=updateRouteFallbacks(instance)
          setRouteFallbacks(previous=>JSON.stringify(previous)===JSON.stringify(next)?previous:next)
        }
        updateBoundaryCaptions()
        updateAnnotationCards()
        const rh=routeHandle.current,guide=routeGuide.current
        if(rh&&instance){
          const edge=instance.edges(':selected').first() as cytoscape.EdgeSingular
          const geometry=edge.empty()?null:routeGeometry(edge)
          rh.hidden=!geometry||!edge.data('routing')?.eligible
          if(guide)guide.style.display=rh.hidden?'none':''
          if(geometry&&!rh.hidden){
            const z=instance.zoom(),p=instance.pan(),point=geometry.control
            rh.style.left=`${point.x*z+p.x}px`;rh.style.top=`${point.y*z+p.y}px`
            if(rh.parentElement)rh.parentElement.style.height=`${instance.height()}px`
            guide?.setAttribute('d',`M ${geometry.start.x*z+p.x} ${geometry.start.y*z+p.y} L ${point.x*z+p.x} ${point.y*z+p.y} L ${geometry.end.x*z+p.x} ${geometry.end.y*z+p.y}`)
          }
        }
        const handle=resizeHandle.current
        if(handle&&instance){
          if(handle.parentElement)handle.parentElement.style.height=`${instance.height()}px`
          const node=instance.nodes(':selected').filter('[!uiAnnotation]').first() as cytoscape.NodeSingular
          handle.hidden=node.empty()
          if(!node.empty()){
            const p=node.renderedPosition(),z=instance.zoom()
            handle.style.left=`${p.x+Number(node.data('width'))*z/2}px`
            handle.style.top=`${p.y+Number(node.data('height'))*z/2}px`
          }
        }
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
      if(!viewport.current||viewport.current.key!==viewKey)fitDiagram(instance,fitPadding,reviewBounds.current)
      return () => {
	    cancel()
	    cancelResize()
	    window.removeEventListener('keydown',escape)
	    window.removeEventListener('pointercancel',cancel)
	    window.removeEventListener('blur',cancel)
	    window.removeEventListener('blur',cancelResize)
	    if(instance)viewport.current={key:viewKey,zoom:instance.zoom(),pan:{...instance.pan()}}
        cancelAnimationFrame(animationFrame)
        resizeObserver?.disconnect()
        instance?.off('pan zoom resize render position', updateOverlays)
        syncOverlays.current = () => undefined
        graph.current = null
        instance?.destroy()
        otherInstance?.destroy()
        otherContainer?.remove()
      }
    } catch {
      graph.current = null
      setRenderFailed(true)
    }
    return () => {
      syncOverlays.current = () => undefined
      graph.current = null
      instance?.destroy()
      otherInstance?.destroy()
      otherContainer?.remove()
    }
  }, [elements, fitPadding,viewKey])

  useEffect(()=>{
    const g=routeGesture.current
    if(g&&(!onRoute||g.edge.id()!==selectedRelationshipKey))routeCancel.current()
    syncOverlays.current()
  },[selectedRelationshipKey,Boolean(onRoute)])

  useEffect(()=>{
    const gesture=resizeGesture.current
    if(gesture&&(!onResize||gesture.node.id()!==selectedID))resizeCancel.current()
    syncOverlays.current()
  },[selectedID,Boolean(onResize)])

  useEffect(()=>{
	const instance=graph.current;if(!instance)return
	instance.nodes().ungrabify()
	if(onPlace&&!placementPending.current)instance.nodes('[!uiAnnotation]').grabify()
  },[Boolean(onPlace),elements])

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
    syncOverlays.current()
  }, [selectedID, selectedRelationshipKey, elements])

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
      compositionChanges={reviewComposition}
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
      {routeFallbacks.length>0&&<p className="map-route-fallback" role="status">Canvas bend dragging unavailable for {routeFallbacks.join('; ')}. The nodes share a center or their shape intersections are unavailable. Stored bends are retained; using default rendering where possible. A curve may be unavailable.</p>}
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
      {!renderFailed && onRoute && selectedRelationshipKey && <div className="map-resize-layer">
        <svg className="route-guide" aria-hidden="true"><path ref={routeGuide}/></svg>
        <button ref={routeHandle} type="button" className="map-route-handle" aria-label="Bend selected link" title="Drag the control point; the guide shows how it bends the link"
          onPointerDown={event=>{event.preventDefault();event.stopPropagation();const edge=graph.current?.getElementById(selectedRelationshipKey) as cytoscape.EdgeSingular|undefined;const geometry=edge&&!edge.empty()?routeGeometry(edge):null;if(!edge||!geometry||!edge.data('routing')?.eligible)return;event.currentTarget.setPointerCapture(event.pointerId);routeGesture.current={edge,route:edge.data('routing'),start:Number(edge.data('distance')),x:event.clientX,y:event.clientY,zoom:edge.cy().zoom(),normal:geometry.normal,submit:onRoute}}}
          onPointerMove={event=>{const g=routeGesture.current;if(!g)return;event.preventDefault();g.edge.data('distance',g.start+((event.clientX-g.x)*g.normal.x+(event.clientY-g.y)*g.normal.y)/g.zoom);syncOverlays.current()}}
          onPointerCancel={()=>routeCancel.current()} onLostPointerCapture={()=>routeCancel.current()}
          onPointerUp={async event=>{const g=routeGesture.current;routeGesture.current=null;if(!g)return;const bend=roundPosition(Number(g.edge.data('distance')));if(event.clientX===g.x&&event.clientY===g.y||bend===g.start){g.edge.data('distance',g.start);return}const kept=await g.submit(g.route,bend);if(!kept&&!g.edge.cy().destroyed())g.edge.data('distance',g.start)}}
        />
      </div>}
      {!renderFailed && onResize && selectedID && <div className="map-resize-layer"><button ref={resizeHandle} className="map-resize-handle" type="button" aria-label="Resize selected node" title="Drag to resize; use Width and Height for precise sizing"
        onPointerDown={event=>{
          event.preventDefault();event.stopPropagation()
          const node=graph.current?.nodes(':selected').filter('[!uiAnnotation]').first() as cytoscape.NodeSingular|undefined
          if(!node||node.empty()||placementPending.current)return
          event.currentTarget.setPointerCapture(event.pointerId)
          resizeGesture.current={node,start:{width:Number(node.data('width')),height:Number(node.data('height'))},x:event.clientX,y:event.clientY,zoom:node.cy().zoom(),submit:onResize}
        }}
        onPointerMove={event=>{
          const g=resizeGesture.current;if(!g)return
          event.preventDefault();event.stopPropagation()
          const size={width:g.start.width+2*(event.clientX-g.x)/g.zoom,height:g.start.height+2*(event.clientY-g.y)/g.zoom}
          // Keep invalid preview dimensions off the renderer; release still rejects them.
          if(size.width>0&&size.height>0)applyDisplaySize(g.node,size)
          syncOverlays.current()
        }}
        onPointerCancel={()=>resizeCancel.current()}
        onLostPointerCapture={()=>resizeCancel.current()}
        onPointerUp={async event=>{
          event.preventDefault();event.stopPropagation()
          const g=resizeGesture.current;resizeGesture.current=null;if(!g)return
          const size={width:roundPosition(g.start.width+2*(event.clientX-g.x)/g.zoom),height:roundPosition(g.start.height+2*(event.clientY-g.y)/g.zoom)}
          applyDisplaySize(g.node,g.start)
          if(size.width<80||size.width>1600||size.height<48||size.height>1200){syncOverlays.current();return}
          if(size.width===g.start.width&&size.height===g.start.height){syncOverlays.current();return}
          applyDisplaySize(g.node,size)
          placementPending.current=true;g.node.cy().nodes().ungrabify()
          try {if(!await g.submit(g.node.data('componentID'),size)&&!g.node.cy().destroyed())applyDisplaySize(g.node,g.start)}
          finally{placementPending.current=false;if(graph.current&&placementHandler.current)graph.current.nodes('[!uiAnnotation]').grabify();syncOverlays.current()}
        }}
        onClick={()=>{
          const width=document.querySelector<HTMLInputElement>('[aria-label="Node width"]')
          const disclosure=width?.closest('details')
          if(disclosure)disclosure.open=true
          width?.focus()
        }}>↘</button></div>}
      {!renderFailed && <button className="map-fit" type="button" onClick={() => {
        if(graph.current)fitDiagram(graph.current,fitPadding,reviewBounds.current)
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
  compositionChanges,
  reviewDiagramID,
  onSelectComponent,
  onSelectRelationship,
}: {
  side: 'with' | 'before'
  components: MapComponent[]
  componentChanges: ReviewMapComponentChange[]
  relationshipChanges: ReviewMapRelationshipChange[]
  compositionChanges?: ReactNode
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
  if (!visibleComponentChanges.length && !visibleRelationshipChanges.length && !compositionChanges) {
    return <p className="map-review-empty">No visual changes in this {reviewDiagramID ? 'diagram' : 'view'}.</p>
  }
  return (
    <div className="map-review-controls" aria-label="Visual changes">
      {(visibleComponentChanges.length > 0 || visibleRelationshipChanges.length > 0) && <p className="map-review-key"><span>＋ Added</span><span>△ Content changed</span><span>− Removed relationship</span></p>}
      <ul>
        {compositionChanges}
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
  const positions = displayPositions(components,options.layoutComponentIDs)
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
    const size=component.size??(component.node_kind==='boundary'?{width:104,height:62}:{width:116,height:54})
    const status = componentStatus.get(component.id) ?? (options.reviewSide ? 'unchanged' : '')
    const annotationCount = options.annotationNodes?.[component.id] ?? 0
    return {
      data: {
        id: component.id,
        componentID: component.component_id ?? component.id,
        label: component.title,
        displayLabel: fittedTitle(component.title,size,component.node_kind==='boundary'),
        width:size.width,
        height:size.height,
        nodeKind: component.node_kind ?? '',
        boundaryHomeTitle: component.boundary_home_title,
        reviewStatus: status,
        positionChanged: options.reviewPositionIDs?.includes(component.component_id ?? component.id) ? 'yes' : '',
        sizeChanged: options.reviewSizeIDs?.includes(component.component_id ?? component.id) ? 'yes' : '',
        annotationCount,
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
          routing: relationship.routing,
          distance: relationship.routing?.display_bend ?? (count === 1 ? 0 : (index - (count - 1) / 2) * 52),
          defaultDistance: count === 1 ? 0 : (index - (count - 1) / 2) * 52,
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

export function roundPosition(value: number): number {
  return Math.sign(value) * Math.floor(Math.abs(value) + 0.5)
}

// These are the renderer's actual shape intersections, shared with its curve
// calculation. Reading them does not create a second shape interpretation.
export function routeGeometry(edge:cytoscape.EdgeSingular){
 const a=edge.source().position(),b=edge.target().position(),length=Math.hypot(b.x-a.x,b.y-a.y)
 if(!length)return null
 const control=edge.controlPoints()?.[0]
 const rs=(edge as unknown as {_private:{rscratch:{srcIntn?:number[];tgtIntn?:number[]}}})._private.rscratch
 if(!control||!rs.srcIntn||!rs.tgtIntn||![...rs.srcIntn,...rs.tgtIntn,control.x,control.y].every(Number.isFinite))return null
 return {control,start:{x:rs.srcIntn[0],y:rs.srcIntn[1]},end:{x:rs.tgtIntn[0],y:rs.tgtIntn[1]},normal:{x:-(b.y-a.y)/length,y:(b.x-a.x)/length}}
}

// Browser presentation only: neither eligibility nor the saved scalar changes.
// Reuse renderer intersections and allow its existing fallback to remain absent
// when it cannot draw finite geometry. Recovery requires no backend mutation.
function updateRouteFallbacks(instance:Core){
 const notices:string[]=[]
 instance.edges().forEach(edge=>{
  if(!edge.data('routing')||edge.source().id()===edge.target().id())return
  const unavailable=!routeGeometry(edge)
  const fallback=edge.scratch('routeFallback') as {distance:number}|undefined
  if(unavailable){
   notices.push(`${edge.data('label')} (occurrence ${edge.data('routing').occurrence})`)
   if(!fallback){edge.scratch('routeFallback',{distance:Number(edge.data('routing').route?.bend??edge.data('distance'))});edge.data('distance',Number(edge.data('defaultDistance')))}
  }else if(fallback){edge.removeScratch('routeFallback');edge.data('distance',fallback.distance)}
 })
 return notices
}

export function displayPositions(components: MapComponent[], layoutIDs?: string[]): Record<string, { x: number; y: number }> {
  const seeds = deterministicPositions(layoutIDs ?? components.map(c => c.component_id ?? c.id))
  const result: Record<string, { x: number; y: number }> = {}
  for (const c of components) {
    result[c.id] = { ...(c.position ?? seeds[c.component_id ?? c.id] ?? {x:0,y:0}) }
  }
  return result
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


// Renderer text is disposable. Exact title source remains in label and the pane.
let titleMeasure: CanvasRenderingContext2D | null | undefined
export function fittedTitle(title:string,size:{width:number;height:number},boundary:boolean):string {
 const width=Math.max(1,(boundary?size.width/2:size.width)-24)
 const height=Math.max(1,(boundary?size.height/2:size.height)-24)
 const lines=Math.max(1,Math.floor(height/18))
 if(titleMeasure===undefined){
  try{titleMeasure=document.createElement('canvas').getContext('2d')}catch{titleMeasure=null}
 }
 if(titleMeasure)titleMeasure.font='14px "IBM Plex Sans"'
 const measure=(s:string)=>titleMeasure?.measureText(s).width??Array.from(s).length*8
 const chars=Array.from(title.replace(/\s+/g,' '))
 const result:string[]=[]
 let rest=chars.join('')
 while(rest&&result.length<lines){
  let n=0
  while(n<rest.length&&measure(rest.slice(0,n+1))<=width)n++
  n=Math.max(1,n)
  if(n<rest.length&&result.length<lines-1){const space=rest.lastIndexOf(' ',n);if(space>0)n=space}
  let line=rest.slice(0,n).trimEnd();rest=rest.slice(n).trimStart()
  if(result.length===lines-1&&rest){while(line&&measure(line+'…')>width)line=Array.from(line).slice(0,-1).join('');line+='…'}
  result.push(line)
 }
 return result.join('\n')
}
function applyDisplaySize(node:cytoscape.NodeSingular,size:{width:number;height:number}){
 node.data({...size,displayLabel:fittedTitle(String(node.data('label')),size,node.data('nodeKind')==='boundary')})
}
function diagramBounds(instance:Core){
 const box=instance.elements().boundingBox()
 instance.nodes('[nodeKind = "boundary"]').forEach(node=>{
  const p=node.position(),w=Number(node.data('width')),h=Number(node.data('height'))
  box.x1=Math.min(box.x1,p.x-w/2);box.x2=Math.max(box.x2,p.x+w/2);box.y2=Math.max(box.y2,p.y+h/2+24)
 })
 box.w=box.x2-box.x1;box.h=box.y2-box.y1
 return box
}
function fitDiagram(instance:Core,padding:number,other?:cytoscape.BoundingBox12){
 const box=other?{...other,w:other.x2-other.x1,h:other.y2-other.y1}:diagramBounds(instance)
 const zoom=Math.max(instance.minZoom(),Math.min(instance.maxZoom(),(instance.width()-2*padding)/Math.max(1,box.w),(instance.height()-2*padding)/Math.max(1,box.h)))
 instance.viewport({zoom,pan:{x:instance.width()/2-zoom*(box.x1+box.x2)/2,y:instance.height()/2-zoom*(box.y1+box.y2)/2}})
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
      'font-size': 14,
      'text-wrap': 'wrap',
      'text-max-width': '2000px',
      'text-valign': 'center',
      'text-halign': 'center',
      width: 'data(width)',
      height: 'data(height)',
      shape: 'round-rectangle',
    },
  },
  { selector: 'node[reviewStatus = "unchanged"]', style: { opacity: 0.48, 'border-style': 'dotted' } },
  { selector: 'node[reviewStatus = "added"]', style: { 'background-color': '#d8eadf', 'border-color': '#126747', 'border-width': 3, shape: 'hexagon' } },
  { selector: 'node[reviewStatus = "content_changed"]', style: { 'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-width': 3, 'border-style': 'dashed' } },
  { selector: 'node[nodeKind = "reference"]', style: { 'border-style': 'dashed', 'background-color': '#eee3c8' } },
  { selector: 'node[positionChanged = "yes"]', style: { 'border-color': '#315e46', 'border-width': 3, opacity: 1, 'text-opacity': 1 } },
  { selector: 'node[sizeChanged = "yes"]', style: { 'border-color': '#315e46', 'border-width': 3, opacity: 1, 'text-opacity': 1 } },
  { selector: 'node[nodeKind = "boundary"]', style: { shape: 'diamond', 'border-style': 'dotted', 'background-color': '#efe7d3' } },
  { selector: 'node.placement-grabbed', style: { 'border-color':'#27251f','overlay-opacity':0.08 } },
  { selector: 'node:selected', style: { 'background-color': '#e7dba9', 'border-color': '#18734f', 'overlay-opacity':0.08, opacity: 1 } },
  { selector: 'node[reviewStatus = "unchanged"]:selected', style: { 'background-color': '#f8f0dc', 'border-color': '#27251f', 'border-style': 'dotted', opacity: 1 } },
  { selector: 'node[reviewStatus = "added"]:selected', style: { 'background-color': '#d8eadf', 'border-color': '#126747', shape: 'hexagon', opacity: 1 } },
  { selector: 'node[reviewStatus = "content_changed"]:selected', style: { 'background-color': '#f1dfad', 'border-color': '#8c5c12', 'border-style': 'dashed', opacity: 1 } },
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
      'edge-distances': 'intersection',
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
