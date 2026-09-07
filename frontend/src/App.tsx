import { FormEvent, KeyboardEvent as ReactKeyboardEvent, ReactNode, useCallback, useEffect, useId, useRef, useState } from 'react'
import {
  ArchitectureMap,
  MapComponent,
  RouteProjection,
  ReviewMapComponentChange,
  ReviewMapRelationshipChange,
  ReviewRelationshipSelection,
} from './ArchitectureMap'
import { MarkdownBody } from './MarkdownBody'
import {DiagramNotePane,type DiagramNote} from './DiagramNotePane'
import { RawDiff } from './RawDiff'
import { ReconciliationTask, type ReconciliationPreview, type ReconciliationResolution, type ReconciliationSnapshot } from './ReconciliationTask'

type CatalogProject = { name?: string; slug?: string; revision?: string; store_id?: string; unavailable?: boolean; conflict?: boolean }

type ArchitectureResult = {
  project_slug: string
  store_id: string
  project_name: string
  state: 'empty' | 'ready'
  revision: string
  format_version: number
  component_count: number
  component_titles: string[]
  components: AuthoringComponent[]
  root_diagram_id?: string
  diagrams?: DiagramProjection[]
  home_move_destinations?: { component_id: string; current_home_id: string; diagram_ids: string[] }[]
  reference_choices?: { diagram_id: string; component_id: string; title: string; context?: string; home_diagram: string }[]
  changes?: ChangesInProgress
  change_sets: ChangesInProgress[]
  unavailable_change_sets?: { id?: string; name?: string; lifecycle?: string; reason: string }[]
  review_submissions?: ReviewSubmissionSummary[]
  unavailable_reviews?: { change_set_id?: string; review_id?: string; reason: string }[]
  submitted_review?: ReviewSubmission
  action_change_set_id?: string
  action_review_id?: string
  stale?: boolean
  parent_diff?: string
  action_error?: string
}

type DiagramProjection = {
	shapes?: {diagram_id:string;component_id:string;shape:'rectangle'|'ellipse'|'diamond'|null}[]
	notes?: DiagramNote[]
  id: string
  title: string
  filename: string
  depth: number
  context?: string
  parent_diagram_id?: string
  parent_anchor_component_id?: string
  breadcrumbs: { id: string; title: string; focus_anchor_component_id?: string }[]
  appearances: DiagramAppearance[]
  boundaries: DiagramBoundary[]
  relationships: DiagramRelationship[]
}

type DiagramAppearance = {
 display_size?: {width:number;height:number}|null
 display_position?: {x:number;y:number}|null
 position_source?: 'stored'|'derived'
	position?: {x:number;y:number}|null
  component_id: string
  role: 'home' | 'reference'
  detail_diagram_id?: string
  detail_diagram_title?: string
}

type DiagramBoundary = {
 display_size?: {width:number;height:number}|null
 position?: {x:number;y:number}|null
 display_position?: {x:number;y:number}|null
 position_source?: 'stored'|'derived'
  key: string
  component_id: string
  title: string
  context?: string
  home_diagram_id: string
  home_diagram_title: string
}

type DiagramRelationship = {
  routing?: RouteProjection
  key: string
  source_node_key: string
  target_node_key: string
  source_component_id: string
  target_component_id: string
  label: string
}

type AuthoringComponent = {
  id: string
  title: string
  description: string
  markdown_source?: string
  filename: string
  relationships: { target_id: string; label: string; projection_key?: string }[]
}

type PendingComponent = AuthoringComponent & { new: boolean }

type ChangesInProgress = {
  change_set_state: string
  candidate_tree?: string
  detail_reassignments?: { diagram_id: string; anchor_component_id: string }[]
  id: string
  name: string
  lifecycle: 'active' | 'applied' | 'no_longer_active'
  proposal_markdown: string
  applied_revision?: string
  out_of_date?: boolean
  read_only?: boolean
  base_revision: string
  generation: number
  components: PendingComponent[]
  relationship_targets?: RelationshipTarget[]
  valid: boolean
  validation_code?: string
  validation_item?: string
  validation_relationship_position?: number
  validation_relationship_field?: 'target' | 'label'
  review?: ChangeReview
  review_blocker?: string
  stale?: boolean
  validation_diagram?: string
  validation_diagram_field?: 'title' | 'home' | 'detail'
  detail_diagrams?: { id: string; path: string; title: string; anchor_component_id: string }[]
  diagram_titles?: { diagram_id: string; title: string }[]
  home_moves?: { component_id: string; diagram_id: string }[]
  references?: { diagram_id: string; component_id: string; present: boolean }[]
  diagram_options?: DiagramAuthoringOption[]
  candidate?: ReviewSnapshot
}

type DiagramAuthoringOption = { id: string; title: string; context?: string }

type RelationshipTarget = {
  id: string
  title: string
  context?: string
  new?: boolean
}

function PositionControls({position,busy,onDirty,onKeep}:{position:{x:number;y:number}|null;busy:boolean;onDirty:(v:boolean)=>void;onKeep:(p:{x:number;y:number}|null)=>Promise<boolean>}) {
  const [x,setX]=useState(String(position?.x??0)),[y,setY]=useState(String(position?.y??0))
  useEffect(() => () => onDirty(false), [onDirty])
  return <section aria-label="Node position"><h3>Position</h3>
    <form onSubmit={async e=>{e.preventDefault();const p={x:Number(x),y:Number(y)};if(!Number.isInteger(p.x)||!Number.isInteger(p.y)||Math.abs(p.x)>100000||Math.abs(p.y)>100000)return;if(await onKeep(p))onDirty(false)}}>
      <div className="position-fields"><label>X<input aria-label="Position X" type="number" min={-100000} max={100000} step={1} required value={x} onChange={e=>{setX(e.target.value);onDirty(true)}} /></label><label>Y<input aria-label="Position Y" type="number" min={-100000} max={100000} step={1} required value={y} onChange={e=>{setY(e.target.value);onDirty(true)}} /></label></div>
      <div className="button-group"><button className="text-action" type="submit" disabled={busy}>Keep position</button><button className="text-action" type="button" onClick={()=>{setX(String(position?.x??0));setY(String(position?.y??0));onDirty(false)}}>Clear edits</button></div>
    </form>
  </section>
}

function SizeControls({size,busy,onDirty,onKeep}:{size:{width:number;height:number};busy:boolean;onDirty:(v:boolean)=>void;onKeep:(s:{width:number;height:number}|null)=>Promise<boolean>}) {
 const [width,setWidth]=useState(String(size.width)),[height,setHeight]=useState(String(size.height))
 useEffect(()=>()=>onDirty(false),[onDirty])
 return <section className="position-controls" aria-label="Node size"><h3>Size</h3>
 <form onSubmit={async e=>{e.preventDefault();const s={width:Number(width),height:Number(height)};if(!Number.isInteger(s.width)||!Number.isInteger(s.height)||s.width<80||s.width>1600||s.height<48||s.height>1200)return;if(await onKeep(s))onDirty(false)}}>
 <div className="position-fields"><label>Width<input aria-label="Node width" type="number" min={80} max={1600} step={1} required value={width} onChange={e=>{setWidth(e.target.value);onDirty(true)}} /></label><label>Height<input aria-label="Node height" type="number" min={48} max={1200} step={1} required value={height} onChange={e=>{setHeight(e.target.value);onDirty(true)}} /></label></div>
 <div className="size-actions"><button className="inline-action" disabled={busy} type="submit">Keep size</button><button className="secondary-action" disabled={busy} type="button" onClick={async()=>{if(await onKeep(null)){onDirty(false)}}}>Restore default size</button><button className="text-action" type="button" onClick={()=>{setWidth(String(size.width));setHeight(String(size.height));onDirty(false)}}>Clear edits</button></div>
 </form></section>
}

function RouteControls({route,busy,onDirty,onKeep}:{route:RouteProjection;busy:boolean;onDirty:(v:boolean)=>void;onKeep:(r:{bend:number}|null)=>Promise<boolean>}){
 const [bend,setBend]=useState(String(route.display_bend)),[dirty,setDirty]=useState(false)
 useEffect(()=>()=>onDirty(false),[onDirty])
 return <details className="position-controls" onToggle={e=>{if(dirty&&!e.currentTarget.open)e.currentTarget.open=true}}><summary>Route · {route.route?'Custom':'Default'}</summary>
 <form onSubmit={async e=>{e.preventDefault();const n=Number(bend);if(!bend||!Number.isInteger(n)||Math.abs(n)>100000)return;if(await onKeep({bend:n})){setDirty(false);onDirty(false)}}}>
 <label>Bend<input aria-label="Bend" type="number" min={-100000} max={100000} step={1} required value={bend} onChange={e=>{setBend(e.target.value);setDirty(true);onDirty(true)}}/></label>
 <p className="field-hint">The diamond is the control point. Drag it to bend the link.</p>
 <div className="size-actions"><button className="inline-action" type="submit" disabled={busy}>Keep route</button><button className="secondary-action" type="button" disabled={busy} onClick={async()=>{if(await onKeep(null)){setDirty(false);onDirty(false)}}}>Restore default</button><button className="text-action" type="button" onClick={()=>{setBend(String(route.display_bend));setDirty(false);onDirty(false)}}>Clear edits</button></div>
 </form></details>
}


type RelationshipValue = { target_id: string; label: string }
type RelationshipRow = RelationshipValue & { rowKey: string }

type ChangeReview = {
  reviewed_state: string
  diff: string
  base_revision: string
  candidate_tree: string
  generation: number
  before: ReviewSnapshot
  with_changes: ReviewSnapshot
  comparison: {
	  node_positions?: {diagram_id:string;component_id:string;before:{x:number;y:number}|null;with:{x:number;y:number}|null;before_source?:string;with_source?:string;path:string}[]
	  node_sizes?: {diagram_id:string;component_id:string;before:{width:number;height:number}|null;with:{width:number;height:number}|null;before_source?:string;with_source?:string;path:string}[]
	  node_shapes?:{diagram_id:string;component_id:string;before:string|null;with:string|null;path:string}[]
	  diagram_notes?:{diagram_id:string;note_id:string;before:Omit<DiagramNote,'id'>|null;with:Omit<DiagramNote,'id'>|null;path:string}[]
	  edge_routes?: (Pick<RouteProjection,'diagram_id'|'source_id'|'target_id'|'label'|'occurrence'>&{before:{bend:number}|null;with:{bend:number}|null;before_state:string;with_state:string;path:string})[]
    components: ReviewMapComponentChange[]
    relationships: ReviewMapRelationshipChange[]
    diagrams?: { diagram_id: string; title: string; status: 'added' | 'title_changed'; path: string }[]
    appearances?: { diagram_id: string; component_id: string; role: 'home' | 'reference'; status: 'added' | 'removed' | 'detail_changed'; side: 'before' | 'with_changes'; detail_diagram_id?: string; path: string }[]
  }
}
type ReviewAppearanceChange = NonNullable<ChangeReview['comparison']['appearances']>[number]

type ReviewAnchor = {
  kind: 'proposal' | 'proposal_markdown' | 'component' | 'component_markdown' | 'diagram' | 'composition' | 'relationship'
  side?: 'before' | 'with_changes'
  component_id?: string
  diagram_id?: string
  aspect?: 'home' | 'reference' | 'detail'
  detail_diagram_id?: string
  source_component_id?: string
  target_component_id?: string
  label?: string
  occurrence?: number
  start_line?: number
  end_line?: number
}

type ReviewSubmissionSummary = {
  id: string
  change_set_id: string
  reviewed_state: string
  binding: { base_revision: string; candidate_tree: string; generation: number }
  verdict: 'comment' | 'approve' | 'request_changes'
  author: string
  submitted_at: string
  comment_count: number
  lifecycle: 'active' | 'applied' | 'no_longer_active'
  current_generation: boolean
  out_of_date?: boolean
}

type ReviewSubmission = ReviewSubmissionSummary & {
  body: string
  comments: { id: string; body: string; anchor: ReviewAnchor }[]
  proposal_markdown: string
  review: ChangeReview
}

type ReviewSubmissionComment = ReviewSubmission['comments'][number]

type ReviewAnnotationGroup = {
  key: string
  label: string
  side?: ReviewAnchor['side']
  comments: ReviewSubmissionComment[]
  mapTarget?: { kind: 'node' | 'relationship'; id: string; diagramID?: string }
}

type ReviewCommentTarget = {
  editorKey?: string
  contextKey: string
  label: string
  anchor: ReviewAnchor
  source?: string
  mapTarget?: { kind: 'node' | 'relationship'; id: string; diagramID?: string }
  localCommentID?: string
  initialBody?: string
}

type LocalReviewComment = ReviewSubmissionComment
type ReviewPresentation = Pick<ReviewSubmission, 'review' | 'proposal_markdown'>

type ReviewSnapshot = {
  revision: string
  format_version?: number
  component_count: number
  component_titles: string[]
  components: AuthoringComponent[]
  root_diagram_id?: string
  diagrams?: DiagramProjection[]
}

type ReviewSide = 'with' | 'before'

type ReviewFocus =
  | { kind: 'component'; key: string; componentID: string; title: string; path: string; status: 'added' | 'content_changed' | 'unchanged' }
  | ({ kind: 'relationship' } & ReviewRelationshipSelection)
  | { kind: 'diagram'; key: string; diagramID: string; title: string; description?: string; path: string; status: 'added' | 'title_changed' | 'appearance_changed'; componentID?: string; role?: 'home' | 'reference'; compositionAspect?: 'home' | 'reference' | 'detail'; detailDiagramID?: string; reviewSide?: ReviewSide }

type ComponentEditor = {
  kind: 'add' | 'edit'
  id?: string
  homeDiagramID?: string
  title: string
  description: string
  descriptionPrefix: string
  titleChanged: boolean
  descriptionChanged: boolean
  initialTitle: string
  initialDescription: string
  relationships: RelationshipRow[]
  initialRelationships: RelationshipValue[]
  relationshipIssue?: { position: number; field: 'target' | 'label' }
  readOnly?: boolean
}

type ParentOption = { component_id: string; title: string; home_diagram_id: string; home_diagram_title: string; filename: string }
type ParentOptions = { diagram_id: string; title: string; current_anchor: ParentOption; eligible: ParentOption[]; candidate_tree: string; generation: number | null }

type DiagramEditor =
  | { kind: 'parent'; diagramID: string; options: ParentOptions; anchorID: string }
  | { kind: 'detail'; componentID: string; title: string; initialTitle: string; invalid?: boolean }
  | { kind: 'title'; diagramID: string; title: string; initialTitle: string; invalid?: boolean }
  | { kind: 'move'; componentID: string; componentTitle: string; diagramID: string; currentDiagramTitle: string }

type WorkspaceTask = 'documentation' | 'changes' | 'empty'

type NavigationIntent =
  | { kind: 'reconcile' }
  | { kind: 'component'; id: string }
  | { kind: 'diagram'; id: string; focusComponentID?: string }
  | { kind: 'changes' }
  | { kind: 'context'; id: string }
  | { kind: 'new-changes' }
  | { kind: 'add' }
  | { kind: 'edit-diagram-title'; id: string; title: string }
  | { kind: 'change-parent'; id: string }
  | { kind: 'submitted-review'; changeSetID: string; reviewID: string }
  | { kind: 'open-another' }
  | { kind: 'route'; slug?: string; proposalChangeSetID?: string; reviewChangeSetID?: string; submittedReviewID?: string }
  | { kind: 'refresh' }
  | { kind: 'clear' }
  | { kind: 'review-result'; result: ArchitectureResult }
  | { kind: 'review-context-replacement'; apply: () => void }
  | { kind: 'authoring-pane'; apply: () => void }
  | { kind: 'continue-editing' }

type ErrorCode =
  | 'name_required'
  | 'project_not_found'
  | 'catalog_conflict'
  | 'catalog_unavailable'
  | 'origin_mismatch'
  | 'lookup_failed'
  | 'project_create_failed'
  | 'architecture_unavailable'
  | 'architecture_invalid'
  | 'architecture_unsupported'

type ErrorPayload = { code?: string }

type ViewState =
  | { kind: 'catalog'; projects: CatalogProject[] }
  | { kind: 'looking' }
  | { kind: 'ready'; value: ArchitectureResult }
  | { kind: 'not-found'; slug: string }
  | { kind: 'catalog-error'; message: string }

let relationshipRowCounter = 0

function newRelationshipRow(value: RelationshipValue = { target_id: '', label: '' }): RelationshipRow {
  relationshipRowCounter += 1
  return { ...value, rowKey: `relationship-row-${relationshipRowCounter}` }
}

function relationshipRows(values: RelationshipValue[]): RelationshipRow[] {
  return values.map((value) => newRelationshipRow(value))
}

function editorDescription(source: string): { description: string; prefix: string } {
  if (source.startsWith('\r\n')) return { description: source.slice(2), prefix: '\r\n' }
  if (source.startsWith('\n')) return { description: source.slice(1), prefix: '\n' }
  return { description: source, prefix: '' }
}

function relationshipValues(values: Array<RelationshipValue | RelationshipRow>): RelationshipValue[] {
  return values.map(({ target_id, label }) => ({ target_id, label }))
}

function sameRelationships(left: RelationshipValue[], right: RelationshipValue[]) {
  return left.length === right.length && left.every((value, index) => value.target_id === right[index].target_id && value.label === right[index].label)
}

function relationshipTargetsFor(result: ArchitectureResult): RelationshipTarget[] {
  if (result.changes?.relationship_targets) return result.changes.relationship_targets
  const titleCounts = new Map<string, number>()
  for (const component of result.components ?? []) {
    titleCounts.set(component.title, (titleCounts.get(component.title) ?? 0) + 1)
  }
  return (result.components ?? []).map((component) => ({
    id: component.id,
    title: component.title,
    ...((titleCounts.get(component.title) ?? 0) > 1 ? { context: component.filename || component.id.slice(0, 8) } : {}),
  }))
}

function relationshipTargetLabel(target: RelationshipTarget) {
  const title = target.title.trim() || 'Untitled component'
  return [title, target.context, target.new ? 'New component' : ''].filter(Boolean).join(' — ')
}

function diagramAuthoringOptionLabel(diagram: DiagramAuthoringOption) {
  return `${diagram.title}${diagram.context ? ` — ${diagram.context}` : ''}`
}

function componentAuthoringLabel(components: AuthoringComponent[], componentID: string) {
  const component = components.find((candidate) => candidate.id === componentID)
  if (!component) return 'Component'
  const title = component.title.trim() || 'Untitled component'
  const collides = components.some((candidate) => candidate.id !== componentID && candidate.title === component.title)
  return collides ? `${title} — ${component.filename || component.id.slice(0, 8)}` : title
}

function relationshipIssueComponentName(changes: ChangesInProgress, component: PendingComponent) {
  const target = changes.relationship_targets?.find((candidate) => candidate.id === component.id)
  const title = (target?.title ?? component.title).trim() || 'Untitled component'
  return target?.context ? `${title} — ${target.context}` : title
}

function reviewSideValue(side: ReviewSide): ReviewAnchor['side'] {
  return side === 'before' ? 'before' : 'with_changes'
}

function annotationGroup(key: string, label: string, comments: ReviewSubmissionComment[], side?: ReviewAnchor['side']): ReviewAnnotationGroup | undefined {
  return comments.length ? { key, label, comments, side } : undefined
}

function componentAnnotation(comments: ReviewSubmissionComment[] | undefined, side: ReviewAnchor['side'], componentID: string, label: string) {
  return annotationGroup(
    `component:${side}:${componentID}`,
    label,
    comments?.filter((comment) => comment.anchor.side === side && comment.anchor.component_id === componentID &&
      (comment.anchor.kind === 'component' || comment.anchor.kind === 'component_markdown')) ?? [],
    side,
  )
}

function diagramAnnotation(comments: ReviewSubmissionComment[] | undefined, side: ReviewAnchor['side'], diagramID: string, label: string) {
  return annotationGroup(
    `diagram:${side}:${diagramID}`,
    label,
    comments?.filter((comment) => comment.anchor.side === side && comment.anchor.kind === 'diagram' && comment.anchor.diagram_id === diagramID) ?? [],
    side,
  )
}

function compositionAnnotation(comments: ReviewSubmissionComment[] | undefined, side: ReviewAnchor['side'], diagramID: string, componentID: string, label: string) {
  return annotationGroup(
    `composition:${side}:${diagramID}:${componentID}`,
    label,
    comments?.filter((comment) => comment.anchor.side === side && comment.anchor.kind === 'composition' &&
      comment.anchor.diagram_id === diagramID && comment.anchor.component_id === componentID) ?? [],
    side,
  )
}

function relationshipAnnotation(comments: ReviewSubmissionComment[] | undefined, side: ReviewAnchor['side'], sourceID: string, targetID: string, label: string, occurrence: number, context: string) {
  return annotationGroup(
    `relationship:${side}:${sourceID}:${targetID}:${label}:${occurrence}`,
    context,
    comments?.filter((comment) => comment.anchor.side === side && comment.anchor.kind === 'relationship' &&
      comment.anchor.source_component_id === sourceID && comment.anchor.target_component_id === targetID &&
      comment.anchor.label === label && comment.anchor.occurrence === occurrence) ?? [],
    side,
  )
}

type DiagramComponent = AuthoringComponent & { appearance: DiagramAppearance }

function componentsForDiagram(result: Pick<ArchitectureResult, 'components'> | ReviewSnapshot, diagram?: DiagramProjection): DiagramComponent[] {
  if (!diagram) return []
  const byID = new Map(result.components.map((component) => [component.id, component]))
  return diagram.appearances.flatMap((appearance) => {
    const component = byID.get(appearance.component_id)
    return component ? [{ ...component, appearance }] : []
  })
}

function mapComponentsForDiagram(result: Pick<ArchitectureResult, 'components'> | ReviewSnapshot, diagram?: DiagramProjection): MapComponent[] {
  if (!diagram) return []
  const byID = new Map(result.components.map((component) => [component.id, component]))
  const nodes = new Map<string, MapComponent>()
  for (const appearance of diagram.appearances) {
    const component = byID.get(appearance.component_id)
    if (!component) continue
    nodes.set(appearance.component_id, {
      id: appearance.component_id,
      component_id: appearance.component_id,
	  shape:diagram.shapes?.find(s=>s.component_id===appearance.component_id)?.shape,
      title: component.title,
      filename: component.filename,
      node_kind: appearance.role,
	  position: appearance.display_position ?? appearance.position,
	  size: appearance.display_size,
      relationships: [],
    })
  }
  for (const boundary of diagram.boundaries) {
    nodes.set(boundary.key, {
      id: boundary.key,
      component_id: boundary.component_id,
	  shape:diagram.shapes?.find(s=>s.component_id===boundary.component_id)?.shape,
      title: boundary.title,
      node_kind: 'boundary',
      boundary_home_title: boundary.home_diagram_title,
      position: boundary.display_position ?? boundary.position,
      size: boundary.display_size,
      relationships: [],
    })
  }
  for (const relationship of diagram.relationships) {
    nodes.get(relationship.source_node_key)?.relationships.push({
      target_id: relationship.target_node_key,
      label: relationship.label,
      projection_key: relationship.key,
      routing: relationship.routing,
    })
  }
  return [...nodes.values(),...(diagram.notes??[]).map(n=>({id:`note:${n.id}`,title:n.text,note:true,shape:'rectangle' as const,position:{x:n.x,y:n.y},size:{width:n.width,height:n.height},relationships:[]}))]
}

function canonicalReviewPath(component: AuthoringComponent) {
  return `components/${component.filename}`
}

function componentStatusText(status: Extract<ReviewFocus, { kind: 'component' }>['status']) {
  if (status === 'added') return 'Added component'
  if (status === 'content_changed') return 'Content changed'
  return 'Unchanged component'
}

function relationshipStatusText(status: Extract<ReviewFocus, { kind: 'relationship' }>['status']) {
  return status === 'added' ? 'Added' : status === 'removed' ? 'Removed' : 'Unchanged'
}

function ShowingMenu({
  changeSets,
  selectedID,
  onSelect,
}: {
  changeSets: ChangesInProgress[]
  selectedID: string
  onSelect: (id: string) => void
}) {
  const labelID = useId()
  const triggerID = useId()
  const listboxID = useId()
  const openGroupID = useId()
  const acceptedGroupID = useId()
  const rootRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const listboxRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const active = changeSets.filter((changeSet) => changeSet.lifecycle === 'active')
  const applied = changeSets.filter((changeSet) => changeSet.lifecycle === 'applied')
  const nameCounts = new Map<string, number>()
  for (const changeSet of changeSets) nameCounts.set(changeSet.name, (nameCounts.get(changeSet.name) ?? 0) + 1)
  const proposalLabel = (changeSet: ChangesInProgress) => `${changeSet.name}${(nameCounts.get(changeSet.name) ?? 0) > 1 ? ` · ${changeSet.id.slice(0, 8)}` : ''}${changeSet.out_of_date ? ' · Out of date' : ''}`
  const options = [
    { id: 'accepted', label: 'Accepted' },
    ...active.map((changeSet) => ({ id: changeSet.id, label: proposalLabel(changeSet) })),
    ...applied.map((changeSet) => ({ id: changeSet.id, label: proposalLabel(changeSet) })),
  ]
  const selectedIndex = Math.max(0, options.findIndex((option) => option.id === selectedID))
  const [activeIndex, setActiveIndex] = useState(selectedIndex)

  useEffect(() => {
    if (!open) return
    listboxRef.current?.focus()
    const closeOnOutsidePress = (event: PointerEvent) => {
      if (event.target instanceof Node && !rootRef.current?.contains(event.target)) setOpen(false)
    }
    document.addEventListener('pointerdown', closeOnOutsidePress)
    return () => document.removeEventListener('pointerdown', closeOnOutsidePress)
  }, [open])

  const showMenu = (index = selectedIndex) => {
    setActiveIndex(index)
    setOpen(true)
  }
  const choose = (id: string) => {
    setOpen(false)
    if (id !== selectedID) onSelect(id)
    triggerRef.current?.focus()
  }
  const move = (offset: number) => setActiveIndex((current) => (current + offset + options.length) % options.length)
  const onTriggerKeyDown = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault()
      showMenu(event.key === 'ArrowDown' ? selectedIndex : (selectedIndex - 1 + options.length) % options.length)
    }
  }
  const onListboxKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault()
      move(event.key === 'ArrowDown' ? 1 : -1)
    } else if (event.key === 'Home' || event.key === 'End') {
      event.preventDefault()
      setActiveIndex(event.key === 'Home' ? 0 : options.length - 1)
    } else if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      choose(options[activeIndex].id)
    } else if (event.key === 'Escape') {
      event.preventDefault()
      setOpen(false)
      triggerRef.current?.focus()
    } else if (event.key === 'Tab') {
      setOpen(false)
    }
  }
  const option = ({ id, label }: { id: string; label: string }, index: number) => (
    <button
      id={`${listboxID}-option-${index}`}
      key={id}
      type="button"
      role="option"
      aria-selected={id === selectedID}
      className={index === activeIndex ? 'active' : undefined}
      onMouseEnter={() => setActiveIndex(index)}
      onClick={() => choose(id)}
    >
      <span>{label}</span>
      {id === selectedID && <span aria-hidden="true">✓</span>}
    </button>
  )

  return (
    <div className="showing-menu" ref={rootRef}>
      <span className="showing-label" id={labelID}>Showing</span>
      <button
        className="showing-trigger"
        id={triggerID}
        ref={triggerRef}
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listboxID : undefined}
        aria-labelledby={`${labelID} ${triggerID}`}
        onClick={() => open ? setOpen(false) : showMenu()}
        onKeyDown={onTriggerKeyDown}
      >
        <span>{options[selectedIndex].label}</span><span className="showing-chevron" aria-hidden="true">▾</span>
      </button>
      {open && (
        <div
          className="showing-listbox"
          id={listboxID}
          ref={listboxRef}
          role="listbox"
          tabIndex={-1}
          aria-labelledby={labelID}
          aria-activedescendant={`${listboxID}-option-${activeIndex}`}
          onKeyDown={onListboxKeyDown}
        >
          {option(options[0], 0)}
          {active.length > 0 && (
            <div className="showing-group" role="group" aria-labelledby={openGroupID}>
              <div className="showing-group-label" id={openGroupID}>Open proposals</div>
              {active.map((changeSet, index) => option({ id: changeSet.id, label: proposalLabel(changeSet) }, index + 1))}
            </div>
          )}
          {applied.length > 0 && (
            <div className="showing-group" role="group" aria-labelledby={acceptedGroupID}>
              <div className="showing-group-label" id={acceptedGroupID}>Accepted proposals</div>
              {applied.map((changeSet, index) => option({ id: changeSet.id, label: proposalLabel(changeSet) }, index + active.length + 1))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function App() {
  const [projectName, setProjectName] = useState('')
  const [state, setState] = useState<ViewState>({ kind: 'looking' })
  const [editor, setEditor] = useState<ComponentEditor | null>(null)
  const [diagramEditor, setDiagramEditor] = useState<DiagramEditor | null>(null)
  const [reconciliation, setReconciliation] = useState<ReconciliationPreview | null>(null)
  const [reconciliationDirty, setReconciliationDirty] = useState(false)
	const [placementBlocked,setPlacementBlocked]=useState(false)
  const [authoringError, setAuthoringError] = useState('')
  const [architectureNotice, setArchitectureNotice] = useState('')
  const [architectureBusy, setArchitectureBusy] = useState(false)
  const [acceptanceUnknown, setAcceptanceUnknown] = useState(false)
  const [selectedComponentID, setSelectedComponentID] = useState<string>()
  const [selectedDiagramID, setSelectedDiagramID] = useState<string>()
  const [workspaceTask, setWorkspaceTask] = useState<WorkspaceTask>('empty')
  const [navigationIntent, setNavigationIntent] = useState<NavigationIntent | null>(null)
  const [discardConfirming, setDiscardConfirming] = useState(false)
  const [reviewSide, setReviewSide] = useState<ReviewSide>('with')
  const [reviewFocus, setReviewFocus] = useState<ReviewFocus | null>(null)
  const [reviewSelectionCleared, setReviewSelectionCleared] = useState(false)
  const [reviewVisible, setReviewVisible] = useState(false)
  const [selectedContextID, setSelectedContextID] = useState('accepted')
  const [changeSetTextDirty, setChangeSetTextDirty] = useState(false)
  const [positionDirty, setPositionDirty] = useState(false)
  const [sizeDirty, setSizeDirty] = useState(false)
	const [noteDirty,setNoteDirty]=useState(false)
	const [noteSelection,setNoteSelection]=useState<{diagramID:string;id:string}|null>(null)
  const [routeDirty,setRouteDirty]=useState(false)
  const [selectedRouteKey,setSelectedRouteKey]=useState<string>()
  const [positionDraftEpoch, setPositionDraftEpoch] = useState(0)
  const [reviewCommentDirty, setReviewCommentDirty] = useState(false)
  const [reviewCommentTarget, setReviewCommentTarget] = useState<ReviewCommentTarget>()
  const openCommentEditor = (target: ReviewCommentTarget) => setReviewCommentTarget({ ...target, editorKey: crypto.randomUUID() })
  const [creatingChangeSet, setCreatingChangeSet] = useState(false)
  const [newChangeSetName, setNewChangeSetName] = useState('')
  const [openReviewAnnotations, setOpenReviewAnnotations] = useState<Record<string, ReviewAnnotationGroup>>({})
  const [localReviewComments, setLocalReviewComments] = useState<LocalReviewComment[]>([])
  const workingPaneRef = useRef<HTMLElement>(null)
  const newChangesPushedHistoryRef = useRef(false)
  const selectedContextIDRef = useRef(selectedContextID)
  selectedContextIDRef.current = selectedContextID
  const reviewVisibleRef = useRef(reviewVisible)
  reviewVisibleRef.current = reviewVisible
  const resetWorkingPaneScroll = useCallback(() => {
    if (workingPaneRef.current) workingPaneRef.current.scrollTop = 0
  }, [])
  const toggleReviewAnnotation = useCallback((group: ReviewAnnotationGroup) => {
    const targetSide = group.side === 'before' ? 'before' : group.side === 'with_changes' ? 'with' : undefined
    const switchingSide = Boolean(targetSide && targetSide !== reviewSide)
    if (targetSide) setReviewSide(targetSide)
    setOpenReviewAnnotations((current) => {
      if (current[group.key] && !switchingSide) {
        const next = { ...current }
        delete next[group.key]
        return next
      }
      return { ...current, [group.key]: group }
    })
  }, [reviewSide])

  const enterWorkspace = useCallback((incoming: ArchitectureResult, task?: WorkspaceTask, requestedContextID?: string) => {
    const changeSets = incoming.change_sets ?? (incoming.changes ? [incoming.changes] : [])
    const contextID = requestedContextID ?? incoming.action_change_set_id ?? incoming.changes?.id ?? selectedContextIDRef.current
    const selectedChangeSet = contextID === 'accepted' ? undefined
      : incoming.submitted_review && incoming.changes?.id === contextID
        ? incoming.changes
        : changeSets.find((changeSet) => changeSet.id === contextID)
    const result = { ...incoming, change_sets: changeSets, changes: selectedChangeSet }
    setSelectedContextID(selectedChangeSet?.id ?? 'accepted')
    setState({ kind: 'ready', value: result })
    const contextProjection = selectedChangeSet?.candidate ?? (selectedChangeSet ? undefined : result)
    if (contextProjection && (contextProjection.format_version ?? 0) >= 2) {
      const selectedDiagram = contextProjection.diagrams?.find((diagram) => diagram.id === selectedDiagramID)
        ?? contextProjection.diagrams?.find((diagram) => diagram.id === contextProjection.root_diagram_id)
      setSelectedDiagramID(selectedDiagram?.id)
      setSelectedComponentID((current) => selectedDiagram?.appearances.some((appearance) => appearance.component_id === current) || selectedDiagram?.boundaries.some(b=>b.component_id===current)
        ? current
        : selectedDiagram?.appearances[0]?.component_id)
    } else {
      setSelectedDiagramID(undefined)
      setSelectedComponentID((current) => contextProjection?.components?.some((component) => component.id === current) ? current : contextProjection?.components?.[0]?.id)
    }
    setWorkspaceTask(task ?? (selectedChangeSet ? 'changes' : result.components?.length ? 'documentation' : 'empty'))
  }, [selectedDiagramID])

  function enterProposalRoute(result: ArchitectureResult, changeSetID: string, historyMode: 'push' | 'replace' | 'none') {
    const selected = result.change_sets.find((changeSet) => changeSet.id === changeSetID)
    setEditor(null)
    setDiagramEditor(null)
    setCreatingChangeSet(false)
    setNewChangeSetName('')
    setDiscardConfirming(false)
    setChangeSetTextDirty(false)
    if (selected) {
      enterWorkspace({ ...result, submitted_review: undefined, changes: selected, action_change_set_id: undefined }, 'changes', changeSetID)
      setReviewVisible(false)
      setArchitectureNotice('')
      const path = proposalRoutePath(result.project_slug, changeSetID)
      if (historyMode === 'push') window.history.pushState({}, '', path)
      if (historyMode === 'replace') window.history.replaceState({}, '', path)
    } else {
      enterWorkspace({ ...result, submitted_review: undefined, changes: undefined, action_change_set_id: undefined }, result.components.length ? 'documentation' : 'empty', 'accepted')
      setReviewVisible(false)
      setArchitectureNotice('That proposal is no longer available.')
      window.history.replaceState({}, '', projectRoutePath(result.project_slug))
    }
    resetWorkingPaneScroll()
  }

  function enterProposalResult(result: ArchitectureResult, requestedChangeSetID?: string) {
    const changeSetID = requestedChangeSetID ?? result.action_change_set_id ?? result.changes?.id
    if (!changeSetID) {
      enterWorkspace(result, 'changes')
      return
    }
    const path = proposalRoutePath(result.project_slug, changeSetID)
    enterProposalRoute(result, changeSetID, window.location.pathname === path ? 'none' : 'push')
  }

  function enterReviewRoute(result: ArchitectureResult, changeSetID: string, historyMode: 'push' | 'replace' | 'none') {
    const selected = result.change_sets.find((changeSet) => changeSet.id === changeSetID)
    setEditor(null)
    setDiagramEditor(null)
    setCreatingChangeSet(false)
    setNewChangeSetName('')
    setDiscardConfirming(false)
    setChangeSetTextDirty(false)
    if (selected?.review) {
      enterWorkspace({ ...result, submitted_review: undefined, changes: selected, action_change_set_id: undefined }, 'changes', changeSetID)
      setReviewVisible(true)
      setArchitectureNotice(result.action_error ? messageForArchitectureAction(result.action_error) : '')
      const path = reviewRoutePath(result.project_slug, changeSetID)
      if (historyMode === 'push') window.history.pushState({}, '', path)
      if (historyMode === 'replace') window.history.replaceState({}, '', path)
    } else if (selected) {
      enterWorkspace({ ...result, submitted_review: undefined, changes: selected, action_change_set_id: undefined }, 'changes', changeSetID)
      setReviewVisible(false)
      setArchitectureNotice('This proposal has changed and needs to be reviewed again.')
      window.history.replaceState({}, '', proposalRoutePath(result.project_slug, changeSetID))
    } else {
      enterWorkspace({ ...result, submitted_review: undefined, changes: undefined, action_change_set_id: undefined }, result.components.length ? 'documentation' : 'empty', 'accepted')
      setReviewVisible(false)
      setArchitectureNotice('That review is no longer available.')
      window.history.replaceState({}, '', projectRoutePath(result.project_slug))
    }
    resetWorkingPaneScroll()
  }

  async function openSubmittedReview(result: ArchitectureResult, changeSetID: string, reviewID: string, historyMode: 'push' | 'replace' | 'none') {
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/architecture/review-submissions/inspect', {
        project_slug: result.project_slug, store_id: result.store_id, change_set_id: changeSetID, review_id: reviewID,
      })
      const payload = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload) || !payload.submitted_review || !payload.changes?.review) {
        setArchitectureNotice('That submitted review is not available.')
        return
      }
      enterWorkspace(payload, 'changes', changeSetID)
      setReviewVisible(true)
      setReviewSide('with')
      setReviewFocus(null)
      setArchitectureNotice(payload.action_error ? messageForArchitectureAction(payload.action_error) : '')
      const path = submittedReviewRoutePath(payload.project_slug, changeSetID, reviewID)
      if (historyMode === 'push') window.history.pushState({}, '', path)
      if (historyMode === 'replace') window.history.replaceState({}, '', path)
      resetWorkingPaneScroll()
    } catch {
      setArchitectureNotice("WorkBraid couldn't open that submitted review. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  function leaveReviewRoute(result: ArchitectureResult, updateHistory: boolean) {
    if (result.changes) {
      enterProposalRoute(result, result.changes.id, updateHistory ? 'push' : 'none')
      return
    }
    setReviewVisible(false)
    setWorkspaceTask(result.changes ? 'changes' : result.components.length ? 'documentation' : 'empty')
    setArchitectureNotice('')
    if (updateHistory && window.location.pathname !== projectRoutePath(result.project_slug)) {
      window.history.pushState({}, '', projectRoutePath(result.project_slug))
    }
    resetWorkingPaneScroll()
  }

  const readyResult = state.kind === 'ready' ? state.value : undefined
  const currentReview = readyResult?.submitted_review?.review ?? readyResult?.changes?.review
  const reviewIdentity = currentReview ? `${readyResult?.submitted_review?.id ?? ''}:${currentReview.base_revision}:${currentReview.candidate_tree}:${currentReview.generation}` : ''
  const editorDirty = editor !== null && (
    editor.title !== editor.initialTitle || editor.description !== editor.initialDescription ||
    !sameRelationships(relationshipValues(editor.relationships), editor.initialRelationships)
  )
  const diagramEditorDirty = diagramEditor !== null && (diagramEditor.kind === 'parent' ? diagramEditor.anchorID !== '' : diagramEditor.kind === 'move'
    ? diagramEditor.diagramID !== ''
    : diagramEditor.title !== diagramEditor.initialTitle)
  const newChangeSetNameDirty = creatingChangeSet && newChangeSetName.trim() !== ''
  const editorDirtyRef = useRef(editorDirty)
  editorDirtyRef.current = editorDirty || diagramEditorDirty || changeSetTextDirty || positionDirty || sizeDirty || noteDirty || routeDirty || reviewCommentDirty || newChangeSetNameDirty || reconciliationDirty
  const stateRef = useRef(state)
  stateRef.current = state

  useEffect(() => {
    setOpenReviewAnnotations({})
    setLocalReviewComments([])
    setReviewCommentTarget(undefined)
    setReviewCommentDirty(false)
    if (!currentReview) {
      setReviewFocus(null)
      setReviewSelectionCleared(false)
      setReviewVisible(false)
      return
    }
    setReviewSide('with')
    setReviewFocus(null)
    setReviewSelectionCleared(false)
    const initialDiagram = (currentReview.with_changes.format_version ?? 0) >= 2
      ? currentReview.with_changes.diagrams?.find((diagram) => diagram.id === selectedDiagramID)
        ?? currentReview.with_changes.diagrams?.find((diagram) => diagram.id === currentReview.with_changes.root_diagram_id)
      : undefined
    const initialComponents = initialDiagram
      ? componentsForDiagram(currentReview.with_changes, initialDiagram)
      : currentReview.with_changes.components
    setSelectedDiagramID(initialDiagram?.id)
    setSelectedComponentID((current) => initialComponents.some((component) => component.id === current)
      ? current
      : initialComponents[0]?.id)
  }, [reviewIdentity])

  useEffect(() => {
    if (state.kind !== 'ready') return
    const heldReview = state.value.changes?.review
    const review = reviewVisible ? heldReview : undefined
    if (review) {
      const projection = reviewSide === 'with' ? review.with_changes : review.before
      if (reviewSelectionCleared) {
        if (selectedComponentID) setSelectedComponentID(undefined)
        return
      }
      if ((projection.format_version ?? 0) >= 2) {
        const diagram = projection.diagrams?.find((candidate) => candidate.id === selectedDiagramID)
          ?? projection.diagrams?.find((candidate) => candidate.id === projection.root_diagram_id)
        const active = diagram ? componentsForDiagram(projection, diagram) : []
        if (reviewFocus?.kind === 'relationship' && !active.some((component) => component.id === reviewFocus.source_id)) {
          if (selectedComponentID) setSelectedComponentID(undefined)
          return
        }
        if (selectedComponentID && (active.some((component) => component.id === selectedComponentID) || diagram?.boundaries.some(b=>b.component_id===selectedComponentID))) return
        setSelectedComponentID(active[0]?.id)
        return
      }
      const active = projection.components
      if (selectedComponentID && active.some((component) => component.id === selectedComponentID)) return
      if (selectedComponentID) setSelectedComponentID(undefined)
      return
    }
    const selectedProjection = state.value.changes
      ? state.value.changes.candidate
      : state.value
    if (!selectedProjection) {
      if (selectedComponentID) setSelectedComponentID(undefined)
      if (selectedDiagramID) setSelectedDiagramID(undefined)
      return
    }
    if ((selectedProjection.format_version ?? 0) >= 2) {
      const diagram = selectedProjection.diagrams?.find((candidate) => candidate.id === selectedDiagramID)
        ?? selectedProjection.diagrams?.find((candidate) => candidate.id === selectedProjection.root_diagram_id)
      if (diagram && diagram.id !== selectedDiagramID && !(heldReview && !reviewVisible)) {
        setSelectedDiagramID(diagram.id)
      }
      if (workspaceTask === 'empty') return
      if (selectedComponentID && (diagram?.appearances.some((appearance) => appearance.component_id === selectedComponentID) || diagram?.boundaries.some(b=>b.component_id===selectedComponentID))) return
      setSelectedComponentID(diagram?.appearances[0]?.component_id)
      return
    }
    if (workspaceTask === 'empty' && selectedProjection.components?.length) return
    if (selectedComponentID && selectedProjection.components?.some((component) => component.id === selectedComponentID)) return
    setSelectedComponentID(selectedProjection.components?.[0]?.id)
  }, [state, selectedComponentID, selectedDiagramID, reviewFocus, reviewSelectionCleared, reviewSide, reviewVisible, workspaceTask])

  useEffect(() => {
    const route = decodeProjectRoute(window.location.pathname)
    if (route?.slug) void openProject(route.slug, true, route.proposalChangeSetID, route.reviewChangeSetID, route.submittedReviewID)
    else void loadCatalog()
    const restoreHistoryRoute = () => {
      const target = decodeProjectRoute(window.location.pathname)
      const current = stateRef.current
      if (current.kind === 'ready' && target?.slug === current.value.project_slug) {
        const alreadyShowingTarget = target.submittedReviewID
          ? current.value.submitted_review?.id === target.submittedReviewID
          : target.reviewChangeSetID
          ? reviewVisibleRef.current && !current.value.submitted_review && current.value.changes?.id === target.reviewChangeSetID
          : target.proposalChangeSetID
            ? !reviewVisibleRef.current && !current.value.submitted_review && current.value.changes?.id === target.proposalChangeSetID
            : !reviewVisibleRef.current && !current.value.submitted_review && selectedContextIDRef.current === 'accepted'
        if (alreadyShowingTarget) return
      }
      if (editorDirtyRef.current && current.kind === 'ready') {
        const currentPath = reviewVisibleRef.current && current.value.changes?.review
          ? reviewRoutePath(current.value.project_slug, current.value.changes.id)
          : current.value.changes
            ? proposalRoutePath(current.value.project_slug, current.value.changes.id)
            : projectRoutePath(current.value.project_slug)
        window.history.pushState({}, '', currentPath)
        setNavigationIntent({ kind: 'route', slug: target?.slug, proposalChangeSetID: target?.proposalChangeSetID, reviewChangeSetID: target?.reviewChangeSetID, submittedReviewID: target?.submittedReviewID })
        return
      }
      void restoreRoute(target?.slug, target?.proposalChangeSetID, target?.reviewChangeSetID, target?.submittedReviewID)
    }
    window.addEventListener('popstate', restoreHistoryRoute)
    return () => window.removeEventListener('popstate', restoreHistoryRoute)
  }, [])

  async function loadCatalog() {
    setState({ kind: 'looking' })
    try {
      const response = await fetch('/api/projects')
      const payload = await response.json() as { projects?: CatalogProject[] } | ErrorPayload
      if (!response.ok || !('projects' in payload)) {
        setState({ kind: 'catalog-error', message: 'WorkBraid could not read the project catalog. Try again.' })
        return
      }
      setState({ kind: 'catalog', projects: payload.projects ?? [] })
    } catch {
      setState({ kind: 'catalog-error', message: 'WorkBraid could not read the project catalog. Try again.' })
    }
  }

  async function openProject(slug: string, replaceRoute = false, proposalChangeSetID?: string, reviewChangeSetID?: string, submittedReviewID?: string) {
	setPlacementBlocked(false)
    setState({ kind: 'looking' })
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/projects/open', { project_slug: slug })
      const result = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in result)) {
        if ('code' in result && result.code === 'project_not_found') setState({ kind: 'not-found', slug })
        else setState({ kind: 'catalog-error', message: messageForError('code' in result ? result.code : undefined) })
        return
      }
      if (reviewChangeSetID && submittedReviewID) {
        await openSubmittedReview(result, reviewChangeSetID, submittedReviewID, replaceRoute ? 'replace' : 'push')
      } else if (reviewChangeSetID) {
        enterReviewRoute(result, reviewChangeSetID, replaceRoute ? 'replace' : 'push')
      } else if (proposalChangeSetID) {
        enterProposalRoute(result, proposalChangeSetID, replaceRoute ? 'replace' : 'push')
      } else {
        window.history[replaceRoute ? 'replaceState' : 'pushState']({}, '', projectRoutePath(result.project_slug))
        enterWorkspace(result, undefined, 'accepted')
        setReviewVisible(false)
      }
    } catch {
      setState({ kind: 'catalog-error', message: 'WorkBraid could not open that project. Try again.' })
    }
  }

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const name = projectName.trim()
    setState({ kind: 'looking' })
    setArchitectureNotice('')
    setAcceptanceUnknown(false)
    try {
      const response = await postJSON('/api/projects/create', { name })
      const result = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in result)) {
        setState({ kind: 'catalog-error', message: messageForError('code' in result ? result.code : undefined) })
        return
      }
      window.history.pushState({}, '', `/projects/${encodeURIComponent(result.project_slug)}`)
      setProjectName('')
      enterWorkspace(result, undefined, 'accepted')
    } catch {
      setState({ kind: 'catalog-error', message: 'WorkBraid could not create that project. Try again.' })
    }
  }

  function editAccepted(component: AuthoringComponent, result: ArchitectureResult) {
    const pending = result.changes?.components.find((change) => change.id === component.id)
    const relationships = pending?.relationships ?? component.relationships ?? []
    const description = editorDescription(pending?.description ?? component.description)
    setAuthoringError('')
    setEditor({
      kind: 'edit',
      id: component.id,
      title: pending?.title ?? component.title,
      description: description.description,
      descriptionPrefix: description.prefix,
      titleChanged: false,
      descriptionChanged: false,
      initialTitle: pending?.title ?? component.title,
      initialDescription: description.description,
      relationships: relationshipRows(relationships),
      initialRelationships: relationshipValues(relationships),
    })
    setWorkspaceTask('documentation')
  }

  function addComponent(homeDiagramID?: string) {
    setAuthoringError('')
    setEditor({
      kind: 'add', homeDiagramID, title: '', description: '', descriptionPrefix: '', titleChanged: false, descriptionChanged: false,
      initialTitle: '', initialDescription: '', relationships: [], initialRelationships: [],
    })
  }

  function editPending(component: PendingComponent, relationshipIssue?: ComponentEditor['relationshipIssue'], readOnly = false) {
    const relationships = component.relationships ?? []
    const description = editorDescription(component.description)
    setAuthoringError('')
    setEditor({
      kind: 'edit', id: component.id, title: component.title, description: description.description,
      descriptionPrefix: description.prefix,
      titleChanged: false, descriptionChanged: false, initialTitle: component.title, initialDescription: description.description,
      relationships: relationshipRows(relationships), initialRelationships: relationshipValues(relationships),
      relationshipIssue, readOnly,
    })
    setWorkspaceTask('changes')
  }

  async function submitComponent(event: FormEvent<HTMLFormElement>, result: ArchitectureResult) {
    event.preventDefault()
    if (!editor) return
    setAuthoringError('')
    const endpoint = editor.kind === 'add' ? '/api/architecture/components/add' : '/api/architecture/components/edit'
    const relationships = relationshipValues(editor.relationships)
    const relationshipsChanged = !sameRelationships(relationships, editor.initialRelationships)
    try {
      const response = await postJSON(endpoint, {
        project_slug: result.project_slug,
        store_id: result.store_id,
        ...(result.changes ? { change_set_id: result.changes.id } : {}),
        expected_revision: result.revision,
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
        ...(editor.id ? { component_id: editor.id } : {}),
        ...(editor.kind === 'add' || editor.titleChanged ? { title: editor.title } : {}),
        ...(editor.kind === 'add' || editor.descriptionChanged ? { description: editor.descriptionPrefix + editor.description } : {}),
        ...(editor.kind === 'add' || relationshipsChanged ? { relationships } : {}),
        ...(editor.kind === 'edit' ? { title_changed: editor.titleChanged, description_changed: editor.descriptionChanged } : {}),
        ...(editor.kind === 'edit' && relationshipsChanged ? { relationships_changed: true } : {}),
        ...(editor.kind === 'add' && (editor.homeDiagramID ?? selectedDiagramID) ? { diagram_id: editor.homeDiagramID ?? selectedDiagramID } : {}),
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setAuthoringError(messageForAuthoringError('code' in payload ? payload.code : undefined))
        return
      }
      enterProposalResult(payload)
    } catch {
      setAuthoringError("WorkBraid couldn't keep that change. Try again.")
    }
  }

  async function reviewChanges(result: ArchitectureResult) {
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/architecture/review', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        ...(result.changes ? { change_set_id: result.changes.id } : {}),
        expected_revision: result.revision,
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if ('state' in payload) {
        setAcceptanceUnknown(false)
        if (payload.changes?.review && editorDirtyRef.current) {
          setNavigationIntent({ kind: 'review-result', result: payload })
        } else if (payload.changes?.review) {
          enterReviewRoute(payload, payload.changes.id, 'push')
        } else {
          enterWorkspace(payload, 'changes')
          setReviewVisible(false)
          resetWorkingPaneScroll()
        }
      } else {
        setArchitectureNotice(messageForArchitectureAction('code' in payload ? payload.code : undefined))
      }
    } catch {
      setArchitectureNotice("WorkBraid couldn't prepare these changes for review. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function submitReviewFeedback(result: ArchitectureResult, input: { author: string; verdict: ReviewSubmissionSummary['verdict']; body: string; comments: { body: string; anchor: ReviewAnchor }[] }) {
    const review = result.changes?.review
    if (!review || !result.changes) return
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/architecture/review-submissions/submit', {
        project_slug: result.project_slug, store_id: result.store_id, change_set_id: result.changes.id,
        reviewed_state: review.reviewed_state, base_revision: review.base_revision,
        candidate_tree: review.candidate_tree, generation: review.generation, ...input,
      })
      const payload = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload) || !payload.action_review_id) {
        const code = 'code' in payload ? payload.code : undefined
        setArchitectureNotice(code === 'review_changed'
          ? 'This proposal changed after you reviewed it. Prepare the review again before submitting feedback.'
          : code === 'review_anchor_invalid'
            ? 'One comment no longer points to this exact review. Check its location and try again.'
            : 'WorkBraid could not submit that review. Check the review and try again.')
        return
      }
      setChangeSetTextDirty(false)
      await openSubmittedReview(payload, result.changes.id, payload.action_review_id, 'push')
    } catch {
      setArchitectureNotice("WorkBraid couldn't submit that review. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function submitDiagramChange(event: FormEvent<HTMLFormElement>, result: ArchitectureResult) {
    event.preventDefault()
    if (!diagramEditor) return
    const endpoint = diagramEditor.kind === 'parent' ? '/api/architecture/diagrams/reassign-detail' : diagramEditor.kind === 'detail'
      ? '/api/architecture/diagrams/detail'
      : diagramEditor.kind === 'title'
        ? '/api/architecture/diagrams/title'
        : '/api/architecture/components/move-home'
    setAuthoringError('')
    try {
      const response = await postJSON(endpoint, {
        project_slug: result.project_slug,
        store_id: result.store_id,
        ...(result.changes ? { change_set_id: result.changes.id } : {}),
        expected_revision: result.revision,
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
        ...(diagramEditor.kind === 'parent' ? { diagram_id: diagramEditor.diagramID, anchor_component_id: diagramEditor.anchorID } : {}),
        ...(diagramEditor.kind === 'detail' ? { component_id: diagramEditor.componentID, title: diagramEditor.title } : {}),
        ...(diagramEditor.kind === 'title' ? { diagram_id: diagramEditor.diagramID, title: diagramEditor.title } : {}),
        ...(diagramEditor.kind === 'move' ? { component_id: diagramEditor.componentID, diagram_id: diagramEditor.diagramID } : {}),
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setAuthoringError('code' in payload && payload.code === 'home_move_unavailable'
          ? 'That destination is no longer available. Choose another diagram.'
          : "WorkBraid couldn't keep that diagram change. Check the selection and try again.")
        return
      }
      enterProposalResult(payload)
    } catch {
      setAuthoringError("WorkBraid couldn't keep that diagram change. Try again.")
    }
  }

  async function changeReference(result: ArchitectureResult, diagramID: string, componentID: string, present: boolean) {
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON(present
        ? '/api/architecture/diagrams/show-component'
        : '/api/architecture/diagrams/stop-showing-component', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        ...(result.changes ? { change_set_id: result.changes.id } : {}),
        expected_revision: result.revision,
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
        diagram_id: diagramID,
        component_id: componentID,
      })
      const payload = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setArchitectureNotice(present ? "WorkBraid couldn't show that component here." : "WorkBraid couldn't stop showing that component here.")
        return
      }
      enterProposalResult(payload)
    } catch {
      setArchitectureNotice("WorkBraid couldn't keep that diagram change. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function keepPosition(result:ArchitectureResult,diagramID:string,componentID:string|undefined,position:{x:number;y:number}|null):Promise<boolean> {
	const requestedRoute=window.location.pathname
	async function inspectAfterFailure() {
	  setPlacementBlocked(true)
	  try {
	    const status = await fetch('/api/agent/v2/status').then(response => response.json())
	    if (window.location.pathname !== requestedRoute || status.context?.project?.store_id !== result.store_id) return
	    await openProject(result.project_slug, true, result.changes?.id)
	    setSelectedDiagramID(diagramID)
	    if (componentID) setSelectedComponentID(componentID)
	  } catch { /* Keep placement blocked until the authority can be inspected. */ }
	}
	setArchitectureBusy(true)
	setArchitectureNotice('')
	try {
	  const response=await postJSON(`/api/architecture/diagrams/${componentID?'set-position':'auto-layout'}`,{
	    project_slug:result.project_slug,store_id:result.store_id,expected_revision:result.revision,
	    change_set_id:result.changes?.id,pending_generation_observed:true,expected_pending_generation:result.changes?.generation??null,
	    diagram_id:diagramID,component_id:componentID,...(position??{}),
	  })
	  const payload=await response.json() as ArchitectureResult|ErrorPayload
	  if(window.location.pathname!==requestedRoute)return false
	  if(!response.ok||!('state' in payload)||payload.action_error){
	    await inspectAfterFailure()
	    setArchitectureNotice('That position was not kept. The current layout has been reopened where available; inspect it before making another move.')
	    return false
	  }
	  if(payload.action_change_set_id||result.changes){enterProposalResult(payload);setWorkspaceTask('documentation')}
	  setSelectedDiagramID(diagramID)
	  if(componentID)setSelectedComponentID(componentID)
	  setChangeSetTextDirty(false)
	  if((result.changes?.candidate?.format_version??result.format_version)===2&&position)setArchitectureNotice('This proposal will save Diagram positions.')
	  return true
	} catch {
	  await inspectAfterFailure()
	  setArchitectureNotice('WorkBraid could not confirm that position. Inspect the current proposal list and layout before making another move; the move has not been retried.')
	  return false
	} finally {setArchitectureBusy(false)}
  }

  async function keepSize(result:ArchitectureResult,diagramID:string,componentID:string,size:{width:number;height:number}|null):Promise<boolean> {
	const requestedRoute=window.location.pathname
	async function inspectAfterFailure() {
	  setPlacementBlocked(true)
	  try {
	    const status = await fetch('/api/agent/v2/status').then(response => response.json())
	    if (window.location.pathname !== requestedRoute || status.context?.project?.store_id !== result.store_id) return
	    await openProject(result.project_slug, true, result.changes?.id)
	    setSelectedDiagramID(diagramID)
	    if (componentID) setSelectedComponentID(componentID)
	  } catch { /* Keep placement blocked until the authority can be inspected. */ }
	}
	setArchitectureBusy(true)
	setArchitectureNotice('')
	try {
	  const response=await postJSON(`/api/architecture/diagrams/${size?'set-size':'restore-default-size'}`,{
	    project_slug:result.project_slug,store_id:result.store_id,expected_revision:result.revision,
	    change_set_id:result.changes?.id,pending_generation_observed:true,expected_pending_generation:result.changes?.generation??null,
	    diagram_id:diagramID,component_id:componentID,...(size??{}),
	  })
	  const payload=await response.json() as ArchitectureResult|ErrorPayload
	  if(window.location.pathname!==requestedRoute)return false
	  if(!response.ok||!('state' in payload)||payload.action_error){
	    await inspectAfterFailure()
	    setArchitectureNotice('That size was not kept. The current layout has been reopened where available; inspect it before making another resize.')
	    return false
	  }
	  if(payload.action_change_set_id||result.changes){enterProposalResult(payload);setWorkspaceTask('documentation')}
	  setSelectedDiagramID(diagramID)
	  if(componentID)setSelectedComponentID(componentID)
	  setChangeSetTextDirty(false)

	  return true
	} catch {
	  await inspectAfterFailure()
	  setArchitectureNotice('WorkBraid could not confirm that size. Inspect the current proposal list and layout before making another resize; the resize has not been retried.')
	  return false
	} finally {setArchitectureBusy(false)}
  }

  async function keepShapeNote(result:ArchitectureResult,diagramID:string,action:string,values:Record<string,unknown>):Promise<boolean>{
    const route=window.location.pathname
    setArchitectureBusy(true);setArchitectureNotice('')
    try{
      const response=await postJSON(`/api/architecture/diagrams/${action}`,{project_slug:result.project_slug,store_id:result.store_id,expected_revision:result.revision,change_set_id:result.changes?.id,pending_generation_observed:true,expected_pending_generation:result.changes?.generation??null,diagram_id:diagramID,...values})
      const payload=await response.json() as ArchitectureResult|ErrorPayload
      if(window.location.pathname!==route)return false
      if(!response.ok||!('state' in payload)||payload.action_error){setPlacementBlocked(true);setArchitectureNotice('That change was not kept. Refresh and inspect the current proposal before editing again.');return false}
      const oldNotes=(result.changes?.candidate??result).diagrams?.find(d=>d.id===diagramID)?.notes??[]
      if(payload.action_change_set_id||result.changes){enterProposalResult(payload);setWorkspaceTask('documentation')}
      setSelectedDiagramID(diagramID);setNoteDirty(false)
      if(action==='add-note'){const added=(payload.changes?.candidate??payload).diagrams?.find(d=>d.id===diagramID)?.notes?.find(n=>!oldNotes.some(old=>old.id===n.id));if(added)setNoteSelection({diagramID,id:added.id})}
      if(action==='delete-note')setNoteSelection(null)
      return true
    }catch{setPlacementBlocked(true);setArchitectureNotice('WorkBraid could not confirm that change. Refresh and inspect before trying again.');return false}finally{setArchitectureBusy(false)}
  }
  async function keepNoteGeometry(result:ArchitectureResult,diagram:DiagramProjection,key:string,geometry:Partial<Pick<DiagramNote,'x'|'y'|'width'|'height'>>):Promise<boolean>{
    const note=diagram.notes?.find(n=>`note:${n.id}`===key);if(!note)return false
    const {id,...value}=note
    return keepShapeNote(result,diagram.id,'edit-note',{note_id:id,...value,...geometry})
  }


  async function keepRoute(result:ArchitectureResult,address:RouteProjection,route:{bend:number}|null):Promise<boolean> {
	const diagramID=address.diagram_id
	const requestedRoute=window.location.pathname
	async function inspectAfterFailure() {
	  setPlacementBlocked(true)
	  try {
	    const status = await fetch('/api/agent/v2/status').then(response => response.json())
	    if (window.location.pathname !== requestedRoute || status.context?.project?.store_id !== result.store_id) return
	    await openProject(result.project_slug, true, result.changes?.id)
	    setSelectedDiagramID(diagramID)

	  } catch { /* Keep placement blocked until the authority can be inspected. */ }
	}
	setArchitectureBusy(true)
	setArchitectureNotice('')
	try {
	  const response=await postJSON(`/api/architecture/diagrams/${route?'set-route':'restore-default-route'}`,{
	    project_slug:result.project_slug,store_id:result.store_id,expected_revision:result.revision,
	    change_set_id:result.changes?.id,pending_generation_observed:true,expected_pending_generation:result.changes?.generation??null,
	    diagram_id:diagramID,source_id:address.source_id,target_id:address.target_id,label:address.label,occurrence:address.occurrence,...(route??{}),
	  })
	  const payload=await response.json() as ArchitectureResult|ErrorPayload
	  if(window.location.pathname!==requestedRoute)return false
	  if(!response.ok||!('state' in payload)||payload.action_error){
	    await inspectAfterFailure()
	    setArchitectureNotice('That route was not kept. The current layout has been reopened where available; inspect it before making another route change.')
	    return false
	  }
	  if(payload.action_change_set_id||result.changes){enterProposalResult(payload);setWorkspaceTask('documentation')}
	  setSelectedDiagramID(diagramID)

	  setChangeSetTextDirty(false);setRouteDirty(false)

	  return true
	} catch {
	  await inspectAfterFailure()
	  setArchitectureNotice('WorkBraid could not confirm that route. Inspect the current proposal list and layout before making another route change; the route change has not been retried.')
	  return false
	} finally {setArchitectureBusy(false)}
  }


  async function updateArchitecture(result: ArchitectureResult) {
    const review = result.changes?.review
    if (!review) return
    setArchitectureBusy(true)
    setArchitectureNotice('')
    setAcceptanceUnknown(true)
    setReviewVisible(false)
    window.history.replaceState({}, '', projectRoutePath(result.project_slug))
    enterWorkspace({ ...result, changes: result.changes ? { ...result.changes, review: undefined } : undefined }, 'changes')
    try {
      const response = await postJSON('/api/architecture/accept', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        ...(result.changes ? { change_set_id: result.changes.id } : {}),
        base_revision: review.base_revision,
        candidate_tree: review.candidate_tree,
        generation: review.generation,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if ('state' in payload) {
        const accepted = response.ok && !payload.action_error && !payload.stale
        if (accepted) {
          const receipt = payload.changes
          if (payload.store_id !== result.store_id || payload.action_change_set_id !== result.changes?.id ||
            receipt?.id !== result.changes?.id || receipt?.lifecycle !== 'applied' || !receipt.applied_revision ||
            receipt.base_revision !== review.base_revision || receipt.generation !== review.generation || receipt.candidate_tree !== review.candidate_tree) {
            setArchitectureNotice('WorkBraid could not confirm what happened. Open this project again to check its current architecture.')
            return
          }
        }
        setAcceptanceUnknown(false)
        window.history.replaceState({}, '', projectRoutePath(payload.project_slug))
        if (accepted) {
          enterWorkspace({ ...payload, submitted_review: undefined, action_change_set_id: undefined }, payload.components.length ? 'documentation' : 'empty', 'accepted')
          setChangeSetTextDirty(false)
          resetWorkingPaneScroll()
        } else {
          enterWorkspace(payload, payload.changes ? 'changes' : 'documentation')
        }
      } else {
        setArchitectureNotice('WorkBraid could not confirm what happened. Open this project again to check its current architecture.')
      }
    } catch {
      setArchitectureNotice('WorkBraid could not confirm what happened. Open this project again to check its current architecture.')
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function createChangeSet(result: ArchitectureResult) {
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/architecture/change-sets/create', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        accepted_revision: result.revision,
        ...(newChangeSetName.trim() ? { name: newChangeSetName } : {}),
      })
      const payload = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setArchitectureNotice("WorkBraid couldn't create that proposal. Choose a different open proposal name or Refresh.")
        return
      }
      const changeSetID = payload.action_change_set_id ?? payload.changes?.id
      if (changeSetID) enterProposalRoute(payload, changeSetID, 'push')
      else enterWorkspace(payload, 'changes')
    } catch {
      setArchitectureNotice("WorkBraid couldn't create that proposal. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function renameChangeSet(result: ArchitectureResult, name: string) {
    const changes = result.changes
    if (!changes) return
    await updateChangeSetText(result, '/api/architecture/change-sets/rename', { name })
  }

  async function saveProposal(result: ArchitectureResult, proposal: string) {
    const changes = result.changes
    if (!changes) return
    await updateChangeSetText(result, '/api/architecture/change-sets/proposal', { proposal_markdown: proposal })
  }

  async function updateChangeSetText(result: ArchitectureResult, endpoint: string, value: { name: string } | { proposal_markdown: string }) {
    const changes = result.changes
    if (!changes) return
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON(endpoint, {
        project_slug: result.project_slug,
        store_id: result.store_id,
        change_set_id: changes.id,
        generation: changes.generation,
        ...value,
      })
      const payload = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setArchitectureNotice("WorkBraid couldn't save that proposal change. Inspect it and try again.")
        return
      }
      setChangeSetTextDirty(false)
      enterWorkspace(payload, 'changes', changes.id)
    } catch {
      setArchitectureNotice("WorkBraid couldn't save that proposal change. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  const busy = state.kind === 'looking'

  function requestNavigation(intent: NavigationIntent) {
    if (editorDirty || diagramEditorDirty || changeSetTextDirty || positionDirty || sizeDirty || noteDirty || routeDirty || reviewCommentDirty || newChangeSetNameDirty || reconciliationDirty) {
      setNavigationIntent(intent)
      return
    }
    void performNavigation(intent)
  }

  function requestReviewContextReplacement(apply: () => void) {
    const intent: NavigationIntent = { kind: 'review-context-replacement', apply }
    if (reviewCommentDirty) {
      setNavigationIntent(intent)
      return
    }
    void performNavigation(intent)
  }

  async function performNavigation(intent: NavigationIntent) {
	setNoteSelection(null);setNoteDirty(false)
    setNavigationIntent(null)
    setSelectedRouteKey(undefined)
    setRouteDirty(false)
    setReconciliation(null)
    setReconciliationDirty(false)
    if (intent.kind === 'review-context-replacement') {
      setReviewCommentTarget(undefined)
      setReviewCommentDirty(false)
      intent.apply()
      return
    }
    if (positionDirty || sizeDirty || routeDirty) {
      setSizeDirty(false)
      setPositionDirty(false)
      setPositionDraftEpoch(epoch => epoch + 1)
    }
    if (intent.kind === 'continue-editing') {
      if (state.kind === 'ready') {
        setLocalReviewComments([])
        setReviewCommentTarget(undefined)
        setReviewCommentDirty(false)
        leaveReviewRoute(state.value, true)
      }
      return
    }
    if (intent.kind === 'new-changes') {
      if (editorDirtyRef.current) {
        setEditor(null)
        setDiagramEditor(null)
        setChangeSetTextDirty(false)
      }
      setArchitectureNotice('')
      if (state.kind === 'ready') {
        const projectPath = projectRoutePath(state.value.project_slug)
        if (!creatingChangeSet) newChangesPushedHistoryRef.current = window.location.pathname !== projectPath
        if (window.location.pathname !== projectPath) window.history.pushState({}, '', projectPath)
        enterWorkspace({ ...state.value, submitted_review: undefined, changes: undefined, action_change_set_id: undefined }, state.value.components.length ? 'documentation' : 'empty', 'accepted')
        setReviewVisible(false)
      }
      setCreatingChangeSet(true)
      setNewChangeSetName('')
      resetWorkingPaneScroll()
      return
    }
    setEditor(null)
    setDiagramEditor(null)
    setReviewCommentTarget(undefined)
    setReviewCommentDirty(false)
    setCreatingChangeSet(false)
    setNewChangeSetName('')
    setAuthoringError('')
    setArchitectureNotice('')
    if (intent.kind === 'authoring-pane') {
      setChangeSetTextDirty(false)
      intent.apply()
      return
    }
    if (intent.kind === 'reconcile') {
      if (state.kind !== 'ready' || !state.value.changes) return
      const changes = state.value.changes
      setArchitectureBusy(true)
      try {
        const response = await postJSON('/api/agent/v2/change-sets/reconcile-preview', {
          store_id: state.value.store_id, change_set_id: changes.id, generation: changes.generation,
          change_set_state: changes.change_set_state, base_revision: changes.base_revision,
          candidate_tree: changes.candidate_tree, accepted_revision: state.value.revision,
        })
        const payload = await response.json()
        if (stateRef.current !== state) return
        if (!payload.ok) { setArchitectureNotice(payload.error?.message ?? 'Reconciliation could not be prepared. Refresh and inspect this proposal.'); return }
        setReviewVisible(false)
        setReconciliation(payload.result as ReconciliationPreview)
        resetWorkingPaneScroll()
      } catch { setArchitectureNotice('Reconciliation could not be prepared. Refresh and inspect this proposal.') }
      finally { setArchitectureBusy(false) }
      return
    }
    if (intent.kind === 'component') {
      setSelectedComponentID(intent.id)
      setWorkspaceTask('documentation')
      return
    }
    if (intent.kind === 'diagram') {
      const result = state.kind === 'ready' ? state.value : undefined
      const projection = result?.changes ? result.changes.candidate : result
      const diagram = projection?.diagrams?.find((candidate) => candidate.id === intent.id)
      setSelectedDiagramID(intent.id)
      setSelectedComponentID(intent.focusComponentID && diagram?.appearances.some((appearance) => appearance.component_id === intent.focusComponentID)
        ? intent.focusComponentID
        : diagram?.appearances[0]?.component_id)
      setWorkspaceTask(diagram?.appearances.length ? 'documentation' : 'empty')
      return
    }
    if (intent.kind === 'changes') {
      if (currentReview) setReviewVisible(false)
      setWorkspaceTask('changes')
      resetWorkingPaneScroll()
      return
    }
    if (intent.kind === 'context') {
      if (state.kind !== 'ready') return
      const selected = intent.id === 'accepted' ? undefined : state.value.change_sets.find((changeSet) => changeSet.id === intent.id)
      if (selected) {
        enterProposalRoute(state.value, selected.id, 'push')
        return
      }
      if (window.location.pathname !== projectRoutePath(state.value.project_slug)) window.history.pushState({}, '', projectRoutePath(state.value.project_slug))
      enterWorkspace({ ...state.value, submitted_review: undefined, changes: selected, action_change_set_id: undefined }, selected ? 'changes' : state.value.components.length ? 'documentation' : 'empty', intent.id)
      setReviewVisible(false)
      setDiscardConfirming(false)
      setChangeSetTextDirty(false)
      setCreatingChangeSet(false)
      resetWorkingPaneScroll()
      return
    }
    if (intent.kind === 'clear') {
      setSelectedComponentID(undefined)
      setReviewFocus(null)
      setWorkspaceTask('empty')
      return
    }
    if (intent.kind === 'add') {
      addComponent(selectedDiagramID)
      return
    }
    if (intent.kind === 'change-parent') {
      if (state.kind !== 'ready') return
      const result = state.value
      try {
        const response = await postJSON('/api/architecture/diagrams/parent-options', {
          project_slug: result.project_slug, store_id: result.store_id,
          expected_revision: result.revision, change_set_id: result.changes?.id,
          pending_generation_observed: true, expected_pending_generation: result.changes?.generation ?? null,
          diagram_id: intent.id,
        })
        const options = await response.json() as ParentOptions
        if (stateRef.current !== state) return
        if (!response.ok) { setArchitectureNotice('Parent choices could not be read. Inspect the proposal and try again.'); return }
        setDiagramEditor({ kind: 'parent', diagramID: intent.id, options, anchorID: '' })
      } catch { setArchitectureNotice('Parent choices could not be read. Try again.') }
      return
    }
    if (intent.kind === 'edit-diagram-title') {
      setDiagramEditor({ kind: 'title', diagramID: intent.id, title: intent.title, initialTitle: intent.title })
      return
    }
    if (intent.kind === 'review-result') {
      setAcceptanceUnknown(false)
      if (intent.result.changes?.review) enterReviewRoute(intent.result, intent.result.changes.id, 'push')
      return
    }
    if (intent.kind === 'submitted-review') {
      if (state.kind === 'ready') await openSubmittedReview(state.value, intent.changeSetID, intent.reviewID, 'push')
      return
    }
    if (intent.kind === 'route') {
      editorDirtyRef.current = false
      setChangeSetTextDirty(false)
      window.history.back()
      return
    }
    if (state.kind !== 'ready') return
    if (intent.kind === 'refresh') {
      await refreshArchitecture(state.value)
      return
    }
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/projects/leave', { project_slug: state.value.project_slug, store_id: state.value.store_id })
      if (response.ok) {
        window.history.pushState({}, '', '/')
        setSelectedComponentID(undefined)
        setSelectedDiagramID(undefined)
        setWorkspaceTask('empty')
        await loadCatalog()
        return
      }
      setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
    } catch {
      setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function restoreRoute(slug?: string, proposalChangeSetID?: string, reviewChangeSetID?: string, submittedReviewID?: string) {
    const current = stateRef.current
    if (slug) {
      if (current.kind === 'ready' && current.value.project_slug === slug) {
        if (reviewChangeSetID && submittedReviewID) await openSubmittedReview(current.value, reviewChangeSetID, submittedReviewID, 'replace')
        else if (reviewChangeSetID) enterReviewRoute(current.value, reviewChangeSetID, 'replace')
        else if (proposalChangeSetID) enterProposalRoute(current.value, proposalChangeSetID, 'replace')
        else {
          setEditor(null)
          setDiagramEditor(null)
          setCreatingChangeSet(false)
          setNewChangeSetName('')
          setDiscardConfirming(false)
          setChangeSetTextDirty(false)
          setReviewVisible(false)
          setArchitectureNotice('')
          enterWorkspace({ ...current.value, submitted_review: undefined, changes: undefined, action_change_set_id: undefined }, current.value.components.length ? 'documentation' : 'empty', 'accepted')
          window.history.replaceState({}, '', projectRoutePath(current.value.project_slug))
          resetWorkingPaneScroll()
        }
      } else {
        await openProject(slug, true, proposalChangeSetID, reviewChangeSetID, submittedReviewID)
      }
      return
    }
    if (current.kind !== 'ready') {
      window.history.replaceState({}, '', '/')
      await loadCatalog()
      return
    }
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/projects/leave', { project_slug: current.value.project_slug, store_id: current.value.store_id })
      if (response.ok) {
        window.history.replaceState({}, '', '/')
        setSelectedComponentID(undefined)
        setSelectedDiagramID(undefined)
        setWorkspaceTask('empty')
        await loadCatalog()
        return
      }
      window.history.replaceState({}, '', `/projects/${encodeURIComponent(current.value.project_slug)}`)
      setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
    } catch {
      window.history.replaceState({}, '', `/projects/${encodeURIComponent(current.value.project_slug)}`)
      setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function discardChanges(result: ArchitectureResult) {
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/architecture/discard', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        ...(result.changes ? { change_set_id: result.changes.id } : {}),
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setArchitectureNotice("WorkBraid couldn't discard these changes. Try again.")
        return
      }
      setDiscardConfirming(false)
      setReviewVisible(false)
      window.history.replaceState({}, '', projectRoutePath(payload.project_slug))
      enterWorkspace(payload, payload.components?.length ? 'documentation' : 'empty')
    } catch {
      setArchitectureNotice("WorkBraid couldn't discard these changes. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function refreshArchitecture(result: ArchitectureResult) {
    setArchitectureBusy(true)
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/architecture/refresh', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        expected_revision: result.revision,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if (!('state' in payload)) {
        setArchitectureNotice(messageForArchitectureAction('code' in payload ? payload.code : undefined))
        return
      }
      const notice = payload.action_error ? messageForArchitectureAction(payload.action_error) : ''
      const nextTask = payload.changes?.stale ? 'changes' : workspaceTask
      const route = decodeProjectRoute(window.location.pathname)
      if (route?.reviewChangeSetID && route.submittedReviewID) {
        await openSubmittedReview(payload, route.reviewChangeSetID, route.submittedReviewID, 'replace')
      } else if (route?.reviewChangeSetID) {
        enterReviewRoute(payload, route.reviewChangeSetID, 'replace')
        if (notice) setArchitectureNotice(notice)
      } else if (route?.proposalChangeSetID) {
        enterProposalRoute(payload, route.proposalChangeSetID, 'replace')
        if (notice) setArchitectureNotice(notice)
      } else {
        if (payload.project_slug !== result.project_slug) window.history.replaceState({}, '', projectRoutePath(payload.project_slug))
        enterWorkspace(payload, nextTask)
        setArchitectureNotice(notice)
      }
    } catch {
      setState((current) => current.kind === 'ready' && current.value.store_id === result.store_id
        ? { kind: 'ready', value: { ...current.value, action_error: 'refresh_failed' } }
        : current)
      setArchitectureNotice(messageForArchitectureAction('refresh_failed'))
    } finally {
      setArchitectureBusy(false)
    }
  }

  if (state.kind === 'ready') {
    const result = state.value
    const review = reviewVisible ? currentReview : undefined
    const activeProjection = review ? (reviewSide === 'with' ? review.with_changes : review.before) : undefined
    const invalidProposalProjection: ReviewSnapshot = {
      revision: result.changes?.base_revision ?? result.revision,
      format_version: result.format_version,
      component_count: 0,
      component_titles: [],
      components: [],
      diagrams: [],
    }
    const diagramProjection = activeProjection ?? (result.changes ? result.changes.candidate ?? invalidProposalProjection : result)
    const candidateOnlyDiagramBefore = Boolean(review && reviewSide === 'before' && selectedDiagramID &&
      review.with_changes.diagrams?.some((diagram) => diagram.id === selectedDiagramID) &&
      !review.before.diagrams?.some((diagram) => diagram.id === selectedDiagramID))
    const candidateFallbackDiagramID = candidateOnlyDiagramBefore && review
      ? [...(review.with_changes.diagrams?.find((diagram) => diagram.id === selectedDiagramID)?.breadcrumbs ?? [])]
        .reverse()
        .find((breadcrumb) => review.before.diagrams?.some((diagram) => diagram.id === breadcrumb.id))?.id
      : undefined
    const activeDiagram = (diagramProjection.format_version ?? 0) >= 2
      ? diagramProjection.diagrams?.find((diagram) => diagram.id === (candidateFallbackDiagramID ?? selectedDiagramID))
        ?? diagramProjection.diagrams?.find((diagram) => diagram.id === diagramProjection.root_diagram_id)
      : undefined
    const focusReviewDiagram = (focus: Extract<ReviewFocus, { kind: 'diagram' }>) => requestReviewContextReplacement(() => {
      setReviewSelectionCleared(false)
      if (focus.reviewSide && focus.reviewSide !== reviewSide) setReviewSide(focus.reviewSide)
      setSelectedDiagramID(focus.diagramID)
      const focusProjection = focus.reviewSide === 'before' ? review?.before : review?.with_changes
      const focusDiagram = focusProjection?.diagrams?.find((diagram) => diagram.id === focus.diagramID)
      setSelectedComponentID(focus.componentID && focusDiagram?.appearances.some((appearance) => appearance.component_id === focus.componentID)
        ? focus.componentID
        : undefined)
      setReviewFocus(focus)
    })
    const mapComposition = new Map<string, ReturnType<typeof appearanceReviewPresentation>>()
    for (const [index, appearance] of (review?.comparison.appearances ?? []).entries()) {
      if (!review || appearance.diagram_id !== activeDiagram?.id || (reviewSide === 'before' && appearance.side !== 'before')) continue
      const presentation = appearanceReviewPresentation(review, appearance, index)
      // The two parent-owned rows describe one child movement. Prefer the
      // visible side's row; a removed link can focus its exact Before context.
      const identity = presentation.movedChild ? `detail:${appearance.detail_diagram_id}` : presentation.focus.key
      if (!mapComposition.has(identity) || appearance.side === reviewSideValue(reviewSide)) mapComposition.set(identity, presentation)
    }
    const authorityIndeterminate = result.action_error === 'refresh_failed'
    const authoringAvailable = !reconciliation && !result.stale && !authorityIndeterminate && !result.changes?.stale && !result.changes?.read_only && !acceptanceUnknown
    const compositionProjection = result.changes?.candidate ?? result
    const compositionDiagrams = result.changes?.diagram_options ?? compositionProjection.diagrams ?? result.diagrams ?? []
    const editorDiagrams = diagramEditor?.kind === 'move'
      ? compositionDiagrams.filter((diagram) => result.home_move_destinations
        ?.find((destinations) => destinations.component_id === diagramEditor.componentID)
        ?.diagram_ids.includes(diagram.id))
      : compositionDiagrams
    const beginHomeMove = (componentID: string) => {
      const destinations = result.home_move_destinations?.find((candidate) => candidate.component_id === componentID)
      if (!destinations?.diagram_ids.length) return
      const component = compositionProjection.components.find((candidate) => candidate.id === componentID)
      const currentDiagram = compositionDiagrams.find((diagram) => diagram.id === destinations.current_home_id)
      if (!component || !currentDiagram) return
      setDiagramEditor({
        kind: 'move', componentID, componentTitle: componentAuthoringLabel(compositionProjection.components, componentID), diagramID: '',
        currentDiagramTitle: diagramAuthoringOptionLabel(currentDiagram),
      })
    }
    const activeDiagramComponents = activeDiagram ? componentsForDiagram(diagramProjection, activeDiagram) : undefined
    const activeComponents = (activeProjection?.format_version ?? 0) >= 2
      ? activeDiagramComponents ?? []
      : activeProjection?.components ?? activeDiagramComponents ?? diagramProjection.components ?? []
    const diagramMapComponents = activeDiagram ? mapComponentsForDiagram(diagramProjection, activeDiagram) : undefined
    const mapComponents: MapComponent[] = diagramMapComponents ?? activeComponents
    const otherReviewSnapshot = review ? (reviewSide === 'with' ? review.before : review.with_changes) : undefined
    const otherReviewDiagram = otherReviewSnapshot?.diagrams?.find(diagram => diagram.id === activeDiagram?.id)
    const reviewOtherComponents = otherReviewSnapshot ? (otherReviewDiagram ? mapComponentsForDiagram(otherReviewSnapshot, otherReviewDiagram) : activeDiagram ? [] : otherReviewSnapshot.components) : undefined
    const selectedRoute = !review && !editor && !diagramEditor && workspaceTask === 'documentation' ? activeDiagram?.relationships.find(r=>r.key===selectedRouteKey) : undefined
    const selectedBoundary = activeDiagram?.boundaries.find(b=>b.component_id===selectedComponentID)
    const selected = activeComponents.find((component) => component.id === selectedComponentID) ?? (selectedBoundary ? diagramProjection.components.find(c=>c.id===selectedComponentID) : undefined)
    const selectedAppearance = activeDiagram?.appearances.find((appearance) => appearance.component_id === selectedComponentID)
    const selectedPosition = selectedAppearance?.display_position ?? selectedAppearance?.position ?? selectedBoundary?.display_position ?? selectedBoundary?.position ?? null
    const selectedSize = selectedAppearance?.display_size ?? selectedBoundary?.display_size ?? (selectedBoundary ? {width:104,height:62}:{width:116,height:54})
    const diagramNodeTitles = new Map<string, string>()
    for (const component of activeDiagramComponents ?? []) diagramNodeTitles.set(component.id, component.title)
    for (const boundary of activeDiagram?.boundaries ?? []) diagramNodeTitles.set(boundary.key, boundary.title)
    const titleCounts = new Map<string, number>()
    for (const component of activeComponents) titleCounts.set(component.title, (titleCounts.get(component.title) ?? 0) + 1)
    const homeDiagramTitles = new Map<string, string>()
    for (const diagram of diagramProjection.diagrams ?? []) {
      for (const appearance of diagram.appearances) {
        if (appearance.role === 'home') homeDiagramTitles.set(appearance.component_id, diagram.title)
      }
    }
    const componentReviewStatus = new Map(review?.comparison.components.map((change) => [change.component_id, change]))
    const diagramReviewStatus = new Map<string, 'Added' | 'Title changed' | 'Changed'>()
    for (const change of review?.comparison.diagrams ?? []) {
      diagramReviewStatus.set(change.diagram_id, change.status === 'added' ? 'Added' : 'Title changed')
    }
    for (const change of review?.comparison.appearances ?? []) {
      if (!diagramReviewStatus.has(change.diagram_id)) diagramReviewStatus.set(change.diagram_id, 'Changed')
    }
	for(const change of review?.comparison.node_positions??[]) {if(!diagramReviewStatus.has(change.diagram_id))diagramReviewStatus.set(change.diagram_id,'Changed')}
	for(const change of [...(review?.comparison.node_sizes??[]),...(review?.comparison.node_shapes??[]),...(review?.comparison.diagram_notes??[])]) {if(!diagramReviewStatus.has(change.diagram_id))diagramReviewStatus.set(change.diagram_id,'Changed')}
	for(const change of review?.comparison.edge_routes??[]) {if(!diagramReviewStatus.has(change.diagram_id))diagramReviewStatus.set(change.diagram_id,'Changed')}
    const submittedReview = result.submitted_review
    const reviewAnnotationComments = submittedReview?.comments ?? localReviewComments
    const reviewPresentation = submittedReview ?? (review && result.changes
      ? { review, proposal_markdown: result.changes.proposal_markdown }
      : undefined)
    const annotationSide = reviewSideValue(reviewSide)
    const commentedNavigationItems = new Map<string, {
      diagram: DiagramProjection
      component: AuthoringComponent
      comments: ReviewSubmissionComment[]
    }>()
    if (review && (diagramProjection.format_version ?? 0) >= 2) {
      for (const comment of reviewAnnotationComments) {
        if (comment.anchor.side !== annotationSide ||
          (comment.anchor.kind !== 'component' && comment.anchor.kind !== 'component_markdown' && comment.anchor.kind !== 'composition')) continue
        const componentID = comment.anchor.component_id
        const component = diagramProjection.components.find((candidate) => candidate.id === componentID)
        const diagram = comment.anchor.kind === 'composition'
          ? diagramProjection.diagrams?.find((candidate) => candidate.id === comment.anchor.diagram_id)
          : diagramProjection.diagrams?.find((candidate) => candidate.appearances.some((appearance) =>
            appearance.component_id === componentID && appearance.role === 'home'))
        if (!component || !diagram) continue
        const key = `${diagram.id}:${component.id}`
        const existing = commentedNavigationItems.get(key)
        if (existing) existing.comments.push(comment)
        else commentedNavigationItems.set(key, { diagram, component, comments: [comment] })
      }
    }
    const commentsOutsideActiveIndex = [...commentedNavigationItems.values()].filter((item) =>
      item.diagram.id !== activeDiagram?.id || !activeComponents.some((component) => component.id === item.component.id))
    const mapNodeAnnotationGroups: Record<string, ReviewAnnotationGroup> = {}
    for (const component of mapComponents) {
      const componentID = component.component_id ?? component.id
      const componentComments = componentAnnotation(reviewAnnotationComments, annotationSide, componentID, component.title)?.comments ?? []
      const placementComments = activeDiagram?.appearances.some((appearance) => appearance.component_id === componentID)
        ? compositionAnnotation(reviewAnnotationComments, annotationSide, activeDiagram.id, componentID, `${component.title} placement`)?.comments ?? []
        : []
      const group = annotationGroup(`map-item:${annotationSide}:${activeDiagram?.id ?? ''}:${componentID}`, component.title,
        [...componentComments, ...placementComments], annotationSide)
      if (group) mapNodeAnnotationGroups[component.id] = { ...group, mapTarget: { kind: 'node', id: component.id, diagramID: activeDiagram?.id } }
    }
    const relationshipAnnotationGroups: Record<string, ReviewAnnotationGroup> = {}
    if (activeDiagram) {
      const occurrences = new Map<string, number>()
      for (const relationship of activeDiagram.relationships) {
        const fact = `${relationship.source_component_id}\u0000${relationship.target_component_id}\u0000${relationship.label}`
        const occurrence = (occurrences.get(fact) ?? 0) + 1
        occurrences.set(fact, occurrence)
        const sourceTitle = diagramNodeTitles.get(relationship.source_node_key) ?? 'Component'
        const targetTitle = diagramNodeTitles.get(relationship.target_node_key) ?? 'Component'
        const group = relationshipAnnotation(reviewAnnotationComments, annotationSide, relationship.source_component_id, relationship.target_component_id,
          relationship.label, occurrence, `${sourceTitle} — ${relationship.label} → ${targetTitle}`)
        if (group) relationshipAnnotationGroups[relationship.key] = { ...group, mapTarget: { kind: 'relationship', id: relationship.key, diagramID: activeDiagram.id } }
      }
    }
    const layoutComponentIDs = review
      ? [...new Set([...review.before.components, ...review.with_changes.components].map((component) => component.id))]
      : undefined
    const selectComponent = (id: string) => {
      if (!review) {
        requestNavigation({ kind: 'component', id })
        return
      }
      requestReviewContextReplacement(() => {
        const component = diagramProjection.components.find((candidate) => candidate.id === id)
        if (!component) return
        const change = componentReviewStatus.get(id)
        setReviewSelectionCleared(false)
        setSelectedComponentID(id)
        setReviewFocus({
          kind: 'component', key: `component:${id}`, componentID: id, title: component.title,
          path: change?.path ?? canonicalReviewPath(component), status: change?.status ?? 'unchanged',
        })
      })
    }
    const selectDiagram = (diagramID: string, focusComponentID?: string) => {
      if (!review) {
        requestNavigation({ kind: 'diagram', id: diagramID, focusComponentID })
        return
      }
      requestReviewContextReplacement(() => {
        const diagram = diagramProjection.diagrams?.find((candidate) => candidate.id === diagramID)
        if (!diagram) return
        setReviewSelectionCleared(false)
        setSelectedDiagramID(diagram.id)
        setSelectedComponentID(focusComponentID && diagram.appearances.some((appearance) => appearance.component_id === focusComponentID)
          ? focusComponentID
          : diagram.appearances[0]?.component_id)
        setWorkspaceTask(diagram.appearances.length ? 'documentation' : 'empty')
        setEditor(null)
        setReviewFocus(null)
      })
    }
    const selectMapNode = (id: string) => {
	  if(id.startsWith('note:')&&activeDiagram){requestNavigation({kind:'authoring-pane',apply:()=>{setNoteSelection({diagramID:activeDiagram.id,id:id.slice(5)});setSelectedComponentID(undefined);setWorkspaceTask('documentation')}});return}
      if (!activeDiagram) {
        selectComponent(id)
        return
      }
      const boundary = activeDiagram.boundaries.find((candidate) => candidate.key === id)
      if (boundary) {
        selectComponent(boundary.component_id)
        return
      }
      selectComponent(id)
    }
    const selectRelationship = (relationship: ReviewRelationshipSelection) => {
      requestReviewContextReplacement(() => {
        const relationshipSide = relationship.review_side ?? reviewSide
        const relationshipProjection = relationshipSide === 'with' ? review?.with_changes : review?.before
        const relationshipDiagram = relationshipProjection && (relationshipProjection.format_version ?? 0) >= 2
          ? relationshipProjection.diagrams?.find((diagram) => diagram.id === selectedDiagramID)
            ?? relationshipProjection.diagrams?.find((diagram) => diagram.id === relationshipProjection.root_diagram_id)
          : undefined
        const relationshipComponents = relationshipProjection && relationshipDiagram
          ? componentsForDiagram(relationshipProjection, relationshipDiagram)
          : relationshipProjection?.components ?? activeComponents
        setReviewSelectionCleared(false)
        if (relationship.review_side && relationship.review_side !== reviewSide) setReviewSide(relationship.review_side)
        setSelectedComponentID(relationshipComponents.some((component) => component.id === relationship.source_id)
          ? relationship.source_id
          : undefined)
        setReviewFocus({ kind: 'relationship', ...relationship })
      })
    }
    const switchReviewSide = (side: ReviewSide) => {
      if (!review || side === reviewSide) return
      requestReviewContextReplacement(() => {
        const nextProjection = side === 'with' ? review.with_changes : review.before
        const nextDiagram = (nextProjection.format_version ?? 0) >= 2
          ? nextProjection.diagrams?.find((diagram) => diagram.id === selectedDiagramID)
            ?? nextProjection.diagrams?.find((diagram) => diagram.id === nextProjection.root_diagram_id)
          : undefined
        const nextComponents = nextDiagram ? componentsForDiagram(nextProjection, nextDiagram) : nextProjection.components
        setReviewSelectionCleared(false)
        setReviewSide(side)
        setReviewFocus((current) => current?.kind === 'diagram' && current.componentID && current.reviewSide &&
          nextProjection.diagrams?.some((diagram) => diagram.id === current.diagramID) ? current : null)
        setSelectedComponentID((current) => current && nextComponents.some((component) => component.id === current) ? current : undefined)
      })
    }
    const openReviewAnnotation = (group: ReviewAnnotationGroup) => {
      const first = group.comments[0]?.anchor
      const exactProjection = first?.side === 'before' ? review?.before : review?.with_changes
      const exactDiagram = exactProjection?.diagrams?.find(diagram => diagram.id === (first?.diagram_id ?? activeDiagram?.id))
      if (!group.mapTarget && first?.component_id && first.kind !== 'diagram') {
        const nodeID = exactDiagram?.appearances.some(appearance => appearance.component_id === first.component_id)
          ? first.component_id : exactDiagram?.boundaries.find(boundary => boundary.component_id === first.component_id)?.key
        if (nodeID) group = { ...group, mapTarget: { kind: 'node', id: nodeID, diagramID: exactDiagram?.id } }
      } else if (!group.mapTarget && first?.kind === 'relationship') {
        const fact = exactDiagram?.relationships.filter(relationship => relationship.source_component_id === first.source_component_id &&
          relationship.target_component_id === first.target_component_id && relationship.label === first.label)[(first.occurrence ?? 1) - 1]
        if (fact) group = { ...group, mapTarget: { kind: 'relationship', id: fact.key, diagramID: exactDiagram?.id } }
      }
      if (group.mapTarget) group = { ...group, key: `map:${group.side}:${group.mapTarget.diagramID}:${group.mapTarget.kind}:${group.mapTarget.id}` }
      const targetSide = group.side === 'before' ? 'before' : group.side === 'with_changes' ? 'with' : undefined
      const targetDiagramID = group.mapTarget?.diagramID ?? first?.diagram_id
      const changesDiagram = Boolean(targetDiagramID && targetDiagramID !== activeDiagram?.id)
      const changesSide = Boolean(targetSide && targetSide !== reviewSide)
      const open = () => {
        if (targetSide) setReviewSide(targetSide)
        if (changesDiagram || changesSide) {
          if (targetDiagramID) setSelectedDiagramID(targetDiagramID)
          setSelectedComponentID(first?.component_id)
          setReviewFocus(null)
        }
        if (changesDiagram || changesSide) setOpenReviewAnnotations(current => ({ ...current, [group.key]: group }))
        else toggleReviewAnnotation(group)
      }
      if (changesDiagram || changesSide) requestReviewContextReplacement(open)
      else open()
    }
    const visibleAnnotationCards = Object.values(openReviewAnnotations).map(group => {
      const anchor = group.comments[0]?.anchor
      const comments = reviewAnnotationComments.filter(comment => {
        const other = comment.anchor
        if (!anchor || other.side !== anchor.side) return false
        if (group.mapTarget?.kind === 'node') return other.component_id === anchor.component_id &&
          (other.kind === 'component' || other.kind === 'component_markdown' ||
            (other.kind === 'composition' && other.diagram_id === group.mapTarget.diagramID))
        if (anchor.kind === 'relationship') return other.kind === 'relationship' &&
          other.source_component_id === anchor.source_component_id && other.target_component_id === anchor.target_component_id &&
          other.label === anchor.label && other.occurrence === anchor.occurrence
        return other.kind === anchor.kind && other.diagram_id === anchor.diagram_id && other.component_id === anchor.component_id
      })
      return { ...group, comments }
    }).filter(group => (!group.side || group.side === annotationSide) && group.comments.length > 0)
    const closeReviewAnnotation = (key: string) => setOpenReviewAnnotations((current) => {
      const next = { ...current }
      delete next[key]
      return next
    })
    const saveReviewComment = (body: string, anchor: ReviewAnchor) => {
      const existingID = reviewCommentTarget?.localCommentID
      const comment: LocalReviewComment = { id: existingID ?? `local-review-comment-${crypto.randomUUID()}`, body, anchor }
      setLocalReviewComments((current) => existingID
        ? current.map((item) => item.id === existingID ? comment : item)
        : [...current, comment])
      const mapTarget = reviewCommentTarget?.mapTarget
      const key = mapTarget
        ? `map:${anchor.side}:${mapTarget.diagramID}:${mapTarget.kind}:${mapTarget.id}`
        : `diagram:${anchor.side}:${anchor.diagram_id ?? ''}`
      if (mapTarget || anchor.kind === 'diagram') setOpenReviewAnnotations((current) => ({
        ...current,
        [key]: {
          key,
          label: reviewCommentTarget?.label ?? 'Review comment',
          side: anchor.side,
          comments: [...(current[key]?.comments ?? []).filter(item => item.id !== comment.id), comment],
          mapTarget,
        },
      }))
      setReviewCommentTarget(undefined)
      setReviewCommentDirty(false)
    }
    const removeLocalMapComment = (commentID: string) => {
      if (reviewCommentTarget?.localCommentID === commentID) {
        setReviewCommentTarget(undefined)
        setReviewCommentDirty(false)
      }
      setLocalReviewComments((current) => current.filter((comment) => comment.id !== commentID))
    }
    const editLocalMapComment = (comment: LocalReviewComment, group: ReviewAnnotationGroup) => {
      requestReviewContextReplacement(() => openCommentEditor({
        contextKey: `${group.mapTarget ? 'map-edit' : 'diagram-map'}:${comment.id}`,
        label: group.label,
        anchor: comment.anchor,
        mapTarget: group.mapTarget,
        localCommentID: comment.id,
        initialBody: comment.body,
        source: comment.anchor.kind === 'component_markdown'
          ? (comment.anchor.side === 'before' ? review?.before : review?.with_changes)?.components.find(component => component.id === comment.anchor.component_id)?.markdown_source
          : undefined,
      }))
    }
    const externalReferences = activeDiagram && activeDiagram.boundaries.length > 0 ? (
      <nav className="diagram-boundary-dock" aria-label="Components that live elsewhere">
        {activeDiagram.boundaries.map((boundary) => (
          <section key={boundary.key}>
            <button type="button" aria-label={[boundary.title, boundary.context, `Lives in ${boundary.home_diagram_title}`].filter(Boolean).join(', ')} onClick={() => selectMapNode(boundary.key)}>
              <span>{boundary.title}{boundary.context && <small> {boundary.context}</small>}</span>
              <small className="home-location-note">Lives in {boundary.home_diagram_title}</small>
            </button>
            <ul aria-label={`Relationships for ${[boundary.title, boundary.context].filter(Boolean).join(', ')}`}>
              {activeDiagram.relationships.filter((relationship) => relationship.source_node_key === boundary.key || relationship.target_node_key === boundary.key).map((relationship) => (
                <li key={relationship.key} tabIndex={0}>
                  <span>{diagramNodeTitles.get(relationship.source_node_key)}</span>
                  <strong>{relationship.label}</strong>
                  <span aria-hidden="true">→</span>
                  <span>{diagramNodeTitles.get(relationship.target_node_key)}</span>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </nav>
    ) : undefined
    const currentProposalRecord = submittedReview ? result.change_sets.find((changeSet) => changeSet.id === submittedReview.change_set_id) : undefined
    const mapAnnotationCards = reviewPresentation ? visibleAnnotationCards.filter((group) => group.mapTarget && group.mapTarget.diagramID === activeDiagram?.id) : []
    const diagramAnnotationCards = reviewPresentation ? visibleAnnotationCards.filter((group) => !group.mapTarget && group.comments.some(comment => comment.anchor.diagram_id === activeDiagram?.id)) : []
    const activeDiagramAnnotations = activeDiagram
      ? diagramAnnotation(reviewAnnotationComments, annotationSide, activeDiagram.id, activeDiagram.title)
      : undefined
    const openDiagramComment = () => {
      if (!activeDiagram || submittedReview) return
      requestReviewContextReplacement(() => openCommentEditor({
        contextKey: `diagram-map:${annotationSide}:${activeDiagram.id}`,
        label: activeDiagram.title,
        anchor: { kind: 'diagram', side: annotationSide, diagram_id: activeDiagram.id },
      }))
    }
    const requestComment = (target: ReviewCommentTarget) => requestReviewContextReplacement(() => openCommentEditor(target))
    const cancelComment = () => { setReviewCommentTarget(undefined); setReviewCommentDirty(false) }
    const annotationOverlay = reviewPresentation ? (
      <>
        {mapAnnotationCards.length > 0 && <ReviewAnnotationCards
          groups={mapAnnotationCards}
          review={reviewPresentation}
          onClose={closeReviewAnnotation}
          onEditLocal={submittedReview ? undefined : editLocalMapComment}
          onRemoveLocal={submittedReview ? undefined : removeLocalMapComment}
          onAdd={submittedReview ? undefined : group => {
            const first = group.comments[0]?.anchor
            if (!first) return
            requestComment({ contextKey: `map-add:${group.key}`, label: group.label, mapTarget: group.mapTarget,
              anchor: group.mapTarget?.kind === 'node' ? { kind: 'component', side: first.side, component_id: first.component_id } : first })
          }}
        />}
        {!submittedReview && reviewCommentTarget?.mapTarget && <InlineReviewCommentEditor
          key={reviewCommentTarget.editorKey}
          target={reviewCommentTarget}
          onDirty={setReviewCommentDirty}
          onSave={saveReviewComment}
          onCancel={cancelComment}
          mapPopover
        />}
      </>
    ) : undefined
    return (
      <main className="workspace-shell">
        <header className="application-frame">
          <div>
            <p className="eyebrow">WorkBraid</p>
            <p className="workspace-context"><strong>{result.project_name}</strong><span>Architecture</span></p>
          </div>
          <div className="frame-actions">
            {result.submitted_review
              ? <span className="submitted-review-context">Submitted review</span>
              : <ShowingMenu changeSets={result.change_sets} selectedID={selectedContextID} onSelect={(id) => requestNavigation({ kind: 'context', id })} />}
            <button className="text-action" type="button" disabled={architectureBusy || acceptanceUnknown || result.stale || authorityIndeterminate} onClick={() => requestNavigation({ kind: 'new-changes' })}>New changes</button>
            <button className="text-action" type="button" disabled={architectureBusy || acceptanceUnknown} onClick={() => requestNavigation({ kind: 'refresh' })}>
              Refresh
            </button>
            <button className="text-action" type="button" disabled={architectureBusy || acceptanceUnknown} onClick={() => requestNavigation({ kind: 'open-another' })}>
              Open another project
            </button>
          </div>
        </header>
        {result.unavailable_change_sets?.length ? <div className="stale-banner" role="alert">{result.unavailable_change_sets.length} proposal{result.unavailable_change_sets.length === 1 ? '' : 's'} unavailable. Accepted and other proposals remain available.</div> : null}
        {result.stale && <div className="stale-banner" role="alert">The current architecture could not be loaded. This earlier view is read-only.</div>}
        {architectureNotice && (
          <div className="workspace-notice" role="alert">
            <span>{architectureNotice}</span>
            <button className="notice-dismiss" type="button" aria-label="Dismiss message" onClick={() => setArchitectureNotice('')}>×</button>
          </div>
        )}
        <div className={`architecture-workbench ${review ? 'reviewing' : ''} ${reconciliation ? 'reconciling' : ''}`}>
          <nav className="component-index" aria-label={(diagramProjection.format_version ?? 0) >= 2 ? 'Diagrams and components' : 'Components'}>
            {(diagramProjection.format_version ?? 0) >= 2 && (
              <div className="diagram-navigator">
                <div className="index-heading">
                  <h1>Diagrams</h1>
                  {!review && authoringAvailable && activeDiagram && (
                    <button className="index-add diagram-title-edit" type="button" onClick={() => requestNavigation({ kind: 'edit-diagram-title', id: activeDiagram.id, title: activeDiagram.title })}>Edit title</button>
                  )}
                </div>
                <ul className="diagram-tree">
                  {diagramProjection.diagrams?.map((diagram) => {
                    const reviewStatus = diagramReviewStatus.get(diagram.id)
                    const reviewClass = reviewStatus === 'Added' ? 'review-added' : reviewStatus ? 'review-content-changed' : ''
                    const comments = diagramAnnotation(reviewAnnotationComments, annotationSide, diagram.id, diagram.title)
                    return (
                      <li key={diagram.id}>
                        <div className="index-entry-row">
                          <button
                            type="button"
                            className={[diagram.id === activeDiagram?.id ? 'selected' : '', reviewClass].filter(Boolean).join(' ') || undefined}
                            style={{ paddingLeft: `${16 + diagram.depth * 18}px` }}
                            aria-label={[diagram.title, diagram.context, reviewStatus].filter(Boolean).join(', ')}
                            aria-current={diagram.id === activeDiagram?.id ? 'page' : undefined}
                            onClick={() => selectDiagram(diagram.id)}
                          >
                            <span>{diagram.title}</span>
                            {diagram.context && <small>{diagram.context}</small>}
                            {reviewStatus && <small className="index-review-status">{reviewStatus}</small>}
                          </button>
                          {comments
                            ? <AnnotationMarker group={comments} onToggle={() => openReviewAnnotation(comments)} />
                            : review && !submittedReview && diagram.id === activeDiagram?.id
                              ? <AnnotationAddMarker label={`Comment on ${diagram.title}`} onClick={openDiagramComment} />
                              : null}
                        </div>
                      </li>
                    )
                  })}
                </ul>
              </div>
            )}
            <div className="index-heading component-heading"><h1>{review ? (reviewSide === 'with' ? 'With changes' : 'Before changes') : 'Components'}</h1></div>
            {activeComponents.length ? (
              <ul>
                {activeComponents.map((component) => {
                  const reviewStatus = componentReviewStatus.get(component.id)?.status ?? (review ? 'unchanged' : '')
                  const statusLabel = reviewStatus === 'added' ? 'Added' : reviewStatus === 'content_changed' ? 'Content changed' : ''
                  const appearance = activeDiagram?.appearances.find((candidate) => candidate.component_id === component.id)
                  const referenceHomeTitle = appearance?.role === 'reference' ? homeDiagramTitles.get(component.id) : undefined
                  const referenceContext = referenceHomeTitle ? `Included here · Lives in ${referenceHomeTitle}` : ''
                  const componentLabel = (titleCounts.get(component.title) ?? 0) > 1 ? `${component.title}, ${component.filename || component.id.slice(0, 8)}` : component.title
                  const componentComments = componentAnnotation(reviewAnnotationComments, annotationSide, component.id, component.title)
                  const placementComments = activeDiagram ? compositionAnnotation(reviewAnnotationComments, annotationSide, activeDiagram.id, component.id, `${component.title} placement`) : undefined
                  const comments = annotationGroup(`item:${annotationSide}:${activeDiagram?.id ?? ''}:${component.id}`, component.title,
                    [...(componentComments?.comments ?? []), ...(placementComments?.comments ?? [])], annotationSide)
                  return (
                  <li key={component.id}>
                    <div className="index-entry-row">
                      <button
                        type="button"
                        className={`${component.id === selectedComponentID && !editor ? 'selected' : ''} ${review ? `review-${reviewStatus.replace('_', '-')}` : ''}`.trim()}
                        aria-label={[componentLabel, referenceContext, statusLabel].filter(Boolean).join(', ')}
                        aria-current={component.id === selectedComponentID && !editor ? 'page' : undefined}
                        onClick={() => selectComponent(component.id)}
                      >
                        <span>{component.title}</span>
                        {(titleCounts.get(component.title) ?? 0) > 1 && <small>{' '}{component.filename || component.id.slice(0, 8)}</small>}
                        {referenceContext && <small className="appearance-note">{referenceContext}</small>}
                        {statusLabel && <small className="index-review-status">{statusLabel}</small>}
                      </button>
                      {comments
                        ? <AnnotationMarker group={comments} onToggle={() => openReviewAnnotation(comments)} />
                        : review && !submittedReview && selectedComponentID === component.id && reviewFocus?.kind !== 'relationship'
                          ? <AnnotationAddMarker label={`Comment on ${component.title}`} onClick={() => requestComment({ contextKey: `map-node:${annotationSide}:${component.id}`, label: component.title,
                            anchor: { kind: 'component', side: annotationSide, component_id: component.id }, mapTarget: { kind: 'node', id: component.id, diagramID: activeDiagram?.id } })} />
                          : null}
                    </div>
                  </li>
                  )
                })}
              </ul>
            ) : <p className="index-empty">No components</p>}
            {commentsOutsideActiveIndex.length > 0 && (
              <div className="commented-elsewhere">
                <div className="index-heading"><h1>Comments elsewhere</h1></div>
                <ul>{commentsOutsideActiveIndex.map(({ diagram, component, comments }) => {
                  const group = annotationGroup(`elsewhere:${annotationSide}:${diagram.id}:${component.id}`, component.title, comments, annotationSide)!
                  return <li key={`${diagram.id}:${component.id}`}>
                    <div className="index-entry-row">
                      <button type="button" aria-label={`${component.title}, in ${diagram.title}`} onClick={() => selectDiagram(diagram.id, component.id)}>
                        <span>{component.title}</span><small>In {diagram.title}</small>
                      </button>
                      <AnnotationMarker group={group} onToggle={() => openReviewAnnotation({ ...group, mapTarget: { kind: 'node', id: component.id, diagramID: diagram.id } })} />
                    </div>
                  </li>
                })}</ul>
              </div>
            )}
            {!review && authoringAvailable && (
              <button className="index-add" type="button" onClick={() => requestNavigation({ kind: 'add' })}>Add component</button>
            )}
			{activeDiagram&&<div className="diagram-notes"><details><summary>Notes{activeDiagram.notes?.length?` · ${activeDiagram.notes.length}`:''}</summary>{(activeDiagram.notes??[]).map(n=><button className="text-action" key={n.id} type="button" onClick={()=>selectMapNode(`note:${n.id}`)}>{n.text.slice(0,80)}</button>)}{!review&&authoringAvailable&&<button className="text-action" type="button" onClick={()=>requestNavigation({kind:'authoring-pane',apply:()=>{setNoteSelection({diagramID:activeDiagram.id,id:'new'});setWorkspaceTask('documentation')}})}>Add note</button>}</details></div>}
            {!review && authoringAvailable && activeDiagram && (result.reference_choices?.some((choice) => choice.diagram_id === activeDiagram.id)) && (
              <label className="reference-picker">Show component here
                <select value="" onChange={(event) => {
                  if (event.target.value) void changeReference(result, activeDiagram.id, event.target.value, true)
                }}>
                  <option value="">Choose a component</option>
                  {result.reference_choices.filter((choice) => choice.diagram_id === activeDiagram.id).map((choice) => (
                    <option key={choice.component_id} value={choice.component_id}>{choice.title}{choice.context ? ` — ${choice.context}` : ''} — Lives in {choice.home_diagram}</option>
                  ))}
                </select>
              </label>
            )}
          </nav>
          {!reconciliation && <section className={`map-region ${activeDiagram ? 'has-diagram' : ''}`}>
            <div className="region-label">{review ? (reviewSide === 'with' ? 'With changes map' : 'Before changes map') : activeDiagram?.title ?? 'Architecture map'}</div>
            {candidateOnlyDiagramBefore && <p className="candidate-only-note">That diagram exists only with the changes. Before changes shows the earlier architecture map.</p>}
            {activeDiagram && (
              <nav className="diagram-breadcrumbs" aria-label="Diagram breadcrumbs">
                {activeDiagram.breadcrumbs.map((breadcrumb, index) => (
                  <span key={breadcrumb.id}>
                    {index > 0 && <span aria-hidden="true">/</span>}
                    {breadcrumb.id === activeDiagram.id
                      ? <strong aria-current="page">{breadcrumb.title}</strong>
                      : <button type="button" onClick={() => selectDiagram(breadcrumb.id, breadcrumb.focus_anchor_component_id)}>{breadcrumb.title}</button>}
                  </span>
                ))}
                {review && (activeDiagramAnnotations
                  ? <AnnotationMarker group={activeDiagramAnnotations} onToggle={() => openReviewAnnotation(activeDiagramAnnotations)} />
                  : !submittedReview && <AnnotationAddMarker label={`Comment on ${activeDiagram.title}`} onClick={openDiagramComment} />)}
                {!review && authoringAvailable && activeDiagram.parent_anchor_component_id && <button className="diagram-parent-action" type="button" onClick={() => requestNavigation({ kind: 'change-parent', id: activeDiagram.id })}>Change parent component</button>}
				{!review&&authoringAvailable&&<button className="diagram-parent-action" type="button" disabled={architectureBusy||placementBlocked||editorDirtyRef.current} onClick={()=>keepPosition(result,activeDiagram.id,undefined,null)}>Auto-layout</button>}
              </nav>
            )}
            {reviewPresentation && activeDiagram && ((!submittedReview && reviewCommentTarget?.anchor.kind === 'diagram' && reviewCommentTarget.contextKey.startsWith('diagram-map:')) || diagramAnnotationCards.length > 0) && (
              <aside className="diagram-review-annotation" aria-label={`Comments on ${activeDiagram.title}`}>
                <div className="diagram-review-annotation-heading">
                  <strong>{activeDiagram.title}</strong>
                  {!submittedReview && !reviewCommentTarget && <AnnotationAddMarker label={`Add comment on ${activeDiagram.title}`} onClick={openDiagramComment} />}
                </div>
                {diagramAnnotationCards.length > 0 && <ReviewAnnotationCards
                  groups={diagramAnnotationCards}
                  review={reviewPresentation}
                  onClose={closeReviewAnnotation}
                  onEditLocal={submittedReview ? undefined : editLocalMapComment}
                  onRemoveLocal={submittedReview ? undefined : removeLocalMapComment}
                />}
                {!submittedReview && reviewCommentTarget?.contextKey.startsWith('diagram-map:') && <InlineReviewCommentEditor
                  key={reviewCommentTarget.editorKey}
                  target={reviewCommentTarget}
                  onDirty={setReviewCommentDirty}
                  onSave={saveReviewComment}
                  onCancel={cancelComment}
                />}
              </aside>
            )}
            {result.changes && !result.changes.candidate && !review ? (
              <div className="workspace-empty invalid-proposal-map"><p className="eyebrow">Proposed Architecture</p><h2>Needs correction</h2><p>This proposal has no valid complete Architecture to display. Use its exact authored facts to repair the issue.</p></div>
            ) : (
              <ArchitectureMap
				viewKey={`${result.store_id}:${activeDiagram?.id??'root'}:${review?.candidate_tree??'authoring'}`}
				onPlace={!review&&authoringAvailable&&!architectureBusy&&!placementBlocked&&!editor&&!diagramEditor&&!editorDirtyRef.current&&activeDiagram ? (id,p)=>id.startsWith('note:')?keepNoteGeometry(result,activeDiagram,id,p):keepPosition(result,activeDiagram.id,id,p):undefined}
				onResize={!review&&authoringAvailable&&!architectureBusy&&!placementBlocked&&!editor&&!diagramEditor&&!editorDirtyRef.current&&activeDiagram ? (id,s)=>id.startsWith('note:')?keepNoteGeometry(result,activeDiagram,id,s):keepSize(result,activeDiagram.id,id,s):undefined}
                revision={`${activeProjection?.revision ?? diagramProjection.revision}${activeDiagram ? `:${activeDiagram.id}` : ''}`}
                components={mapComponents}
                reviewOtherComponents={reviewOtherComponents}
                selectedID={noteSelection?.diagramID===activeDiagram?.id?`note:${noteSelection?.id}`:selectedBoundary?.key ?? selectedComponentID}
                onSelect={selectMapNode}
                onRoute={!review&&authoringAvailable&&!architectureBusy&&!placementBlocked&&!editor&&!diagramEditor&&!editorDirtyRef.current ? (route,bend)=>keepRoute(result,route,{bend}):undefined}
                {...(!review ? {selectedRelationshipKey:selectedRoute?.key,onSelectRelationship:(edge:ReviewRelationshipSelection)=>requestNavigation({kind:'authoring-pane',apply:()=>{setSelectedRouteKey(edge.key);setWorkspaceTask('documentation')}})}:{})}
                emptyMessage={activeDiagram ? 'This diagram has no components.' : undefined}
                {...(review ? {
                  layoutComponentIDs,
                  reviewSide,
                  reviewComponents: review.comparison.components,
                  reviewPositionIDs: review.comparison.node_positions?.filter(p => p.diagram_id === activeDiagram?.id).map(p => p.component_id),
                  reviewSizeIDs: [...(review.comparison.node_sizes??[]),...(review.comparison.node_shapes??[])].filter(p => p.diagram_id === activeDiagram?.id).map(p => p.component_id),
                  reviewRelationships: review.comparison.relationships,
                  reviewComposition: <>
                    {review.comparison.edge_routes?.filter(r=>r.diagram_id===activeDiagram?.id).map(r=><li key={JSON.stringify([r.source_id,r.target_id,r.label,r.occurrence])}><button type="button" onClick={()=>focusReviewDiagram({kind:'diagram',key:JSON.stringify(r),title:activeDiagram?.title??'Diagram',status:'appearance_changed',description:'Route changed',reviewSide,diagramID:r.diagram_id,path:r.path})}>Route changed: {r.label} · occurrence {r.occurrence} · {r.before?`Bend ${r.before.bend}`:r.before_state==='not_applicable'?'Not visible':'Default'} → {r.with?`Bend ${r.with.bend}`:r.with_state==='not_applicable'?'Not visible':'Default'}</button></li>)}
                    {review.comparison.node_sizes?.filter(s=>s.diagram_id===activeDiagram?.id).map(s=><li key={`size:${s.component_id}`}><button type="button" onClick={()=>focusReviewDiagram({kind:'diagram',key:`size:${s.component_id}`,title:activeDiagram?.title??'Diagram',status:'appearance_changed',description:'Size changed',reviewSide,diagramID:s.diagram_id,path:s.path,componentID:s.component_id})}>Size changed: {diagramProjection.components.find(c=>c.id===s.component_id)?.title??'Component'} · {s.before?`${s.before.width} × ${s.before.height}`:'Not visible'} → {s.with?`${s.with.width} × ${s.with.height}`:'Not visible'}</button></li>)}
					{review.comparison.node_shapes?.filter(s=>s.diagram_id===activeDiagram?.id).map(s=><li key={`shape:${s.component_id}`}><button type="button" onClick={()=>selectMapNode(s.component_id)}>Shape changed: {diagramProjection.components.find(c=>c.id===s.component_id)?.title??'Component'} · {s.before??'Default'} → {s.with??'Default'}</button></li>)}
					{review.comparison.diagram_notes?.filter(n=>n.diagram_id===activeDiagram?.id).map(n=><li key={`note:${n.note_id}`}><button type="button" onClick={()=>{if(!n.with)setReviewSide('before');selectMapNode(`note:${n.note_id}`)}}>Note {n.before?(n.with?'changed':'removed'):'added'}: {(n.with??n.before)?.text.slice(0,80)}</button></li>)}
                    {[...mapComposition.entries()].map(([identity, item]) => <li key={identity}><button type="button" onClick={() => focusReviewDiagram(item.focus)}>Composition: {item.subject} {item.description}</button></li>)}
                    {review.comparison.node_positions?.filter(p => p.diagram_id === activeDiagram?.id).map(p => <li key={`position:${p.component_id}`}>
                      <button type="button" onClick={() => focusReviewDiagram({
                        kind: 'diagram', key: `position:${p.component_id}`, title: activeDiagram?.title ?? 'Diagram',
                        status: 'appearance_changed', description: 'Position changed', reviewSide,
                        diagramID: p.diagram_id, path: p.path, componentID: p.component_id,
                      })}>
                        Position changed: {diagramProjection.components.find(c => c.id === p.component_id)?.title ?? 'Component'} · {p.before ? `${p.before.x}, ${p.before.y}` : p.before_source==='derived'?'Derived v2 layout':'Not visible'} → {p.with ? `${p.with.x}, ${p.with.y}` : p.with_source==='derived'?'Derived v2 layout':'Not visible'}
                      </button>
                    </li>)}
                  </>,
                  reviewDiagramID: activeDiagram?.id,
                  selectedRelationshipKey: reviewFocus?.kind === 'relationship' ? reviewFocus.key : undefined,
                  onSelectRelationship: selectRelationship,
                } : {})}
                annotationNodes={Object.fromEntries(Object.entries(mapNodeAnnotationGroups).map(([key, group]) => [key, group.comments.length]))}
                annotationRelationships={Object.fromEntries(Object.entries(relationshipAnnotationGroups).map(([key, group]) => [key, group.comments.length]))}
                annotationAddNodeID={review && !submittedReview && reviewFocus?.kind !== 'relationship' && selectedComponentID && mapComponents.some((component) => component.id === selectedComponentID) && !mapNodeAnnotationGroups[selectedComponentID]
                  ? selectedComponentID : undefined}
                annotationAddRelationshipKey={!submittedReview && reviewFocus?.kind === 'relationship' && !relationshipAnnotationGroups[reviewFocus.key]
                  ? reviewFocus.key : undefined}
                onSelectNodeAnnotation={(id) => {
                  const group = mapNodeAnnotationGroups[id]
                  if (group) {
                    openReviewAnnotation(group)
                    return
                  }
                  const component = mapComponents.find((candidate) => candidate.id === id)
                  if (!component || submittedReview) return
                  const componentID = component.component_id ?? component.id
                  requestReviewContextReplacement(() => openCommentEditor({
                    contextKey: `map-node:${annotationSide}:${id}`,
                    label: component.title,
                    anchor: { kind: 'component', side: annotationSide, component_id: componentID },
                    mapTarget: { kind: 'node', id, diagramID: activeDiagram?.id },
                  }))
                }}
                onSelectRelationshipAnnotation={(key) => {
                  const group = relationshipAnnotationGroups[key]
                  if (group) {
                    openReviewAnnotation(group)
                    return
                  }
                  if (submittedReview || reviewFocus?.kind !== 'relationship' || reviewFocus.key !== key) return
                  const exactSide = reviewFocus.review_side ? reviewSideValue(reviewFocus.review_side) : annotationSide
                  requestReviewContextReplacement(() => openCommentEditor({
                    contextKey: `map-relationship:${exactSide}:${key}`,
                    label: `${reviewFocus.source_title} — ${reviewFocus.label} → ${reviewFocus.target_title}`,
                    anchor: { kind: 'relationship', side: exactSide, source_component_id: reviewFocus.source_id,
                      target_component_id: reviewFocus.target_id, label: reviewFocus.label, occurrence: reviewFocus.occurrence },
                    mapTarget: { kind: 'relationship', id: key, diagramID: activeDiagram?.id },
                  }))
                }}
                externalReferences={externalReferences}
                annotationOverlay={annotationOverlay}
              />
            )}
          </section>}
          <aside className="working-pane" aria-label="Architecture task" ref={workingPaneRef}>
            {result.changes && !review && !reconciliation && !creatingChangeSet && workspaceTask !== 'changes' && <nav className="proposal-task-navigation" aria-label="Proposal task"><button className="text-action" type="button" disabled={architectureBusy} onClick={() => requestNavigation({ kind: 'changes' })}>Back to proposal</button></nav>}
            {selectedRoute?.routing ? <section aria-label="Link route"><p className="eyebrow">Link</p><h2>{selectedRoute.label}</h2><p>{diagramProjection.components.find(c=>c.id===selectedRoute.source_component_id)?.title} → {diagramProjection.components.find(c=>c.id===selectedRoute.target_component_id)?.title}</p><p>Occurrence {selectedRoute.routing.occurrence} of {selectedRoute.routing.count}</p><button className="text-action" type="button" onClick={()=>requestNavigation({kind:'authoring-pane',apply:()=>setSelectedRouteKey(undefined)})}>Clear selection</button>{selectedRoute.routing.eligible&&authoringAvailable ? <RouteControls key={`${positionDraftEpoch}:${selectedRoute.key}:${selectedRoute.routing.display_bend}:${Boolean(selectedRoute.routing.route)}`} route={selectedRoute.routing} busy={architectureBusy||placementBlocked} onDirty={setRouteDirty} onKeep={route=>keepRoute(result,selectedRoute.routing!,route)}/> : <p>{selectedRoute.routing.reason==='self_link'?'Self-links use their default route.':'These nodes share a center. Routing is unavailable; default rendering is used where possible until they separate.'}</p>}</section> : reconciliation ? <ReconciliationTask
              key={reconciliation.inputs.change_set_state}
              name={result.changes?.name ?? 'Proposal'}
              initial={reconciliation}
              onDirty={setReconciliationDirty}
              onLeave={() => requestNavigation({ kind: 'changes' })}
              onCheck={async (resolutions: ReconciliationResolution[]) => {
                const response = await postJSON('/api/agent/v2/change-sets/reconcile-preview', { ...reconciliation.inputs, resolutions })
                const payload = await response.json()
                if (stateRef.current !== state) throw new Error('The workspace changed. Prepare again from the current proposal.')
                if (!payload.ok) throw new Error(payload.error?.message ?? 'Choices could not be checked.')
                return payload.result as ReconciliationPreview
              }}
              onApply={async (resolutions: ReconciliationResolution[]) => {
                const response = await postJSON('/api/agent/v2/change-sets/reconcile-apply', { ...reconciliation.inputs, resolutions })
                const payload = await response.json()
                if (!payload.ok) throw new Error(payload.error?.message ?? 'The Apply response could not be confirmed.')
                if (stateRef.current !== state) return
                setReconciliation(null)
                setReconciliationDirty(false)
                editorDirtyRef.current = false
                enterProposalResult(payload.result.workspace as ArchitectureResult, reconciliation.inputs.change_set_id)
                setArchitectureNotice(payload.result.remaining_changes ? 'Reconciliation applied to this proposal. Review changes to continue.' : 'No Architecture changes remain. This proposal stays open on the current Accepted basis.')
              }}
              renderCandidate={(projection: ReconciliationSnapshot, requestedID, selectedID, onSelect) => {
                const snapshot = projection as ReviewSnapshot
                const diagram = snapshot.diagrams?.find((item) => item.id === (requestedID ?? snapshot.root_diagram_id))
                const nodes = diagram ? mapComponentsForDiagram(snapshot, diagram) : snapshot.components
                return <ArchitectureMap
                  revision={`${projection.revision}:${diagram?.id ?? ''}`}
                  components={nodes}
                  selectedID={nodes.find((node) => (('component_id' in node && node.component_id) || node.id) === selectedID)?.id}
                  onSelect={(id) => { const node = nodes.find((node) => node.id === id); onSelect((node && 'component_id' in node && node.component_id) || id) }}
                  emptyMessage="This diagram has no components."
                  fitPadding={12}
                />
              }}
            /> : creatingChangeSet ? (
              <NewChangesTask
                name={newChangeSetName}
                busy={architectureBusy}
                onName={setNewChangeSetName}
                onCreate={() => createChangeSet(result)}
                onCancel={() => {
                  const shouldReturnThroughHistory = newChangesPushedHistoryRef.current
                  newChangesPushedHistoryRef.current = false
                  editorDirtyRef.current = false
                  setCreatingChangeSet(false)
                  setNewChangeSetName('')
                  resetWorkingPaneScroll()
                  if (shouldReturnThroughHistory) window.history.back()
                }}
              />
            ) : noteSelection&&noteSelection.diagramID===activeDiagram?.id&&(noteSelection.id==='new'||activeDiagram?.notes?.some(n=>n.id===noteSelection.id)) ? <DiagramNotePane key={`${noteSelection.id}:${diagramProjection.revision}`} note={activeDiagram?.notes?.find(n=>n.id===noteSelection.id)} readOnly={Boolean(review)||!authoringAvailable} busy={architectureBusy||placementBlocked} onDirty={setNoteDirty} onClear={()=>requestNavigation({kind:'clear'})} onKeep={v=>keepShapeNote(result,activeDiagram!.id,noteSelection.id==='new'?'add-note':'edit-note',noteSelection.id==='new'?{text:v.text}:{note_id:noteSelection.id,...v})} onDelete={()=>keepShapeNote(result,activeDiagram!.id,'delete-note',{note_id:noteSelection.id})}/> : review && result.changes ? (
              <ChangesTask
                result={result}
                busy={architectureBusy}
                acceptanceUnknown={acceptanceUnknown}
                discardConfirming={discardConfirming}
                reviewSide={reviewSide}
                selectedReviewComponent={selected}
                reviewFocus={reviewFocus}
                activeReviewSubmission={result.submitted_review}
                activeDiagram={activeDiagram}
                commentTarget={reviewCommentTarget}
                onCommentTarget={requestComment}
                onCommentSave={saveReviewComment}
                onCommentCancel={cancelComment}
                onCommentDirty={setReviewCommentDirty}
                unfinishedComment={Boolean(reviewCommentTarget)}
                onOpenSubmittedReview={(reviewID) => requestNavigation({ kind: 'submitted-review', changeSetID: result.changes!.id, reviewID })}
                onOpenCurrentReview={currentProposalRecord?.review ? () => requestNavigation({ kind: 'review-result', result: { ...result, submitted_review: undefined, changes: currentProposalRecord } }) : undefined}
                onOpenProposal={currentProposalRecord ? () => requestNavigation({ kind: 'context', id: currentProposalRecord.id }) : undefined}
                onOpenAccepted={() => requestNavigation({ kind: 'context', id: 'accepted' })}
                onOpenAnnotation={openReviewAnnotation}
                localReviewComments={localReviewComments}
                onRemoveLocalComment={removeLocalMapComment}
                onSubmitReview={(input) => submitReviewFeedback(result, input)}
                onReviewSide={switchReviewSide}
                onContinueEditing={() => requestNavigation({ kind: 'continue-editing' })}
                onClearReviewFocus={() => requestReviewContextReplacement(() => {
                  setReviewSelectionCleared(true)
                  setSelectedComponentID(undefined)
                  setReviewFocus(null)
                })}
                onFocusDiagram={focusReviewDiagram}
                onEdit={(component) => editPending(component, undefined, result.stale || result.changes?.stale)}
                onFixRelationship={(component) => editPending(component, {
                  position: result.changes?.validation_relationship_position ?? 0,
                  field: result.changes?.validation_relationship_field ?? 'target',
                })}
                onReview={() => reviewChanges(result)}
                onUpdate={() => updateArchitecture(result)}
                onRename={(name) => renameChangeSet(result, name)}
                onSaveProposal={(proposal) => saveProposal(result, proposal)}
                onTextDirty={setChangeSetTextDirty}
                onBeginDiscard={() => setDiscardConfirming(true)}
                onCancelDiscard={() => setDiscardConfirming(false)}
                onDiscard={() => discardChanges(result)}
              />
            ) : diagramEditor ? (
              <DiagramEditorForm
                editor={diagramEditor}
                setEditor={setDiagramEditor}
                diagrams={editorDiagrams}
                error={authoringError}
                onCancel={() => setDiagramEditor(null)}
                onSubmit={(event) => submitDiagramChange(event, result)}
              />
            ) : editor ? (
              <ComponentEditorForm
                editor={editor}
                setEditor={setEditor}
                targets={relationshipTargetsFor(result)}
                error={authoringError}
                onCancel={() => setEditor(null)}
                onSubmit={(event) => submitComponent(event, result)}
              />
            ) : workspaceTask === 'changes' && result.changes ? (
              <ChangesTask
                onReconcile={result.changes.out_of_date && result.changes.valid && !result.changes.read_only && !result.stale && !authorityIndeterminate ? () => requestNavigation({ kind: 'reconcile' }) : undefined}
                result={result}
                busy={architectureBusy}
                acceptanceUnknown={acceptanceUnknown}
                discardConfirming={discardConfirming}
                onOpenSubmittedReview={(reviewID) => requestNavigation({ kind: 'submitted-review', changeSetID: result.changes!.id, reviewID })}
                localReviewComments={localReviewComments}
                onRemoveLocalComment={removeLocalMapComment}
                onReturnToReview={currentReview && !acceptanceUnknown ? () => {
                  requestNavigation({ kind: 'review-result', result })
                } : undefined}
                onEdit={(component) => editPending(component, undefined, result.stale || result.changes?.stale)}
                onFixRelationship={(component) => editPending(component, {
                  position: result.changes?.validation_relationship_position ?? 0,
                  field: result.changes?.validation_relationship_field ?? 'target',
                })}
                onAddComponent={(result.format_version ?? 0) >= 2 ? (diagramID) => requestNavigation({kind:'authoring-pane',apply:()=>addComponent(diagramID)}) : undefined}
                onCreateDetail={(result.format_version ?? 0) >= 2 ? (componentID) => requestNavigation({kind:'authoring-pane',apply:()=>setDiagramEditor({ kind: 'detail', componentID, title: '', initialTitle: '' })}) : undefined}
                onEditDiagramTitle={(result.format_version ?? 0) >= 2 ? (diagramID, title, invalid) => requestNavigation({kind:'authoring-pane',apply:()=>setDiagramEditor({ kind: 'title', diagramID, title, initialTitle: title, invalid })}) : undefined}
                onMoveHome={(result.format_version ?? 0) >= 2 ? (componentID)=>requestNavigation({kind:'authoring-pane',apply:()=>beginHomeMove(componentID)}) : undefined}
                onShowComponent={(diagramID, componentID) => changeReference(result, diagramID, componentID, true)}
                onStopShowing={(diagramID, componentID) => changeReference(result, diagramID, componentID, false)}
                onReview={() => reviewChanges(result)}
                onUpdate={() => updateArchitecture(result)}
                onRename={(name) => renameChangeSet(result, name)}
                onSaveProposal={(proposal) => saveProposal(result, proposal)}
                onTextDirty={setChangeSetTextDirty}
                onBeginDiscard={() => setDiscardConfirming(true)}
                onCancelDiscard={() => setDiscardConfirming(false)}
                onDiscard={() => discardChanges(result)}
              />
            ) : selected ? (
              <article className="component-documentation">
                <div className="pane-heading pane-heading-with-action"><div><p className="eyebrow">Component</p><h2>{selected.title}</h2></div><button className="text-action" type="button" onClick={() => requestNavigation({ kind: 'clear' })}>Clear selection</button></div>
                <MarkdownBody source={selected.description} />
				{selectedBoundary&&<button className="inline-action" type="button" onClick={()=>selectDiagram(selectedBoundary.home_diagram_id,selectedBoundary.component_id)}>Open home · {selectedBoundary.home_diagram_title}</button>}
                {authoringAvailable&&activeDiagram&&<details className="position-controls" key={`geometry:${activeDiagram.id}:${selected.id}`}>
                  <summary>Position and size</summary>
				  <label>Shape<select aria-label="Node shape" disabled={architectureBusy||placementBlocked||positionDirty||sizeDirty} value={activeDiagram.shapes?.find(s=>s.component_id===selected.id)?.shape??'default'} onChange={e=>void keepShapeNote(result,activeDiagram.id,e.target.value==='default'?'restore-default-shape':'set-shape',{component_id:selected.id,...(e.target.value==='default'?{}:{shape:e.target.value})})}><option value="default">Default</option><option value="rectangle">Rectangle</option><option value="ellipse">Ellipse</option><option value="diamond">Diamond</option></select></label>
                  <PositionControls key={`${positionDraftEpoch}:${activeDiagram.id}:${selected.id}:${selectedPosition?.x}:${selectedPosition?.y}`} position={selectedPosition} busy={architectureBusy||placementBlocked||sizeDirty} onDirty={setPositionDirty} onKeep={p=>keepPosition(result,activeDiagram.id,selected.id,p)} />
                  <SizeControls key={`size:${positionDraftEpoch}:${activeDiagram.id}:${selected.id}:${selectedSize.width}:${selectedSize.height}`} size={selectedSize} busy={architectureBusy||placementBlocked||positionDirty} onDirty={setSizeDirty} onKeep={s=>keepSize(result,activeDiagram.id,selected.id,s)} />
                </details>}
                {(authoringAvailable || selectedAppearance?.detail_diagram_id) && (
                  <div className="component-documentation-actions">
                    {authoringAvailable && <button className="inline-action" type="button" onClick={() => requestNavigation({kind:'authoring-pane',apply:()=>editAccepted(selected, result)})}>Edit component</button>}
                    {selectedAppearance?.detail_diagram_id && (
                      <button className="secondary-action detail-link" type="button" onClick={() => selectDiagram(selectedAppearance.detail_diagram_id!)}>
                        Open {selectedAppearance.detail_diagram_title}
                      </button>
                    )}
                    {authoringAvailable && selectedAppearance && (
                      <div className="diagram-composition-actions" role="group" aria-label="Diagram composition">
                        <span>Diagram</span>
                        <div>
                          {selectedAppearance.role === 'home' && !selectedAppearance.detail_diagram_id && (
                            <button className="text-action diagram-composition-link" type="button" onClick={() => requestNavigation({kind:'authoring-pane',apply:()=>setDiagramEditor({ kind: 'detail', componentID: selected.id, title: '', initialTitle: '' })})}>Create detail diagram</button>
                          )}
                          {selectedAppearance.role === 'home' && result.home_move_destinations?.find((destinations) => destinations.component_id === selected.id)?.diagram_ids.length ? <button className="text-action diagram-composition-link" type="button" onClick={() => requestNavigation({kind:'authoring-pane',apply:()=>beginHomeMove(selected.id)})}>Change where {componentAuthoringLabel(compositionProjection.components, selected.id)} lives</button> : null}
                          {selectedAppearance.role === 'reference' && activeDiagram && <button className="text-action diagram-composition-link" type="button" onClick={() => requestNavigation({kind:'authoring-pane',apply:()=>{void changeReference(result, activeDiagram.id, selected.id, false)}})}>Stop showing here</button>}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </article>
            ) : activeDiagram ? (
              <div className="workspace-empty"><p className="eyebrow">Diagram</p><h2>{activeDiagram.appearances.length ? 'Select a component' : 'No components here'}</h2><p>{activeDiagram.appearances.length ? 'Choose a component from the index or map to read its documentation.' : 'This diagram is intentionally empty.'}</p></div>
            ) : result.components?.length ? (
              <div className="workspace-empty"><p className="eyebrow">Architecture</p><h2>Select a component</h2><p>Choose a component from the index or map to read its documentation.</p></div>
            ) : (
              <div className="workspace-empty"><p className="eyebrow">Architecture</p><h2>Start with a component</h2><p>Add the first part of this architecture to begin the map.</p></div>
            )}
            {!reconciliation && !creatingChangeSet && !(workspaceTask === 'changes' && result.changes) && !review && (
              <details className="technical-details">
                <summary>Technical details</summary>
                <dl><dt>Project slug</dt><dd>{result.project_slug}</dd><dt>Revision</dt><dd>{result.revision}</dd></dl>
                {result.parent_diff && <div className="accepted-diff"><h3>Parent diff</h3><pre>{result.parent_diff}</pre></div>}
              </details>
            )}
          </aside>
        </div>
        {navigationIntent && (
          <div className="navigation-guard" role="dialog" aria-modal="true" aria-labelledby="unsaved-heading">
            <div>
              <h2 id="unsaved-heading">Leave without keeping?</h2>
              <p>Your latest edits have not been kept.</p>
              <div className="button-group">
                <button className="secondary-action" type="button" onClick={() => setNavigationIntent(null)}>Keep editing</button>
                <button className="destructive-action" type="button" onClick={() => performNavigation(navigationIntent)}>Leave without keeping</button>
              </div>
            </div>
          </div>
        )}
      </main>
    )
  }

  return (
    <main className="shell">
      <article className="sheet">
        <header className="sheet-header">
          <p className="eyebrow">WorkBraid</p>
          <h1>Projects</h1>
          <p className="introduction">Open an Architecture project or begin a new one.</p>
        </header>

        <form onSubmit={createProject}>
          <label htmlFor="project-name">New project</label>
          <div className="input-row">
            <input
              id="project-name"
              name="project-name"
              type="text"
              value={projectName}
              onChange={(event) => setProjectName(event.target.value)}
              placeholder="Example project"
              autoComplete="off"
            />
            <button type="submit" disabled={busy || !projectName.trim()}>{busy ? 'Working…' : 'Create project'}</button>
          </div>
        </form>
        {state.kind === 'catalog' && state.projects.length > 0 && (
          <nav className="project-catalog" aria-label="Projects">
            {state.projects.map((project, index) => project.unavailable ? (
              <div className="catalog-project unavailable" key={project.store_id ?? index}>
                <strong>Project unavailable</strong>
                <small>{project.store_id}</small>
              </div>
            ) : project.conflict ? (
              <div className="catalog-project conflict" key={project.store_id ?? index}>
                <strong>{project.name}</strong><span>Project address conflict</span><small>{project.slug}</small>
              </div>
            ) : (
              <button
                className="catalog-project"
                type="button"
                key={project.store_id ?? project.slug}
                aria-label={state.projects.filter((candidate) => !candidate.unavailable && candidate.name === project.name).length > 1
                  ? `${project.name}, ${project.slug}`
                  : project.name}
                onClick={() => void openProject(project.slug ?? '')}
              >
                <strong>{project.name}</strong>
                {state.projects.filter((candidate) => !candidate.unavailable && candidate.name === project.name).length > 1 && <small>{project.slug}</small>}
              </button>
            ))}
          </nav>
        )}
        {state.kind === 'catalog' && state.projects.length === 0 && <p className="catalog-empty">No projects yet.</p>}
        {state.kind === 'looking' && <p className="lookup-status">Opening projects…</p>}
        {state.kind === 'not-found' && <section className="result-note error" role="alert"><h2>Project not found</h2><p>No project currently uses <strong>{state.slug}</strong>.</p><button className="inline-action" type="button" onClick={() => { window.history.pushState({}, '', '/'); void loadCatalog() }}>Back to projects</button></section>}
        {state.kind === 'catalog-error' && <section className="result-note error" role="alert"><h2>Projects unavailable</h2><p>{state.message}</p><button className="inline-action" type="button" onClick={() => void loadCatalog()}>Try again</button></section>}
      </article>
    </main>
  )
}

function ComponentEditorForm({
  editor,
  setEditor,
  targets,
  error,
  onCancel,
  onSubmit,
}: {
  editor: ComponentEditor
  setEditor: (editor: ComponentEditor) => void
  targets: RelationshipTarget[]
  error: string
  onCancel: () => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  const issueRowKey = editor.relationshipIssue && editor.relationshipIssue.position > 0
    ? editor.relationships[editor.relationshipIssue.position - 1]?.rowKey
    : undefined
  useEffect(() => {
    if (!issueRowKey || !editor.relationshipIssue) return
    document.getElementById(`relationship-${editor.relationshipIssue.field}-${issueRowKey}`)?.focus()
  }, [editor.relationshipIssue, issueRowKey])

  return (
    <form className="component-form" onSubmit={(event) => {
      if (editor.readOnly) {
        event.preventDefault()
        return
      }
      onSubmit(event)
    }}>
      <div className="pane-heading"><p className="eyebrow">Architecture</p><h2>{editor.readOnly ? 'Change details' : editor.kind === 'add' ? 'Add component' : 'Edit component'}</h2></div>
      {editor.readOnly && <p className="stale-change-note">These changes started from an older architecture and cannot be edited.</p>}
      <label htmlFor="component-title">Title</label>
      <input
        id="component-title"
        value={editor.title}
        readOnly={editor.readOnly}
        onChange={(event) => setEditor({ ...editor, title: event.target.value, titleChanged: true })}
        autoComplete="off"
      />
      <label htmlFor="component-description">Description</label>
      <textarea
        id="component-description"
        value={editor.description}
        readOnly={editor.readOnly}
        onChange={(event) => setEditor({ ...editor, description: event.target.value, descriptionChanged: true })}
        rows={14}
      />
      <fieldset className="relationship-editor">
        <legend>Outgoing relationships</legend>
        {editor.relationships.length ? (
          <div className="relationship-rows">
            {editor.relationships.map((relationship, index) => (
              <div className="relationship-row" key={relationship.rowKey}>
                <label htmlFor={`relationship-target-${relationship.rowKey}`}>Target</label>
                <select
                  id={`relationship-target-${relationship.rowKey}`}
                  value={relationship.target_id}
                  disabled={editor.readOnly}
                  aria-invalid={editor.relationshipIssue?.position === index + 1 && editor.relationshipIssue.field === 'target' ? true : undefined}
                  aria-describedby={editor.relationshipIssue?.position === index + 1 && editor.relationshipIssue.field === 'target' ? `relationship-target-guidance-${relationship.rowKey}` : undefined}
                  onChange={(event) => setEditor({
                    ...editor,
                    relationships: editor.relationships.map((row) => row.rowKey === relationship.rowKey ? { ...row, target_id: event.target.value } : row),
                  })}
                >
                  <option value="">Choose a component</option>
                  {targets.map((target) => <option key={target.id} value={target.id}>{relationshipTargetLabel(target)}</option>)}
                </select>
                {editor.relationshipIssue?.position === index + 1 && editor.relationshipIssue.field === 'target' && (
                  <p className="field-guidance" id={`relationship-target-guidance-${relationship.rowKey}`}>Choose a component for this relationship.</p>
                )}
                <label htmlFor={`relationship-label-${relationship.rowKey}`}>Label</label>
                <textarea
                  id={`relationship-label-${relationship.rowKey}`}
                  value={relationship.label}
                  readOnly={editor.readOnly}
                  aria-invalid={editor.relationshipIssue?.position === index + 1 && editor.relationshipIssue.field === 'label' ? true : undefined}
                  aria-describedby={editor.relationshipIssue?.position === index + 1 && editor.relationshipIssue.field === 'label' ? `relationship-label-guidance-${relationship.rowKey}` : undefined}
                  rows={2}
                  placeholder="calls"
                  onChange={(event) => setEditor({
                    ...editor,
                    relationships: editor.relationships.map((row) => row.rowKey === relationship.rowKey ? { ...row, label: event.target.value } : row),
                  })}
                />
                {editor.relationshipIssue?.position === index + 1 && editor.relationshipIssue.field === 'label' && (
                  <p className="field-guidance" id={`relationship-label-guidance-${relationship.rowKey}`}>Add a label to this relationship.</p>
                )}
                {!editor.readOnly && <button
                  className="text-action relationship-remove"
                  type="button"
                  aria-label={`Remove relationship ${index + 1}`}
                  onClick={() => setEditor({ ...editor, relationships: editor.relationships.filter((row) => row.rowKey !== relationship.rowKey) })}
                >Remove</button>}
              </div>
            ))}
          </div>
        ) : <p className="relationship-empty">No outgoing relationships</p>}
        {!editor.readOnly && <button
          className="secondary-action relationship-add"
          type="button"
          onClick={() => setEditor({ ...editor, relationships: [...editor.relationships, newRelationshipRow()] })}
        >Add relationship</button>}
      </fieldset>
      {error && <p className="authoring-error" role="alert">{error}</p>}
      <div className="button-group">
        <button className="secondary-action" type="button" onClick={onCancel}>{editor.readOnly ? 'Back' : 'Cancel'}</button>
        {!editor.readOnly && <button className="inline-action" type="submit">Keep change</button>}
      </div>
    </form>
  )
}

function DiagramEditorForm({
  editor,
  setEditor,
  diagrams,
  error,
  onCancel,
  onSubmit,
}: {
  editor: DiagramEditor
  setEditor: (editor: DiagramEditor) => void
  diagrams: DiagramAuthoringOption[]
  error: string
  onCancel: () => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  if (editor.kind === 'parent') {
    const groups = [...new Set(editor.options.eligible.map((option) => option.home_diagram_id))]
    return <form className="component-form diagram-editor" onSubmit={onSubmit}>
      <div className="pane-heading"><p className="eyebrow">{editor.options.title} · Diagram composition</p><h2>Change parent component</h2></div>
      <p>Currently under {editor.options.current_anchor.title} · {editor.options.current_anchor.home_diagram_title}.</p>
      {groups.length === 0 && <p className="empty-note">No free parent component is available outside this Diagram’s subtree. Add a suitable Component to the proposal, then try again.</p>}
      {groups.map((home) => <fieldset className="parent-choices" key={home}>
        <legend>Lives in {editor.options.eligible.find((option) => option.home_diagram_id === home)?.home_diagram_title}</legend>
        {editor.options.eligible.filter((option) => option.home_diagram_id === home).map((option) => <label className="parent-choice" key={option.component_id}>
          <input type="radio" name="parent-component" checked={editor.anchorID === option.component_id} onChange={() => setEditor({ ...editor, anchorID: option.component_id })} />
          <span>{option.title}{editor.options.eligible.filter((other) => other.title === option.title).length > 1 && <small>{option.filename}</small>}</span>
        </label>)}
      </fieldset>)}
      {error && <p className="authoring-error" role="alert">{error}</p>}
      <div className="button-group"><button className="secondary-action" type="button" onClick={onCancel}>Cancel</button><button className="inline-action" type="submit" disabled={!editor.anchorID}>Keep change</button></div>
    </form>
  }
  return (
    <form className="component-form diagram-editor" onSubmit={onSubmit}>
      <div className="pane-heading"><p className="eyebrow">Diagram composition</p><h2>{editor.kind === 'detail' ? 'Create detail diagram' : editor.kind === 'title' ? 'Edit diagram title' : `Change where ${editor.componentTitle} lives`}</h2></div>
      {editor.kind === 'move' ? (
        <>
          <p>Currently lives in {editor.currentDiagramTitle}.</p>
          <label>Diagram
            <select autoFocus value={editor.diagramID} onChange={(event) => setEditor({ ...editor, diagramID: event.target.value })}>
              <option value="">Choose a diagram</option>
              {diagrams.map((diagram) => <option key={diagram.id} value={diagram.id}>{diagramAuthoringOptionLabel(diagram)}</option>)}
            </select>
          </label>
        </>
      ) : (
        <label>Diagram title
          <textarea
            autoFocus
            aria-invalid={editor.invalid || undefined}
            rows={2}
            value={editor.title}
            onChange={(event) => setEditor({ ...editor, title: event.target.value })}
          />
        </label>
      )}
      {error && <p className="authoring-error" role="alert">{error}</p>}
      <div className="button-group">
        <button className="secondary-action" type="button" onClick={onCancel}>Cancel</button>
        <button className="inline-action" type="submit" disabled={editor.kind === 'move' && !editor.diagramID}>Keep change</button>
      </div>
    </form>
  )
}

function NewChangesTask({
  name,
  busy,
  onName,
  onCreate,
  onCancel,
}: {
  name: string
  busy: boolean
  onName: (name: string) => void
  onCreate: () => Promise<void>
  onCancel: () => void
}) {
  return (
    <form className="new-changes-task" onSubmit={(event) => { event.preventDefault(); void onCreate() }}>
      <div className="pane-heading">
        <p className="eyebrow">Architecture</p>
        <h2>New changes</h2>
      </div>
      <p className="new-changes-introduction">Starts from current Accepted.</p>
      <label htmlFor="new-proposal-name">Name</label>
      <input id="new-proposal-name" autoFocus value={name} onChange={(event) => onName(event.target.value)} />
      <p className="field-hint">Leave blank to generate a name.</p>
      <div className="button-group new-changes-actions">
        <button className="inline-action" type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create'}</button>
        <button className="secondary-action" type="button" disabled={busy} onClick={onCancel}>Cancel</button>
      </div>
    </form>
  )
}

function ChangesTask({
  onReconcile,
  result,
  busy,
  acceptanceUnknown,
  discardConfirming,
  reviewSide,
  selectedReviewComponent,
  reviewFocus,
  activeReviewSubmission,
  activeDiagram,
  onOpenSubmittedReview,
  onOpenCurrentReview,
  onOpenProposal,
  onOpenAccepted,
  onOpenAnnotation,
  localReviewComments,
  onRemoveLocalComment,
  commentTarget,
  onCommentTarget,
  onCommentSave,
  onCommentCancel,
  onCommentDirty,
  unfinishedComment = false,
  onSubmitReview,
  onReviewSide,
  onContinueEditing,
  onReturnToReview,
  onClearReviewFocus,
  onFocusDiagram,
  onEdit,
  onFixRelationship,
  onAddComponent,
  onCreateDetail,
  onEditDiagramTitle,
  onMoveHome,
  onShowComponent,
  onStopShowing,
  onReview,
  onUpdate,
  onRename,
  onSaveProposal,
  onTextDirty,
  onBeginDiscard,
  onCancelDiscard,
  onDiscard,
}: {
  onReconcile?: () => void
  result: ArchitectureResult
  busy: boolean
  acceptanceUnknown: boolean
  discardConfirming: boolean
  reviewSide?: ReviewSide
  selectedReviewComponent?: AuthoringComponent
  reviewFocus?: ReviewFocus | null
  activeReviewSubmission?: ReviewSubmission
  activeDiagram?: DiagramProjection
  onOpenSubmittedReview?: (reviewID: string) => void
  onOpenCurrentReview?: () => void
  onOpenProposal?: () => void
  onOpenAccepted?: () => void
  onOpenAnnotation?: (group: ReviewAnnotationGroup) => void
  localReviewComments: LocalReviewComment[]
  onRemoveLocalComment: (commentID: string) => void
  commentTarget?: ReviewCommentTarget
  onCommentTarget?: (target: ReviewCommentTarget) => void
  onCommentSave?: (body: string, anchor: ReviewAnchor) => void
  onCommentCancel?: () => void
  onCommentDirty?: (dirty: boolean) => void
  unfinishedComment?: boolean
  onSubmitReview?: (input: { author: string; verdict: ReviewSubmissionSummary['verdict']; body: string; comments: { body: string; anchor: ReviewAnchor }[] }) => void
  onReviewSide?: (side: ReviewSide) => void
  onContinueEditing?: () => void
  onReturnToReview?: () => void
  onClearReviewFocus?: () => void
  onFocusDiagram?: (focus: Extract<ReviewFocus, { kind: 'diagram' }>) => void
  onEdit: (component: PendingComponent) => void
  onFixRelationship: (component: PendingComponent) => void
  onAddComponent?: (diagramID: string) => void
  onCreateDetail?: (componentID: string) => void
  onEditDiagramTitle?: (diagramID: string, title: string, invalid?: boolean) => void
  onMoveHome?: (componentID: string) => void
  onShowComponent?: (diagramID: string, componentID: string) => void
  onStopShowing?: (diagramID: string, componentID: string) => void
  onReview: () => void
  onUpdate: () => void
  onRename: (name: string) => void
  onSaveProposal: (proposal: string) => void
  onTextDirty: (dirty: boolean) => void
  onBeginDiscard: () => void
  onCancelDiscard: () => void
  onDiscard: () => void
}) {
  const changes = result.changes
  const [reviewFormDirty, setReviewFormDirty] = useState(false)
  const [proposalCommentsOpen, setProposalCommentsOpen] = useState(false)
  const reviewContextRef = useRef<HTMLDivElement>(null)
  const setCommentTarget = (target: ReviewCommentTarget) => onCommentTarget?.(target)
  useEffect(() => {
    setReviewFormDirty(false)
  }, [changes?.review?.reviewed_state, activeReviewSubmission?.id])
  useEffect(() => {
    if (!changes?.review || activeReviewSubmission) return
    onTextDirty(reviewFormDirty || localReviewComments.length > 0)
  }, [changes?.review, activeReviewSubmission, reviewFormDirty, localReviewComments.length, onTextDirty])
  useEffect(() => {
    if (reviewFocus) reviewContextRef.current?.scrollIntoView?.({ block: 'start' })
  }, [reviewFocus?.key, selectedReviewComponent?.id])
  if (!changes) return null
  const readOnly = Boolean(result.stale || result.action_error === 'refresh_failed' || changes.stale || changes.read_only || changes.lifecycle === 'applied')
  const relationshipIssueComponent = changes.validation_relationship_position && changes.validation_relationship_field
    ? changes.components.find((component) => component.id === changes.validation_item)
    : undefined
  const relationshipIssueName = relationshipIssueComponent ? relationshipIssueComponentName(changes, relationshipIssueComponent) : ''
  const diagramCompositionNeedsAttention = Boolean(!changes.candidate && changes.validation_diagram_field && changes.validation_code)
  const affectedDiagramTitle = changes.validation_diagram_field === 'title'
    ? changes.detail_diagrams?.find((diagram) => diagram.id === changes.validation_diagram)?.title
      ?? changes.diagram_titles?.find((diagram) => diagram.diagram_id === changes.validation_diagram)?.title
    : undefined
  const compositionComponents = changes.candidate?.components ?? result.components
  const reviewAnnotationComments = activeReviewSubmission?.comments ?? localReviewComments
  const proposalAnnotations = annotationGroup(
    `proposal:${activeReviewSubmission?.id ?? changes.review?.reviewed_state ?? changes.id}`,
    'Proposal feedback',
    reviewAnnotationComments.filter((comment) => comment.anchor.kind === 'proposal' || comment.anchor.kind === 'proposal_markdown'),
  )
  const inlineCommentEditor = (contextKey: string) => commentTarget?.contextKey === contextKey && onCommentSave && onCommentCancel && onCommentDirty ? (
    <InlineReviewCommentEditor
      key={commentTarget.editorKey}
      target={commentTarget}
      onDirty={onCommentDirty}
      onSave={onCommentSave}
      onCancel={onCommentCancel}
    />
  ) : null
  const discardAction = !acceptanceUnknown && !readOnly && changes.lifecycle === 'active'
    ? <button className="discard-action" type="button" disabled={busy} onClick={onBeginDiscard}>Delete proposal</button>
    : null
  if (changes.review && reviewSide && onReviewSide && onClearReviewFocus) {
    return (
      <section className="changes-in-progress review-workspace-pane" aria-labelledby="review-heading">
        <div className="review-heading-row">
          <div className="pane-heading"><p className="eyebrow">{activeReviewSubmission ? 'Submitted feedback' : 'Architecture'}</p><h2 id="review-heading">{activeReviewSubmission ? verdictLabel(activeReviewSubmission.verdict) : 'Review changes'}</h2></div>
          <div className="review-side-toggle" role="group" aria-label="Review side">
            <button type="button" aria-pressed={reviewSide === 'with'} onClick={() => onReviewSide('with')}>With changes</button>
            <button type="button" aria-pressed={reviewSide === 'before'} onClick={() => onReviewSide('before')}>Before changes</button>
          </div>
        </div>
        <p className="review-proposal-name"><span>{changes.lifecycle === 'applied' ? 'Accepted proposal' : changes.lifecycle === 'no_longer_active' ? 'Proposal no longer active' : 'Open proposal'}</span><strong>{changes.name}</strong></p>
        {!activeReviewSubmission && <nav className="proposal-task-navigation" aria-label="Proposal task">
          {onContinueEditing && <button className="text-action" type="button" disabled={busy} onClick={onContinueEditing}>Back to proposal</button>}
          {changes.lifecycle === 'active' && onSubmitReview && <button className="text-action" type="button" onClick={() => {
            const heading = document.getElementById('review-feedback-heading')
            heading?.focus()
            heading?.scrollIntoView?.({ block: 'start' })
          }}>Write review</button>}
        </nav>}
        {activeReviewSubmission && (
          <>
            <nav className="submitted-review-navigation" aria-label="Review navigation">
              {onOpenCurrentReview && <button className="secondary-action" type="button" onClick={onOpenCurrentReview}>Current review</button>}
              {onOpenProposal && <button className="secondary-action" type="button" onClick={onOpenProposal}>Proposal workspace</button>}
              {onOpenAccepted && <button className="secondary-action" type="button" onClick={onOpenAccepted}>Accepted workspace</button>}
            </nav>
            <SubmittedReviewList reviews={result.review_submissions?.filter((item) => item.change_set_id === changes.id) ?? []} onOpen={onOpenSubmittedReview} compact />
          </>
        )}
        {activeReviewSubmission && <SubmittedReviewDetails review={activeReviewSubmission} />}
        <p className="review-introduction">{changes.out_of_date
          ? 'Out of date with Accepted. You can inspect this review, but it cannot update Architecture until the proposal matches Accepted.'
          : readOnly
            ? 'Review the proposed architecture and its complete file changes.'
            : 'Review the proposed architecture and its complete file changes before updating Architecture.'}</p>
        <details className="review-proposal-document" aria-label="Proposal" open>
          <summary>Proposal</summary>
          <div className="context-comment-actions">
            {!activeReviewSubmission && <button className="text-action" type="button" onClick={() => setCommentTarget({ contextKey: 'proposal', label: 'Proposal', anchor: { kind: 'proposal' } })}>Comment on proposal</button>}
            {!activeReviewSubmission && changes.proposal_markdown && <button className="text-action" type="button" onClick={() => setCommentTarget({ contextKey: 'proposal', label: 'Proposal lines', anchor: { kind: 'proposal_markdown' }, source: changes.proposal_markdown })}>Comment on lines</button>}
            {proposalAnnotations && <AnnotationMarker group={proposalAnnotations} onToggle={() => setProposalCommentsOpen(open => !open)} />}
          </div>
          {proposalCommentsOpen && proposalAnnotations && <ReviewAnnotationCards groups={[proposalAnnotations]}
            review={activeReviewSubmission ?? { review: changes.review, proposal_markdown: changes.proposal_markdown }} onClose={() => setProposalCommentsOpen(false)} />}
          {inlineCommentEditor('proposal')}
          {changes.proposal_markdown
            ? <MarkdownBody source={changes.proposal_markdown} />
            : <p className="proposal-empty">No proposal document.</p>}
        </details>
        {(changes.review.comparison.diagrams?.length || changes.review.comparison.appearances?.length) ? (
          <section className="diagram-review-summary" aria-label="Diagram changes">
            <h3>Diagram changes</h3>
            <ul>
              {changes.review.comparison.diagrams?.map((diagram) => {
                const side = diagram.status === 'added' ? 'with_changes' : reviewSide === 'before' ? 'before' : 'with_changes'
                const annotations = diagramAnnotation(reviewAnnotationComments, side, diagram.diagram_id, diagram.title)
                const contextKey = `diagram-change:${diagram.diagram_id}:${diagram.status}`
                return <li key={`${diagram.diagram_id}:${diagram.status}`}><span className="diagram-review-row"><button className="diagram-review-target text-action" type="button" onClick={() => onFocusDiagram?.({ kind: 'diagram', key: `diagram:${diagram.diagram_id}`, diagramID: diagram.diagram_id, title: diagram.title, path: diagram.path, status: diagram.status })}><strong>{diagram.title}</strong> {diagram.status === 'added' ? 'added' : 'title changed'}</button><span className="diagram-review-actions">{!activeReviewSubmission && <button className="annotation-add-action" type="button" onClick={() => setCommentTarget({ contextKey, label: diagram.title, anchor: { kind: 'diagram', side, diagram_id: diagram.diagram_id } })}>Add comment</button>}{annotations && onOpenAnnotation && <AnnotationMarker group={annotations} onToggle={() => onOpenAnnotation(annotations)} />}</span></span>{inlineCommentEditor(contextKey)}</li>
              })}
              {changes.review.comparison.appearances?.map((appearance, index) => {
                const presentation = appearanceReviewPresentation(changes.review!, appearance, index)
                const { subject, description, focus, movedChild } = presentation
                const label = `${subject} ${description}`
                if (movedChild) {
                  const sides = [...changes.review!.comparison.appearances!.entries()].filter(([, item]) => item.status === 'detail_changed' && item.detail_diagram_id === appearance.detail_diagram_id)
                  if (sides[0][0] !== index) return null
                  sides.sort(([, a], [, b]) => a.side === b.side ? 0 : a.side === 'before' ? -1 : 1)
                  const resultChild = changes.review!.with_changes.diagrams!.find((diagram) => diagram.id === appearance.detail_diagram_id)!
                  return <li className="composition-review-change" key={`reassignment:${appearance.detail_diagram_id}`}>
                    <p className="reassignment-review-label"><strong>{diagramAuthoringOptionLabel(resultChild)}</strong> {description}</p>
                    {sides.map(([sideIndex, item]) => {
                      const snapshot = item.side === 'before' ? changes.review!.before : changes.review!.with_changes
                      const parent = snapshot.diagrams!.find((diagram) => diagram.id === item.diagram_id)!
                      const child = snapshot.diagrams!.find((diagram) => diagram.id === item.detail_diagram_id)!
                      const sideLabel = item.side === 'before' ? 'Before changes' : 'With changes'
                      const sideContext = `${componentAuthoringLabel(snapshot.components, item.component_id)} → ${diagramAuthoringOptionLabel(child)} · in ${diagramAuthoringOptionLabel(parent)}`
                      const sideFocus = appearanceReviewPresentation(changes.review!, item, sideIndex).focus
                      const anchor: ReviewAnchor = { kind: 'composition', side: item.side, diagram_id: item.diagram_id, component_id: item.component_id, aspect: 'detail', detail_diagram_id: item.detail_diagram_id }
                      const contextKey = `appearance-change:${item.diagram_id}:${item.component_id}:${item.status}:${sideIndex}`
                      const annotations = compositionAnnotation(reviewAnnotationComments, item.side, item.diagram_id, item.component_id, `${sideLabel} · ${sideContext}`)
                      return <div className="reassignment-review-side" key={item.side}>
                        <span className="diagram-review-row"><button className="diagram-review-target text-action" type="button" onClick={() => onFocusDiagram?.(sideFocus)}><strong>{sideLabel}</strong> · {sideContext}</button>
                          <span className="diagram-review-actions">{!activeReviewSubmission && <button className="annotation-add-action" type="button" onClick={() => setCommentTarget({ contextKey, label: `${sideLabel} · ${sideContext}`, anchor })}>Comment on {sideLabel}</button>}{annotations && onOpenAnnotation && <AnnotationMarker group={annotations} onToggle={() => onOpenAnnotation(annotations)} />}</span>
                        </span>
                        {inlineCommentEditor(contextKey)}
                      </div>
                    })}
                  </li>
                }
                const detailDiagramID = appearance.status === 'detail_changed' ? appearance.detail_diagram_id : undefined
                const anchor: ReviewAnchor = { kind: 'composition', side: appearance.side, diagram_id: appearance.diagram_id, component_id: appearance.component_id,
                  aspect: detailDiagramID ? 'detail' : appearance.role, ...(detailDiagramID ? { detail_diagram_id: detailDiagramID } : {}) }
                const annotations = compositionAnnotation(reviewAnnotationComments, appearance.side, appearance.diagram_id, appearance.component_id, label)
                const contextKey = `appearance-change:${appearance.diagram_id}:${appearance.component_id}:${appearance.status}:${index}`
                return <li className="composition-review-change" key={`${appearance.diagram_id}:${appearance.component_id}:${appearance.status}:${index}`}><span className="diagram-review-row"><button className="diagram-review-target text-action" type="button" onClick={() => onFocusDiagram?.(focus)}><strong>{subject}</strong> {description}</button><span className="diagram-review-actions">{!activeReviewSubmission && <button className="annotation-add-action" type="button" onClick={() => setCommentTarget({ contextKey, label, anchor })}>Comment on this change</button>}{annotations && onOpenAnnotation && <AnnotationMarker group={annotations} onToggle={() => onOpenAnnotation(annotations)} />}</span></span>{inlineCommentEditor(contextKey)}</li>
              })}
            </ul>
          </section>
        ) : null}
        <div ref={reviewContextRef} className="review-context-container"><ReviewContext
          side={reviewSide}
          component={selectedReviewComponent}
          components={reviewSide === 'with' ? changes.review.with_changes.components : changes.review.before.components}
          diagram={activeDiagram}
          focus={reviewFocus}
          onClear={onClearReviewFocus}
          onComment={!activeReviewSubmission ? setCommentTarget : undefined}
          comments={reviewAnnotationComments}
          renderCommentEditor={inlineCommentEditor}
          onOpenAnnotation={onOpenAnnotation}
        /></div>
        {changes.review.diff ? <section className="raw-diff-region" aria-labelledby="raw-diff-heading">
          <div className="review-section-heading"><h3 id="raw-diff-heading">Complete change</h3><span>Raw unified diff</span></div>
          <RawDiff
            diff={changes.review.diff}
            focusPath={reviewFocus?.path}
            focusToken={reviewFocus?.key}
          />
        </section> : <p>No Architecture changes; this proposal contains only proposal text.</p>}
        {!activeReviewSubmission && changes.lifecycle === 'active' && onSubmitReview && (
          <ReviewComposer
            key={changes.review.reviewed_state}
            comments={localReviewComments}
            onRemove={onRemoveLocalComment}
            onEdit={comment => setCommentTarget({
              contextKey: `review-summary:${comment.id}`,
              label: reviewAnchorPresentation({ review: changes.review!, proposal_markdown: changes.proposal_markdown }, comment.anchor).label,
              anchor: comment.anchor,
              localCommentID: comment.id,
              initialBody: comment.body,
              source: comment.anchor.kind === 'proposal_markdown' ? changes.proposal_markdown
                : comment.anchor.kind === 'component_markdown'
                  ? (comment.anchor.side === 'before' ? changes.review!.before : changes.review!.with_changes)
                    .components.find(component => component.id === comment.anchor.component_id)?.markdown_source
                  : undefined,
            })}
            renderEditor={inlineCommentEditor}
            busy={busy}
            onDirty={setReviewFormDirty}
            onSubmit={onSubmitReview}
            unfinishedComment={unfinishedComment}
          />
        )}
        {!activeReviewSubmission && <SubmittedReviewList reviews={result.review_submissions?.filter((item) => item.change_set_id === changes.id) ?? []} onOpen={onOpenSubmittedReview} />}
        <div className="change-actions">
          {changes.review.diff === '' && <p role="status">There is no Architecture change to accept.</p>}
          {!activeReviewSubmission && !readOnly && !changes.out_of_date && changes.review.diff !== '' && <button className="inline-action" type="button" disabled={busy} onClick={onUpdate}>{busy ? 'Updating…' : 'Update architecture'}</button>}
          {discardAction}
        </div>
        {discardConfirming && <DiscardChangesDialog busy={busy} onCancel={onCancelDiscard} onDiscard={onDiscard} />}
        <details className="review-details technical-details">
          <summary>Technical details</summary>
          <dl className="change-set-binding">
            <dt>ID</dt><dd>{changes.id}</dd>
            <dt>Based on</dt><dd>{changes.review.base_revision}</dd>
            <dt>Reviewed tree</dt><dd>{changes.review.candidate_tree}</dd>
            <dt>Change version</dt><dd>{changes.review.generation}</dd>
            {changes.review.reviewed_state && <><dt>Reviewed state</dt><dd>{changes.review.reviewed_state}</dd></>}
          </dl>
        </details>
      </section>
    )
  }
  return (
    <section className="changes-in-progress" aria-labelledby="changes-heading">
      <div className="pane-heading"><p className="eyebrow">{changes.lifecycle === 'applied' ? 'Accepted proposal' : 'Open proposal'}</p><h2 id="changes-heading">{changes.name}</h2></div>
      {!acceptanceUnknown && result.action_error !== 'update_uncertain' && <p className="proposal-status">{changes.lifecycle === 'applied'
        ? 'This is the proposal that updated Architecture. It cannot be changed.'
        : changes.out_of_date
          ? 'Out of date with Accepted. You can still edit and review this proposal, but it cannot update Architecture until it matches Accepted.'
          : 'These changes have not updated Architecture yet.'}</p>}
      {onReconcile && <div className="proposal-reconciliation-action"><button className="inline-action" type="button" disabled={busy} onClick={onReconcile}>Reconcile with Accepted</button><p>Combine current Accepted work with this proposal and resolve conflicting changes.</p></div>}
      <ChangeSetContextEditor key={`${changes.id}:${changes.generation}`} changes={changes} busy={busy} readOnly={readOnly} onRename={onRename} onSaveProposal={onSaveProposal} onDirty={onTextDirty} />
      <h3 className="proposal-work-heading">Architecture work in this proposal</h3>
      <ul>
        {changes.components.map((component) => {
          const ownsReviewBlocker = Boolean(changes.review_blocker && changes.validation_item === component.id)
          return (
          <li className={ownsReviewBlocker ? 'validation-owner' : undefined} aria-invalid={ownsReviewBlocker || undefined} key={component.id}>
            <span>{component.title.trim() || 'Untitled component'}</span>
            {ownsReviewBlocker && <strong className="validation-marker">Needs attention</strong>}
            {!acceptanceUnknown && <button className="text-action" type="button" onClick={() => onEdit(component)}>{readOnly ? 'View' : 'Edit'}</button>}
          </li>
          )
        })}
      </ul>
      {changes.candidate?.diagrams?.length ? (
        <section className="pending-diagram-composition" aria-label="Diagram changes in progress">
          <h3>Diagram composition</h3>
          <div className="pending-diagram-list">
            {changes.candidate.diagrams.map((diagram) => (
              <section className="pending-diagram-row" key={diagram.id}>
                <header className="pending-diagram-header">
                  <div className="pending-diagram-title"><strong>{diagram.title}</strong>{changes.detail_diagrams?.some((addition) => addition.id === diagram.id) && <small> New diagram</small>}</div>
                  <div className="pending-diagram-header-actions">
                    {!readOnly && !acceptanceUnknown && onEditDiagramTitle && <button className="text-action pending-diagram-action" type="button" onClick={() => onEditDiagramTitle(diagram.id, diagram.title)}>Edit title</button>}
                    {!readOnly && !acceptanceUnknown && onAddComponent && <button className="text-action pending-diagram-action" type="button" onClick={() => onAddComponent(diagram.id)}>Add component</button>}
                  </div>
                </header>
                <div className="pending-diagram-body">
                  <ul>
                    {diagram.appearances.map((appearance) => {
                      const component = changes.candidate?.components.find((candidate) => candidate.id === appearance.component_id)
                      return <li key={appearance.component_id}>
                        <span>{component?.title ?? 'Component'}{appearance.role === 'reference' && <small className="appearance-note"> Included here</small>}</span>
                        {!readOnly && appearance.role === 'home' && onMoveHome && result.home_move_destinations?.find((destinations) => destinations.component_id === appearance.component_id)?.diagram_ids.length ? <button className="text-action" type="button" onClick={() => onMoveHome(appearance.component_id)}>Change where {componentAuthoringLabel(compositionComponents, appearance.component_id)} lives</button> : null}
                        {!readOnly && appearance.role === 'home' && !appearance.detail_diagram_id && onCreateDetail && <button className="text-action" type="button" onClick={() => onCreateDetail(appearance.component_id)}>Create detail diagram</button>}
                        {!readOnly && appearance.role === 'reference' && onStopShowing && <button className="text-action" type="button" onClick={() => onStopShowing(diagram.id, appearance.component_id)}>Stop showing here</button>}
                      </li>
                    })}
                  </ul>
                  {!readOnly && onShowComponent && result.reference_choices?.some((choice) => choice.diagram_id === diagram.id) && (
                    <label className="pending-reference-picker">Show component here
                      <select value="" onChange={(event) => {
                        if (event.target.value) onShowComponent(diagram.id, event.target.value)
                      }}>
                        <option value="">Choose a component</option>
                        {result.reference_choices.filter((choice) => choice.diagram_id === diagram.id).map((choice) => (
                          <option key={choice.component_id} value={choice.component_id}>{choice.title}{choice.context ? ` — ${choice.context}` : ''} — Lives in {choice.home_diagram}</option>
                        ))}
                      </select>
                    </label>
                  )}
                </div>
              </section>
            ))}
          </div>
        </section>
      ) : null}
      {diagramCompositionNeedsAttention ? (
        <section className="pending-diagram-attention validation-owner" aria-label="Diagram composition needs attention" aria-invalid="true">
          <h3>Diagram composition needs attention</h3>
          <div role="alert">
            <p><strong>{affectedDiagramTitle?.trim() || 'Untitled diagram'}</strong> <span className="validation-marker">Needs attention</span></p>
            <p>{messageForReviewBlocker(changes.validation_code)}</p>
            {!readOnly && !acceptanceUnknown && changes.validation_diagram_field === 'title' && changes.validation_diagram && onEditDiagramTitle && (
              <button className="inline-action" type="button" onClick={() => onEditDiagramTitle(changes.validation_diagram!, affectedDiagramTitle ?? '', true)}>Fix diagram title</button>
            )}
          </div>
          <p className="pending-preserved-note">All other changes in progress are still kept.</p>
        </section>
      ) : null}
      {changes.review_blocker && !diagramCompositionNeedsAttention && (
        <div className="review-error" role="alert">
          {relationshipIssueComponent ? (
            <>
              <p><strong>{relationshipIssueName}</strong> has a relationship {readOnly ? 'issue in these read-only changes.' : 'to fix.'}</p>
              <p>{readOnly ? messageForReadOnlyReviewBlocker(changes.review_blocker) : messageForReviewBlocker(changes.review_blocker)}</p>
              {!readOnly && <button className="inline-action fix-relationship" type="button" onClick={() => onFixRelationship(relationshipIssueComponent)}>Fix relationship</button>}
            </>
          ) : (
            <>
              <p>{messageForReviewBlocker(changes.review_blocker)}</p>
              {!readOnly && changes.validation_diagram_field === 'title' && changes.validation_diagram && onEditDiagramTitle && (
                <button className="inline-action" type="button" onClick={() => {
                  const pendingDiagram = changes.detail_diagrams?.find((diagram) => diagram.id === changes.validation_diagram)
                  const pendingTitle = changes.diagram_titles?.find((diagram) => diagram.diagram_id === changes.validation_diagram)
                  const candidateDiagram = changes.candidate?.diagrams?.find((diagram) => diagram.id === changes.validation_diagram)
                  onEditDiagramTitle(changes.validation_diagram!, pendingDiagram?.title ?? pendingTitle?.title ?? candidateDiagram?.title ?? '', true)
                }}>Fix diagram title</button>
              )}
            </>
          )}
        </div>
      )}
      {result.action_error && !changes.review_blocker && <p className="review-error" role="alert">{messageForArchitectureAction(result.action_error)}</p>}
      {(!changes.review || readOnly || onReturnToReview) && !acceptanceUnknown && (
        <div className="change-actions">
          {!readOnly && changes.review && onReturnToReview && (
            <button className="inline-action" type="button" onClick={onReturnToReview}>Return to review</button>
          )}
          {!readOnly && changes.lifecycle === 'active' && !changes.review && (
            <button className="inline-action" type="button" disabled={busy} onClick={onReview}>{busy ? 'Preparing…' : 'Review changes'}</button>
          )}
          {discardAction}
        </div>
      )}
      {discardConfirming && <DiscardChangesDialog busy={busy} onCancel={onCancelDiscard} onDiscard={onDiscard} />}
      <SubmittedReviewList reviews={result.review_submissions?.filter((item) => item.change_set_id === changes.id) ?? []} onOpen={onOpenSubmittedReview} />
      <details className="technical-details">
        <summary>Technical details</summary>
        <dl className="change-set-binding">
          <dt>ID</dt><dd>{changes.id}</dd>
          <dt>Based on</dt><dd>{changes.base_revision}</dd>
          <dt>Change version</dt><dd>{changes.generation}</dd>
          {changes.applied_revision && <><dt>Accepted as</dt><dd>{changes.applied_revision}</dd></>}
        </dl>
      </details>
    </section>
  )
}

function ChangeSetContextEditor({
  changes,
  busy,
  readOnly,
  onRename,
  onSaveProposal,
  onDirty,
}: {
  changes: ChangesInProgress
  busy: boolean
  readOnly: boolean
  onRename: (name: string) => void
  onSaveProposal: (proposal: string) => void
  onDirty: (dirty: boolean) => void
}) {
  const [name, setName] = useState(changes.name)
  const [proposal, setProposal] = useState(changes.proposal_markdown)
  const nameDirty = name !== changes.name
  const proposalDirty = proposal !== changes.proposal_markdown
  const dirty = nameDirty || proposalDirty
  useEffect(() => {
    onDirty(dirty)
    return () => onDirty(false)
  }, [dirty, onDirty])
  return (
    <section className="proposal-content" aria-label="Proposal">
      {readOnly ? (
        <section className="proposal-document"><h3>Proposal</h3><MarkdownBody source={changes.proposal_markdown} /></section>
      ) : (
        <>
          <form className="change-set-name-form" onSubmit={(event) => { event.preventDefault(); onRename(name) }}>
            <label htmlFor={`proposal-name-${changes.id}`}>Name</label>
            <input id={`proposal-name-${changes.id}`} value={name} onChange={(event) => setName(event.target.value)} />
            {nameDirty && <button className="text-action proposal-save-action" type="submit" disabled={busy}>Save name</button>}
          </form>
          <form className="proposal-editor" onSubmit={(event) => { event.preventDefault(); onSaveProposal(proposal) }}>
            <label htmlFor={`proposal-document-${changes.id}`}>Proposal</label>
            <textarea id={`proposal-document-${changes.id}`} rows={8} value={proposal} onChange={(event) => setProposal(event.target.value)} />
            {proposalDirty && <button className="secondary-action proposal-save-action" type="submit" disabled={busy}>Save proposal</button>}
          </form>
          <section className="proposal-document proposal-preview"><h3>Preview</h3>{proposal ? <MarkdownBody source={proposal} /> : <p className="proposal-empty">Nothing written yet.</p>}</section>
        </>
      )}
    </section>
  )
}

function appearanceReviewDescription(role: 'home' | 'reference', status: 'added' | 'removed' | 'detail_changed', diagramTitle: string) {
  if (status === 'detail_changed') return `detail diagram link changed in ${diagramTitle}`
  if (role === 'home') return status === 'added' ? `now lives in ${diagramTitle}` : `no longer lives in ${diagramTitle}`
  return status === 'added' ? `shown in ${diagramTitle}` : `no longer shown in ${diagramTitle}`
}

function appearanceReviewPresentation(review: ChangeReview, appearance: ReviewAppearanceChange, index: number) {
  const exactSide = appearance.side === 'before' ? review.before : review.with_changes
  const otherSide = appearance.side === 'before' ? review.with_changes : review.before
  const component = exactSide.components.find((item) => item.id === appearance.component_id)
    ?? otherSide.components.find((item) => item.id === appearance.component_id)
  const diagram = exactSide.diagrams?.find((item) => item.id === appearance.diagram_id)
    ?? otherSide.diagrams?.find((item) => item.id === appearance.diagram_id)
  const detailDiagramID = appearance.status === 'detail_changed' ? appearance.detail_diagram_id : undefined
  const beforeChild = detailDiagramID ? review.before.diagrams?.find((item) => item.id === detailDiagramID) : undefined
  const withChild = detailDiagramID ? review.with_changes.diagrams?.find((item) => item.id === detailDiagramID) : undefined
  const movedChild = Boolean(beforeChild?.parent_anchor_component_id && withChild?.parent_anchor_component_id && beforeChild.parent_anchor_component_id !== withChild.parent_anchor_component_id)
  let subject = component?.title ?? 'Component'
  let description = appearanceReviewDescription(appearance.role, appearance.status, diagram?.title ?? 'Diagram')
  if (movedChild) {
    subject = diagramAuthoringOptionLabel((appearance.side === 'before' ? beforeChild : withChild)!)
    description = `moved from ${componentAuthoringLabel(review.before.components, beforeChild!.parent_anchor_component_id!)} to ${componentAuthoringLabel(review.with_changes.components, withChild!.parent_anchor_component_id!)}`
  }
  const focus: Extract<ReviewFocus, { kind: 'diagram' }> = {
    kind: 'diagram', key: `appearance:${appearance.diagram_id}:${appearance.component_id}:${index}`,
    diagramID: appearance.diagram_id, title: diagram?.title ?? 'Diagram', description: `${subject} ${description}`,
    path: appearance.path, status: 'appearance_changed', componentID: appearance.component_id, role: appearance.role,
    compositionAspect: detailDiagramID ? 'detail' : appearance.role, detailDiagramID,
    reviewSide: appearance.side === 'before' ? 'before' : 'with',
  }
  return { subject, description, movedChild, focus }
}

function verdictLabel(verdict: ReviewSubmissionSummary['verdict']) {
  if (verdict === 'approve') return 'Approved'
  if (verdict === 'request_changes') return 'Changes requested'
  return 'Comment'
}

function anchorLabel(anchor: ReviewAnchor) {
  const side = anchor.side === 'before' ? 'Before changes' : anchor.side === 'with_changes' ? 'With changes' : ''
  switch (anchor.kind) {
    case 'proposal': return 'Whole proposal'
    case 'proposal_markdown': return `Proposal lines ${anchor.start_line}–${anchor.end_line}`
    case 'component': return `${side} Component`
    case 'component_markdown': return `${side} Component lines ${anchor.start_line}–${anchor.end_line}`
    case 'diagram': return `${side} Diagram`
    case 'composition': return `${side} ${anchor.aspect === 'detail' ? 'detail link' : `${anchor.aspect} placement`}`
    case 'relationship': return `${side} Relationship · occurrence ${anchor.occurrence}`
  }
}

function reviewAnchorPresentation(review: ReviewPresentation, anchor: ReviewAnchor) {
  const sideLabel = anchor.side === 'before' ? 'Before changes' : 'With changes'
  const snapshot = anchor.side === 'before' ? review.review.before : review.review.with_changes
  const component = snapshot.components.find((value) => value.id === anchor.component_id)
  const diagram = snapshot.diagrams?.find((value) => value.id === anchor.diagram_id)
  const lineExcerpt = (source: string | undefined) => {
    if (!source || !anchor.start_line || !anchor.end_line) return undefined
    return reviewSourceLines(source).slice(anchor.start_line - 1, anchor.end_line).join('\n')
  }
  if (anchor.kind === 'proposal') return { label: 'Whole proposal' }
  if (anchor.kind === 'proposal_markdown') return { label: `Proposal lines ${anchor.start_line}–${anchor.end_line}`, excerpt: lineExcerpt(review.proposal_markdown) }
  if (anchor.kind === 'component') return { label: `${sideLabel} · ${component?.title ?? 'Component'}` }
  if (anchor.kind === 'component_markdown') return { label: `${sideLabel} · ${component?.title ?? 'Component'} · lines ${anchor.start_line}–${anchor.end_line}`, excerpt: lineExcerpt(component?.markdown_source) }
  if (anchor.kind === 'diagram') return { label: `${sideLabel} · ${diagram?.title ?? 'Diagram'}` }
  if (anchor.kind === 'composition') {
    const placed = snapshot.components.find((value) => value.id === anchor.component_id)
    const aspect = anchor.aspect === 'detail' ? 'detail link' : anchor.aspect === 'reference' ? 'shown here' : 'home'
    return { label: `${sideLabel} · ${placed?.title ?? 'Component'} · ${aspect} in ${diagram?.title ?? 'Diagram'}` }
  }
  if (anchor.kind === 'relationship') {
    const source = snapshot.components.find((value) => value.id === anchor.source_component_id)
    const target = snapshot.components.find((value) => value.id === anchor.target_component_id)
    return { label: `${sideLabel} · ${source?.title ?? 'Component'} — ${anchor.label} → ${target?.title ?? 'Component'} · occurrence ${anchor.occurrence}` }
  }
  return { label: anchorLabel(anchor) }
}

function SubmittedReviewDetails({ review }: { review: ReviewSubmission }) {
  return (
    <section className="submitted-review-details" aria-label="Submitted review feedback">
      <p className="submitted-review-byline">{review.author} · {new Date(review.submitted_at).toLocaleString()}</p>
      {review.lifecycle === 'no_longer_active' && <p className="review-age-note">This proposal is no longer active.</p>}
      {!review.current_generation && review.lifecycle !== 'no_longer_active' && <p className="review-age-note">Feedback on an earlier proposal version.</p>}
      {review.body && <div className="submitted-review-body"><MarkdownBody source={review.body} /></div>}
      {review.comments.length > 0 && <ol className="submitted-review-comments">{review.comments.map((comment) => {
        const anchor = reviewAnchorPresentation(review, comment.anchor)
        return <li key={comment.id}><span>{anchor.label}</span>{anchor.excerpt !== undefined && <pre className="review-anchor-excerpt">{anchor.excerpt || ' '}</pre>}<MarkdownBody source={comment.body} /></li>
      })}</ol>}
    </section>
  )
}

function AnnotationMarker({ group, onToggle, label }: { group: ReviewAnnotationGroup; onToggle: () => void; label?: string }) {
  return <button className="review-annotation-marker" type="button" aria-label={`${group.comments.length} ${label ?? 'review comment'}${group.comments.length === 1 ? '' : 's'} on ${group.label}`} onClick={(event) => { event.stopPropagation(); onToggle() }}>✎ {group.comments.length}</button>
}

function AnnotationAddMarker({ label, onClick }: { label: string; onClick: () => void }) {
  return <button className="review-annotation-marker review-annotation-add" type="button" aria-label={label} onClick={(event) => { event.stopPropagation(); onClick() }}>✎ +</button>
}

function ReviewAnnotationCards({ groups, review, onClose, onEditLocal, onRemoveLocal, onAdd }: {
  groups: ReviewAnnotationGroup[]
  review: ReviewPresentation
  onClose: (key: string) => void
  onEditLocal?: (comment: LocalReviewComment, group: ReviewAnnotationGroup) => void
  onRemoveLocal?: (commentID: string) => void
  onAdd?: (group: ReviewAnnotationGroup) => void
}) {
  return (
    <aside className="review-annotation-cards" aria-label="Open review comments">
      {groups.map((group) => <article
        className="review-annotation-card"
        key={group.key}
        {...(group.mapTarget ? { 'data-map-annotation-kind': group.mapTarget.kind, 'data-map-annotation-id': group.mapTarget.id } : {})}
      >
        <header><strong>{group.label}</strong><button type="button" aria-label={`Close comments on ${group.label}`} onClick={() => onClose(group.key)}>×</button></header>
        {group.comments.map((comment) => {
          const context = reviewAnchorPresentation(review, comment.anchor)
          const local = comment.id.startsWith('local-review-comment-')
          return <section key={comment.id}><span>{context.label}</span>{context.excerpt !== undefined && <pre>{context.excerpt || ' '}</pre>}<MarkdownBody source={comment.body} />{local && (onEditLocal || onRemoveLocal) && <div className="review-annotation-card-actions">{onEditLocal && <button className="text-action" type="button" onClick={() => onEditLocal(comment, group)}>Edit</button>}{onRemoveLocal && <button className="text-action" type="button" onClick={() => onRemoveLocal(comment.id)}>Remove</button>}</div>}</section>
        })}
        {onAdd && <button className="text-action annotation-note-add" type="button" onClick={() => onAdd(group)}>Add comment</button>}
      </article>)}
    </aside>
  )
}

function SubmittedReviewList({ reviews, onOpen, compact = false }: { reviews: ReviewSubmissionSummary[]; onOpen?: (reviewID: string) => void; compact?: boolean }) {
  if (reviews.length === 0) return null
  return (
    <section className={`submitted-review-list ${compact ? 'compact' : ''}`} aria-label="Submitted reviews">
      <details>
        <summary>Review history <span>{reviews.length}</span></summary>
        <ul>{reviews.map((item) => (
          <li key={item.id}><button className="text-action" type="button" onClick={() => onOpen?.(item.id)}><strong>{verdictLabel(item.verdict)}</strong><span>{item.author} · {new Date(item.submitted_at).toLocaleString()} · version {item.binding.generation}{item.lifecycle === 'applied' ? ' · accepted proposal' : item.lifecycle === 'no_longer_active' ? ' · proposal no longer active' : item.current_generation ? '' : ' · earlier version'}{item.out_of_date ? ' · out of date' : ''}</span></button></li>
        ))}</ul>
      </details>
    </section>
  )
}

function reviewSourceLines(source: string) {
  if (!source) return []
  const lines = source.split('\n')
  if (lines[lines.length - 1] === '') lines.pop()
  return lines
}

function InlineReviewCommentEditor({
  target, onDirty, onSave, onCancel, mapPopover = false,
}: {
  target: ReviewCommentTarget
  onDirty: (dirty: boolean) => void
  onSave: (body: string, anchor: ReviewAnchor) => void
  onCancel: () => void
  mapPopover?: boolean
}) {
  const [body, setBody] = useState(target.initialBody ?? '')
  const [startLine, setStartLine] = useState(target.anchor.start_line)
  const [endLine, setEndLine] = useState(target.anchor.end_line)
  const [anchorLine, setAnchorLine] = useState(target.anchor.start_line)
  const composerRef = useRef<HTMLElement>(null)
  const lines = reviewSourceLines(target.source ?? '')
  const lineAnchor = target.anchor.kind === 'proposal_markdown' || target.anchor.kind === 'component_markdown'
  const selectedRange = Boolean(!lineAnchor || (startLine && endLine && endLine >= startLine && endLine <= lines.length))
  const dirty = body !== (target.initialBody ?? '') || (lineAnchor &&
    (startLine !== target.anchor.start_line || endLine !== target.anchor.end_line))
  useEffect(() => { onDirty(dirty); return () => onDirty(false) }, [dirty, onDirty])
  useEffect(() => {
    if (!mapPopover) composerRef.current?.scrollIntoView?.({ block: 'nearest' })
  }, [mapPopover])
  const selectLine = (line: number, extend: boolean) => {
    if (!anchorLine || (!extend && startLine !== endLine)) {
      setAnchorLine(line)
      setStartLine(line)
      setEndLine(line)
      return
    }
    setStartLine(Math.min(anchorLine, line))
    setEndLine(Math.max(anchorLine, line))
  }
  const save = () => {
    if (!body.trim() || !selectedRange) return
    onSave(body, lineAnchor ? { ...target.anchor, start_line: startLine, end_line: endLine } : target.anchor)
  }
  return (
    <section
      ref={composerRef}
      className={`anchored-comment-composer${mapPopover ? ' map-comment-composer' : ''}`}
      aria-label={`Comment on ${target.label}`}
      {...(mapPopover && target.mapTarget ? { 'data-map-annotation-kind': target.mapTarget.kind, 'data-map-annotation-id': target.mapTarget.id } : {})}
    >
      <div className="review-section-heading"><h4>{target.localCommentID ? 'Edit comment' : 'Add comment'}</h4><span>{target.label}</span></div>
      {lineAnchor && <div className="review-line-picker">
        <p>Choose the first and last line. Shift-click also extends the selected range.</p>
        <div className="review-source-lines" role="listbox" aria-label={`${target.label} source lines`} aria-multiselectable="true">
          {lines.map((line, index) => {
            const lineNumber = index + 1
            const selected = Boolean(startLine && endLine && lineNumber >= startLine && lineNumber <= endLine)
            return <button key={lineNumber} type="button" role="option" aria-selected={selected} onClick={(event) => selectLine(lineNumber, event.shiftKey)}><b>{lineNumber}</b><span>{line || ' '}</span></button>
          })}
        </div>
      </div>}
      <label>Comment<textarea rows={4} value={body} onChange={(event) => setBody(event.target.value)} /></label>
      <div className="anchored-comment-actions"><button className="secondary-action" type="button" disabled={!body.trim() || !selectedRange} onClick={save}>{target.localCommentID ? 'Save comment' : 'Add comment'}</button><button className="text-action" type="button" onClick={onCancel}>Cancel</button></div>
    </section>
  )
}

function ReviewComposer({
  comments, onEdit, onRemove, renderEditor, busy, onDirty, onSubmit, unfinishedComment,
}: {
  comments: LocalReviewComment[]
  onEdit: (comment: LocalReviewComment) => void
  onRemove: (commentID: string) => void
  renderEditor: (contextKey: string) => ReactNode
  busy: boolean
  unfinishedComment: boolean
  onDirty: (dirty: boolean) => void
  onSubmit: (input: { author: string; verdict: ReviewSubmissionSummary['verdict']; body: string; comments: { body: string; anchor: ReviewAnchor }[] }) => void
}) {
  const [author, setAuthor] = useState('')
  const [verdict, setVerdict] = useState<ReviewSubmissionSummary['verdict']>('comment')
  const [body, setBody] = useState('')
  const submitReasonID = useId()
  const dirty = author !== '' || body !== '' || comments.length > 0 || verdict !== 'comment'
  useEffect(() => { onDirty(dirty); return () => onDirty(false) }, [dirty, onDirty])
  const valid = author.trim() !== '' && (verdict !== 'comment' || body.trim() !== '' || comments.length > 0)
  const submitReason = unfinishedComment ? 'Add or cancel the open comment before submitting.'
    : !author.trim() ? 'Enter a reviewer name.'
      : !valid ? 'Add a review summary or a comment to submit a Comment review.' : undefined
  return (
    <section className="review-composer" aria-labelledby="review-feedback-heading">
      <div className="review-section-heading"><h3 id="review-feedback-heading" tabIndex={-1}>Submit feedback</h3><span>Informational only</span></div>
      <p className="review-submission-guidance">Add comments on the map or beside the content you’re reviewing. They will be included here when you submit.</p>
      <label>Reviewer name<input value={author} onChange={(event) => setAuthor(event.target.value)} placeholder="Your name or agent label" /></label>
      <label>Conclusion<select value={verdict} onChange={(event) => setVerdict(event.target.value as ReviewSubmissionSummary['verdict'])}><option value="comment">Comment</option><option value="approve">Approve</option><option value="request_changes">Request changes</option></select></label>
      <label>Review summary <span className="field-optional">Optional</span><textarea rows={4} value={body} onChange={(event) => setBody(event.target.value)} /><span className="field-hint">Summarize the reason for your conclusion.</span></label>
      {comments.length > 0 && <section className="review-draft-summary" aria-label="Comments to submit">
        <h4>Comments to submit <span>{comments.length}</span></h4>
        <ol className="review-comment-drafts">{comments.map(comment => <li key={comment.id}>
          <span>{anchorLabel(comment.anchor)}</span><p>{comment.body}</p>
          <div><button className="text-action" type="button" onClick={() => onEdit(comment)}>Edit</button>
            <button className="text-action" type="button" onClick={() => onRemove(comment.id)}>Remove</button></div>
          {renderEditor(`review-summary:${comment.id}`)}
        </li>)}</ol>
      </section>}
      {submitReason && <p className="field-hint" id={submitReasonID}>{submitReason}</p>}
      <button className="inline-action" type="button" aria-describedby={submitReason ? submitReasonID : undefined} disabled={busy || !valid || unfinishedComment} onClick={() => onSubmit({ author, verdict, body, comments: comments.map(({ body: commentBody, anchor }) => ({ body: commentBody, anchor })) })}>{busy ? 'Submitting…' : 'Submit review'}</button>
      <p className="field-hint">This records feedback on this exact version. It does not update Architecture.</p>
    </section>
  )
}

function ReviewContext({
  side,
  component,
  components,
  diagram,
  focus,
  onClear,
  onComment,
  comments,
  renderCommentEditor,
  onOpenAnnotation,
}: {
  side: ReviewSide
  component?: AuthoringComponent
  components: AuthoringComponent[]
  diagram?: DiagramProjection
  focus?: ReviewFocus | null
  onClear: () => void
  onComment?: (target: ReviewCommentTarget) => void
  comments?: ReviewSubmissionComment[]
  renderCommentEditor?: (contextKey: string) => ReactNode
  onOpenAnnotation?: (group: ReviewAnnotationGroup) => void
}) {
  const titles = new Map(components.map((candidate) => [candidate.id, candidate.title]))
  const exactSide = reviewSideValue(side)
  const openAnnotation = (group?: ReviewAnnotationGroup) => group && onOpenAnnotation?.(group)
  if (focus?.kind === 'diagram') {
    const contextKey = `review-context:${focus.key}`
    const focusSide = focus.reviewSide ? reviewSideValue(focus.reviewSide) : exactSide
    const anchor: ReviewAnchor = focus.componentID && focus.role
      ? { kind: 'composition', side: focusSide, diagram_id: focus.diagramID, component_id: focus.componentID,
        aspect: focus.compositionAspect ?? focus.role, ...(focus.compositionAspect === 'detail' && focus.detailDiagramID ? { detail_diagram_id: focus.detailDiagramID } : {}) }
      : { kind: 'diagram', side: focusSide, diagram_id: focus.diagramID }
    const annotations = focus.componentID
      ? compositionAnnotation(comments, focusSide, focus.diagramID, focus.componentID, focus.title)
      : diagramAnnotation(comments, focusSide, focus.diagramID, focus.title)
    return (
      <section className="review-context" aria-label="Review context">
        <div className="review-context-heading"><div><p className="eyebrow">Diagram composition</p><h3>{focus.title}</h3></div><button className="text-action" type="button" onClick={onClear}>Clear focus</button></div>
        <p>{focus.description ?? (focus.status === 'added' ? 'This diagram is added with the changes.' : focus.status === 'title_changed' ? 'This diagram title changes.' : 'A component placement changes in this diagram.')}</p>
        <div className="context-comment-actions">{onComment && <button className="text-action" type="button" onClick={() => onComment({ contextKey, label: focus.title, anchor })}>{focus.componentID ? 'Comment on this change' : 'Comment on diagram'}</button>}{annotations && <AnnotationMarker group={annotations} onToggle={() => openAnnotation(annotations)} />}</div>
        {renderCommentEditor?.(contextKey)}
      </section>
    )
  }
  if (focus?.kind === 'relationship') {
    const contextKey = `review-context:${focus.key}`
    const focusSide = focus.review_side ? reviewSideValue(focus.review_side) : exactSide
    const anchor: ReviewAnchor = { kind: 'relationship', side: focusSide, source_component_id: focus.source_id,
      target_component_id: focus.target_id, label: focus.label, occurrence: focus.occurrence }
    const annotations = relationshipAnnotation(comments, focusSide, focus.source_id, focus.target_id, focus.label, focus.occurrence, 'Relationship')
    return (
      <section className="review-context" aria-label="Review context">
        <div className="review-context-heading">
          <div><p className="eyebrow">{relationshipStatusText(focus.status)} relationship</p><h3>Relationship</h3></div>
          <button className="text-action" type="button" onClick={onClear}>Clear focus</button>
        </div>
        <p className={`review-relationship-summary review-${focus.status}`}>
          <span>{focus.source_title}</span><strong>{focus.label}</strong><span>{focus.target_title}</span>
        </p>
        <div className="context-comment-actions">{onComment && <button className="text-action" type="button" onClick={() => onComment({ contextKey, label: `${focus.source_title} — ${focus.label} → ${focus.target_title}`, anchor })}>Add comment</button>}{annotations && <AnnotationMarker group={annotations} onToggle={() => openAnnotation(annotations)} />}</div>
        {renderCommentEditor?.(contextKey)}
      </section>
    )
  }
  if (!component) {
    return (
      <section className="review-context review-context-empty" aria-label="Review context">
        <p className="eyebrow">{side === 'with' ? 'With changes' : 'Before changes'}</p>
        <h3>Select a change</h3>
        <p>Choose a component or relationship to inspect its documentation and exact diff.</p>
      </section>
    )
  }
  const contextKey = `review-context:${exactSide}:${diagram?.id ?? ''}:${component.id}`
  const componentAnnotations = component ? componentAnnotation(comments, exactSide, component.id, component.title) : undefined
  const relationshipOccurrences = new Map<string, number>()
  return (
    <section className="review-context" aria-label="Review context">
      <div className="review-context-heading">
        <div>
          <p className="eyebrow">{focus?.kind === 'component' ? componentStatusText(focus.status) : side === 'with' ? 'With changes' : 'Before changes'}</p>
          <h3>{component.title}</h3>
        </div>
        <button className="text-action" type="button" onClick={onClear}>Clear focus</button>
      </div>
      <div className="context-comment-actions">
        {onComment && <button className="text-action" type="button" onClick={() => onComment({ contextKey, label: component.title, anchor: { kind: 'component', side: exactSide, component_id: component.id } })}>Comment on component</button>}
        {onComment && component.markdown_source && <button className="text-action" type="button" onClick={() => onComment({ contextKey, label: `${component.title} lines`, anchor: { kind: 'component_markdown', side: exactSide, component_id: component.id }, source: component.markdown_source })}>Comment on lines</button>}
        {componentAnnotations && <AnnotationMarker group={componentAnnotations} onToggle={() => openAnnotation(componentAnnotations)} />}
      </div>
      {renderCommentEditor?.(contextKey)}
      <MarkdownBody source={component.description} />
      {component.relationships.length > 0 && (
        <details className="review-relationships">
          <summary>Outgoing relationships ({component.relationships.length})</summary>
          <ul>{component.relationships.map((relationship, index) => {
            const fact = `${component.id}\u0000${relationship.target_id}\u0000${relationship.label}`
            const occurrence = (relationshipOccurrences.get(fact) ?? 0) + 1
            relationshipOccurrences.set(fact, occurrence)
            const targetTitle = titles.get(relationship.target_id) ?? 'Component unavailable'
            const anchor: ReviewAnchor = { kind: 'relationship', side: exactSide, source_component_id: component.id, target_component_id: relationship.target_id, label: relationship.label, occurrence }
            const annotations = relationshipAnnotation(comments, exactSide, component.id, relationship.target_id, relationship.label, occurrence, `${component.title} — ${relationship.label} → ${targetTitle}`)
            return <li key={relationship.projection_key ?? `${relationship.target_id}:${index}`}><span>{relationship.label}</span> → <span>{targetTitle}</span><span className="relationship-comment-actions">{onComment && <button className="text-action" type="button" onClick={() => onComment({ contextKey, label: `${component.title} — ${relationship.label} → ${targetTitle}`, anchor })}>Add comment</button>}{annotations && <AnnotationMarker group={annotations} onToggle={() => openAnnotation(annotations)} />}</span></li>
          })}</ul>
        </details>
      )}
    </section>
  )
}

function DiscardChangesDialog({ busy, onCancel, onDiscard }: { busy: boolean; onCancel: () => void; onDiscard: () => void }) {
  return (
    <div className="navigation-guard" role="dialog" aria-modal="true" aria-labelledby="discard-heading">
      <div className="discard-confirmation">
        <h2 id="discard-heading">Delete this proposal?</h2>
        <p>This permanently removes this whole open proposal. Accepted Architecture and every other proposal stay as they are.</p>
        <div className="button-group">
          <button className="secondary-action" type="button" onClick={onCancel}>Keep proposal</button>
          <button className="destructive-action" type="button" disabled={busy} onClick={onDiscard}>Delete proposal</button>
        </div>
      </div>
    </div>
  )
}

async function postJSON(path: string, value: unknown) {
  return fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(value),
  })
}

const errorMessages: Record<ErrorCode, string> = {
  name_required: 'Enter a project name.',
  project_not_found: 'That project does not exist.',
  catalog_conflict: 'Two projects use the same address. Resolve the catalog conflict before opening either one.',
  catalog_unavailable: 'WorkBraid could not read the project catalog.',
  origin_mismatch: 'Open WorkBraid at the address printed in the terminal.',
  lookup_failed: "WorkBraid couldn't look that up. Try again.",
  project_create_failed: 'WorkBraid could not create the project. Try again.',
  architecture_unavailable: 'WorkBraid could not open this architecture.',
  architecture_invalid: 'WorkBraid could not open this architecture.',
  architecture_unsupported: 'This architecture is not supported yet.',
}

function projectRoutePath(slug: string) {
  return `/projects/${encodeURIComponent(slug)}`
}

function proposalRoutePath(slug: string, changeSetID: string) {
  return `${projectRoutePath(slug)}/proposals/${encodeURIComponent(changeSetID)}`
}

function reviewRoutePath(slug: string, changeSetID: string) {
  return `${proposalRoutePath(slug, changeSetID)}/review`
}

function submittedReviewRoutePath(slug: string, changeSetID: string, reviewID: string) {
  return `${proposalRoutePath(slug, changeSetID)}/reviews/${encodeURIComponent(reviewID)}`
}

function decodeProjectRoute(pathname: string): { slug: string; proposalChangeSetID?: string; reviewChangeSetID?: string; submittedReviewID?: string } | undefined {
  const submittedMatch = /^\/projects\/([^/]+)\/proposals\/([^/]+)\/reviews\/([^/]+)\/?$/.exec(pathname)
  const reviewMatch = /^\/projects\/([^/]+)\/proposals\/([^/]+)\/review\/?$/.exec(pathname)
  const proposalMatch = /^\/projects\/([^/]+)\/proposals\/([^/]+)\/?$/.exec(pathname)
  const projectMatch = /^\/projects\/([^/]+)\/?$/.exec(pathname)
  try {
    if (submittedMatch) return { slug: decodeURIComponent(submittedMatch[1]), reviewChangeSetID: decodeURIComponent(submittedMatch[2]), submittedReviewID: decodeURIComponent(submittedMatch[3]) }
    if (reviewMatch) return { slug: decodeURIComponent(reviewMatch[1]), reviewChangeSetID: decodeURIComponent(reviewMatch[2]) }
    if (proposalMatch) return { slug: decodeURIComponent(proposalMatch[1]), proposalChangeSetID: decodeURIComponent(proposalMatch[2]) }
    if (projectMatch) return { slug: decodeURIComponent(projectMatch[1]) }
  } catch {
    return undefined
  }
  return undefined
}

function messageForError(code?: string) {
  if (code && Object.prototype.hasOwnProperty.call(errorMessages, code)) {
    return errorMessages[code as ErrorCode]
  }
  return errorMessages.lookup_failed
}

function messageForAuthoringError(code?: string) {
  if (code === 'origin_mismatch') return 'Open WorkBraid at the address printed in the terminal.'
  if (code === 'changes_elsewhere') return 'Changes are already in progress for another architecture.'
  if (code === 'component_not_found') return 'That component is no longer available to edit.'
  if (code === 'architecture_not_open') return 'Open the project again, then try your change.'
  if (code === 'changes_unavailable') return 'Changes are not available for this architecture yet.'
  return "WorkBraid couldn't keep that change. Try again."
}

function messageForReviewBlocker(code?: string) {
  if (code === 'title_required') return 'Add a title to the untitled component before updating architecture.'
  if (code === 'title_one_line') return 'Use a one-line component title before updating architecture.'
  if (code === 'relationship_label_required') return 'Add a label to this relationship.'
  if (code === 'relationship_target_required') return 'Choose a component for this relationship.'
  if (code === 'diagram_title_required') return 'Add a title to this diagram.'
  if (code === 'diagram_cycle') return 'Move this component somewhere outside its own detail diagrams.'
  if (code === 'diagram_home_invalid') return 'Choose a valid place for this component.'
  return 'Correct the component changes before updating architecture.'
}

function messageForReadOnlyReviewBlocker(code?: string) {
  if (code === 'relationship_label_required') return 'This relationship has no label.'
  if (code === 'relationship_target_required') return 'This relationship has no component selected.'
  return 'This component change is incomplete.'
}

function messageForArchitectureAction(code?: string) {
  if (code === 'catalog_conflict') return 'Two projects now use the same address. Resolve the catalog conflict before refreshing this project.'
  if (code === 'architecture_stale') return 'These changes are out of date because the architecture changed.'
  if (code === 'review_changed') return 'The changes were edited after this review. Review them again before updating architecture.'
  if (code === 'updated_reload') return 'Architecture was updated, but this page could not refresh. Open the project again.'
  if (code === 'update_uncertain') return 'WorkBraid could not confirm the current architecture. Open the project again.'
  if (code === 'update_failed') return "WorkBraid couldn't update the architecture. Try again."
  if (code === 'review_failed') return "WorkBraid couldn't prepare these changes for review. Try again."
  if (code === 'refresh_failed') return "WorkBraid couldn't check for architecture changes. Try Refresh again."
  if (code === 'refresh_changed') return 'Architecture changed again while WorkBraid was refreshing. Refresh once more.'
  if (code === 'refresh_unsupported') return 'The current architecture uses features this version of WorkBraid cannot open.'
  if (code === 'refresh_invalid') return 'The current architecture could not be read. This earlier view is read-only.'
  if (code === 'refresh_unavailable') return 'The current architecture could not be found. This earlier view is read-only.'
  return "WorkBraid couldn't complete that action. Try again."
}
