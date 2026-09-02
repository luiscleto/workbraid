import { FormEvent, useCallback, useEffect, useRef, useState } from 'react'
import {
  ArchitectureMap,
  MapComponent,
  ReviewMapComponentChange,
  ReviewMapRelationshipChange,
  ReviewRelationshipSelection,
} from './ArchitectureMap'
import { MarkdownBody } from './MarkdownBody'
import { RawDiff } from './RawDiff'

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
  stale?: boolean
  parent_diff?: string
  action_error?: string
}

type DiagramProjection = {
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
  component_id: string
  role: 'home' | 'reference'
  detail_diagram_id?: string
  detail_diagram_title?: string
}

type DiagramBoundary = {
  key: string
  component_id: string
  title: string
  context?: string
  home_diagram_id: string
  home_diagram_title: string
}

type DiagramRelationship = {
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
  filename: string
  relationships: { target_id: string; label: string; projection_key?: string }[]
}

type PendingComponent = AuthoringComponent & { new: boolean }

type ChangesInProgress = {
  base_revision?: string
  generation?: number
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

type RelationshipValue = { target_id: string; label: string }
type RelationshipRow = RelationshipValue & { rowKey: string }

type ChangeReview = {
  diff: string
  base_revision: string
  candidate_tree: string
  generation: number
  before: ReviewSnapshot
  with_changes: ReviewSnapshot
  comparison: {
    components: ReviewMapComponentChange[]
    relationships: ReviewMapRelationshipChange[]
    diagrams?: { diagram_id: string; title: string; status: 'added' | 'title_changed'; path: string }[]
    appearances?: { diagram_id: string; component_id: string; role: 'home' | 'reference'; status: 'added' | 'removed' | 'detail_changed'; path: string }[]
  }
}

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
  | { kind: 'diagram'; key: string; diagramID: string; title: string; path: string; status: 'added' | 'title_changed' | 'appearance_changed' }

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

type DiagramEditor =
  | { kind: 'detail'; componentID: string; title: string; initialTitle: string; invalid?: boolean }
  | { kind: 'title'; diagramID: string; title: string; initialTitle: string; invalid?: boolean }
  | { kind: 'move'; componentID: string; componentTitle: string; diagramID: string; currentDiagramTitle: string }

type WorkspaceTask = 'documentation' | 'changes' | 'empty'

type NavigationIntent =
  | { kind: 'component'; id: string }
  | { kind: 'diagram'; id: string; focusComponentID?: string }
  | { kind: 'changes' }
  | { kind: 'add' }
  | { kind: 'edit-diagram-title'; id: string; title: string }
  | { kind: 'open-another' }
  | { kind: 'route'; slug?: string }
  | { kind: 'refresh' }
  | { kind: 'clear' }
  | { kind: 'review-result'; result: ArchitectureResult }

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
      title: component.title,
      filename: component.filename,
      node_kind: appearance.role,
      relationships: [],
    })
  }
  for (const boundary of diagram.boundaries) {
    nodes.set(boundary.key, {
      id: boundary.key,
      component_id: boundary.component_id,
      title: boundary.title,
      node_kind: 'boundary',
      boundary_home_title: boundary.home_diagram_title,
      relationships: [],
    })
  }
  for (const relationship of diagram.relationships) {
    nodes.get(relationship.source_node_key)?.relationships.push({
      target_id: relationship.target_node_key,
      label: relationship.label,
      projection_key: relationship.key,
    })
  }
  return [...nodes.values()]
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
  return status === 'added' ? 'Added' : 'Removed'
}

export function App() {
  const [projectName, setProjectName] = useState('')
  const [state, setState] = useState<ViewState>({ kind: 'looking' })
  const [editor, setEditor] = useState<ComponentEditor | null>(null)
  const [diagramEditor, setDiagramEditor] = useState<DiagramEditor | null>(null)
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

  const enterWorkspace = useCallback((result: ArchitectureResult, task?: WorkspaceTask) => {
    setState({ kind: 'ready', value: result })
    if (result.format_version === 2) {
      const selectedDiagram = result.diagrams?.find((diagram) => diagram.id === selectedDiagramID)
        ?? result.diagrams?.find((diagram) => diagram.id === result.root_diagram_id)
      setSelectedDiagramID(selectedDiagram?.id)
      setSelectedComponentID((current) => selectedDiagram?.appearances.some((appearance) => appearance.component_id === current)
        ? current
        : selectedDiagram?.appearances[0]?.component_id)
    } else {
      setSelectedDiagramID(undefined)
      setSelectedComponentID((current) => result.components?.some((component) => component.id === current) ? current : result.components?.[0]?.id)
    }
    setWorkspaceTask(task ?? (result.changes ? 'changes' : result.components?.length ? 'documentation' : 'empty'))
  }, [selectedDiagramID])

  const readyResult = state.kind === 'ready' ? state.value : undefined
  const currentReview = readyResult?.stale || readyResult?.changes?.stale
    ? undefined
    : readyResult?.changes?.review
  const reviewIdentity = currentReview ? `${currentReview.base_revision}:${currentReview.candidate_tree}:${currentReview.generation}` : ''
  const editorDirty = editor !== null && (
    editor.title !== editor.initialTitle || editor.description !== editor.initialDescription ||
    !sameRelationships(relationshipValues(editor.relationships), editor.initialRelationships)
  )
  const diagramEditorDirty = diagramEditor !== null && (diagramEditor.kind === 'move'
    ? diagramEditor.diagramID !== ''
    : diagramEditor.title !== diagramEditor.initialTitle)
  const editorDirtyRef = useRef(editorDirty)
  editorDirtyRef.current = editorDirty || diagramEditorDirty
  const stateRef = useRef(state)
  stateRef.current = state

  useEffect(() => {
    if (!currentReview) {
      setReviewFocus(null)
      setReviewSelectionCleared(false)
      setReviewVisible(false)
      return
    }
    setReviewSide('with')
    setReviewFocus(null)
    setReviewSelectionCleared(false)
    setReviewVisible(true)
    const initialDiagram = currentReview.with_changes.format_version === 2
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
    const heldReview = state.value.stale || state.value.changes?.stale
      ? undefined
      : state.value.changes?.review
    const review = reviewVisible ? heldReview : undefined
    if (review) {
      const projection = reviewSide === 'with' ? review.with_changes : review.before
      if (reviewSelectionCleared) {
        if (selectedComponentID) setSelectedComponentID(undefined)
        return
      }
      if (projection.format_version === 2) {
        const diagram = projection.diagrams?.find((candidate) => candidate.id === selectedDiagramID)
          ?? projection.diagrams?.find((candidate) => candidate.id === projection.root_diagram_id)
        const active = diagram ? componentsForDiagram(projection, diagram) : []
        if (reviewFocus?.kind === 'relationship' && !active.some((component) => component.id === reviewFocus.source_id)) {
          if (selectedComponentID) setSelectedComponentID(undefined)
          return
        }
        if (selectedComponentID && active.some((component) => component.id === selectedComponentID)) return
        setSelectedComponentID(active[0]?.id)
        return
      }
      const active = projection.components
      if (selectedComponentID && active.some((component) => component.id === selectedComponentID)) return
      if (selectedComponentID) setSelectedComponentID(undefined)
      return
    }
    if (state.value.format_version === 2) {
      const diagram = state.value.diagrams?.find((candidate) => candidate.id === selectedDiagramID)
        ?? state.value.diagrams?.find((candidate) => candidate.id === state.value.root_diagram_id)
      if (diagram && diagram.id !== selectedDiagramID && !(heldReview && !reviewVisible)) {
        setSelectedDiagramID(diagram.id)
      }
      if (workspaceTask === 'empty') return
      if (selectedComponentID && diagram?.appearances.some((appearance) => appearance.component_id === selectedComponentID)) return
      setSelectedComponentID(diagram?.appearances[0]?.component_id)
      return
    }
    if (workspaceTask === 'empty' && state.value.components?.length) return
    if (selectedComponentID && state.value.components?.some((component) => component.id === selectedComponentID)) return
    setSelectedComponentID(state.value.components?.[0]?.id)
  }, [state, selectedComponentID, selectedDiagramID, reviewFocus, reviewSelectionCleared, reviewSide, reviewVisible, workspaceTask])

  useEffect(() => {
    const slug = decodeProjectSlug(window.location.pathname)
    if (slug) void openProject(slug, true)
    else void loadCatalog()
    const restoreHistoryRoute = () => {
      const targetSlug = decodeProjectSlug(window.location.pathname)
      const current = stateRef.current
      if (current.kind === 'ready' && targetSlug === current.value.project_slug) return
      if (editorDirtyRef.current && current.kind === 'ready') {
        window.history.replaceState({}, '', `/projects/${encodeURIComponent(current.value.project_slug)}`)
        setNavigationIntent({ kind: 'route', slug: targetSlug })
        return
      }
      void restoreRoute(targetSlug)
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

  async function openProject(slug: string, replaceRoute = false) {
    setState({ kind: 'looking' })
    setArchitectureNotice('')
    try {
      const response = await postJSON('/api/projects/open', { project_slug: slug })
      const result = await response.json() as ArchitectureResult | ErrorPayload
      if (!response.ok && 'state' in result) {
        window.history.replaceState({}, '', `/projects/${encodeURIComponent(result.project_slug)}`)
        enterWorkspace({ ...result, action_error: undefined }, 'changes')
        setArchitectureNotice('Keep working here or discard these changes before opening another project.')
        return
      }
      if (!response.ok || !('state' in result)) {
        if ('code' in result && result.code === 'project_not_found') setState({ kind: 'not-found', slug })
        else setState({ kind: 'catalog-error', message: messageForError('code' in result ? result.code : undefined) })
        return
      }
      window.history[replaceRoute ? 'replaceState' : 'pushState']({}, '', `/projects/${encodeURIComponent(result.project_slug)}`)
      enterWorkspace(result)
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
      enterWorkspace(result)
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
      enterWorkspace(payload, 'changes')
      setEditor(null)
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
        expected_revision: result.revision,
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if ('state' in payload) {
        setAcceptanceUnknown(false)
        if (payload.changes?.review && editorDirtyRef.current) {
          setNavigationIntent({ kind: 'review-result', result: payload })
        } else {
          enterWorkspace(payload, 'changes')
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

  async function submitDiagramChange(event: FormEvent<HTMLFormElement>, result: ArchitectureResult) {
    event.preventDefault()
    if (!diagramEditor) return
    const endpoint = diagramEditor.kind === 'detail'
      ? '/api/architecture/diagrams/detail'
      : diagramEditor.kind === 'title'
        ? '/api/architecture/diagrams/title'
        : '/api/architecture/components/move-home'
    setAuthoringError('')
    try {
      const response = await postJSON(endpoint, {
        project_slug: result.project_slug,
        store_id: result.store_id,
        expected_revision: result.revision,
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
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
      setDiagramEditor(null)
      enterWorkspace(payload, 'changes')
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
      enterWorkspace(payload, 'changes')
    } catch {
      setArchitectureNotice("WorkBraid couldn't keep that diagram change. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function updateArchitecture(result: ArchitectureResult) {
    const review = result.changes?.review
    if (!review) return
    setArchitectureBusy(true)
    setArchitectureNotice('')
    setAcceptanceUnknown(true)
    enterWorkspace({ ...result, changes: result.changes ? { ...result.changes, review: undefined } : undefined }, 'changes')
    try {
      const response = await postJSON('/api/architecture/accept', {
        project_slug: result.project_slug,
        store_id: result.store_id,
        base_revision: review.base_revision,
        candidate_tree: review.candidate_tree,
        generation: review.generation,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if ('state' in payload) {
        setAcceptanceUnknown(false)
        enterWorkspace(payload, payload.changes ? 'changes' : 'documentation')
      } else {
        setArchitectureNotice('WorkBraid could not confirm what happened. Open this project again to check its current architecture.')
      }
    } catch {
      setArchitectureNotice('WorkBraid could not confirm what happened. Open this project again to check its current architecture.')
    } finally {
      setArchitectureBusy(false)
    }
  }

  const busy = state.kind === 'looking'

  function requestNavigation(intent: NavigationIntent) {
    if (editorDirty || diagramEditorDirty) {
      setNavigationIntent(intent)
      return
    }
    void performNavigation(intent)
  }

  async function performNavigation(intent: NavigationIntent) {
    setNavigationIntent(null)
    setEditor(null)
    setDiagramEditor(null)
    setAuthoringError('')
    setArchitectureNotice('')
    if (intent.kind === 'component') {
      setSelectedComponentID(intent.id)
      setWorkspaceTask('documentation')
      return
    }
    if (intent.kind === 'diagram') {
      const projection = state.kind === 'ready' ? state.value : undefined
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
    if (intent.kind === 'edit-diagram-title') {
      setDiagramEditor({ kind: 'title', diagramID: intent.id, title: intent.title, initialTitle: intent.title })
      return
    }
    if (intent.kind === 'review-result') {
      setAcceptanceUnknown(false)
      enterWorkspace(intent.result, 'changes')
      return
    }
    if (intent.kind === 'route') {
      await restoreRoute(intent.slug)
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
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if ('state' in payload) {
        enterWorkspace({ ...payload, action_error: undefined }, 'changes')
        setArchitectureNotice('Keep working here or discard these changes before opening another project.')
      } else {
        setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
      }
    } catch {
      setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
    } finally {
      setArchitectureBusy(false)
    }
  }

  async function restoreRoute(slug?: string) {
    const current = stateRef.current
    if (slug) {
      await openProject(slug, true)
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
      const payload = await response.json() as ArchitectureResult | ErrorPayload
      if ('state' in payload) {
        window.history.replaceState({}, '', `/projects/${encodeURIComponent(payload.project_slug)}`)
        enterWorkspace({ ...payload, action_error: undefined }, 'changes')
        setArchitectureNotice('Keep working here or discard these changes before leaving this project.')
      } else {
        window.history.replaceState({}, '', `/projects/${encodeURIComponent(current.value.project_slug)}`)
        setArchitectureNotice("WorkBraid couldn't leave this project. Try again.")
      }
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
        pending_generation_observed: true,
        expected_pending_generation: result.changes?.generation ?? null,
      })
      const payload = (await response.json()) as ArchitectureResult | ErrorPayload
      if (!response.ok || !('state' in payload)) {
        setArchitectureNotice("WorkBraid couldn't discard these changes. Try again.")
        return
      }
      setDiscardConfirming(false)
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
      if (payload.project_slug !== result.project_slug) window.history.replaceState({}, '', `/projects/${encodeURIComponent(payload.project_slug)}`)
      enterWorkspace(payload, nextTask)
      setArchitectureNotice(notice)
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
    const diagramProjection = activeProjection ?? result
    const candidateOnlyDiagramBefore = Boolean(review && reviewSide === 'before' && selectedDiagramID &&
      review.with_changes.diagrams?.some((diagram) => diagram.id === selectedDiagramID) &&
      !review.before.diagrams?.some((diagram) => diagram.id === selectedDiagramID))
    const candidateFallbackDiagramID = candidateOnlyDiagramBefore && review
      ? [...(review.with_changes.diagrams?.find((diagram) => diagram.id === selectedDiagramID)?.breadcrumbs ?? [])]
        .reverse()
        .find((breadcrumb) => review.before.diagrams?.some((diagram) => diagram.id === breadcrumb.id))?.id
      : undefined
    const activeDiagram = diagramProjection.format_version === 2
      ? diagramProjection.diagrams?.find((diagram) => diagram.id === (candidateFallbackDiagramID ?? selectedDiagramID))
        ?? diagramProjection.diagrams?.find((diagram) => diagram.id === diagramProjection.root_diagram_id)
      : undefined
    const authoringAvailable = !result.stale && !result.changes?.stale && !acceptanceUnknown
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
    const activeComponents = activeProjection?.format_version === 2
      ? activeDiagramComponents ?? []
      : activeProjection?.components ?? activeDiagramComponents ?? result.components ?? []
    const diagramMapComponents = activeDiagram ? mapComponentsForDiagram(diagramProjection, activeDiagram) : undefined
    const mapComponents = diagramMapComponents ?? activeComponents
    const selected = activeComponents.find((component) => component.id === selectedComponentID)
    const selectedAppearance = activeDiagram?.appearances.find((appearance) => appearance.component_id === selectedComponentID)
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
    const layoutComponentIDs = review
      ? [...new Set([...review.before.components, ...review.with_changes.components].map((component) => component.id))]
      : undefined
    const selectComponent = (id: string) => {
      if (!review) {
        requestNavigation({ kind: 'component', id })
        return
      }
      const component = activeComponents.find((candidate) => candidate.id === id)
      if (!component) return
      const change = componentReviewStatus.get(id)
      setReviewSelectionCleared(false)
      setSelectedComponentID(id)
      setReviewFocus({
        kind: 'component', key: `component:${id}`, componentID: id, title: component.title,
        path: change?.path ?? canonicalReviewPath(component), status: change?.status ?? 'unchanged',
      })
    }
    const selectDiagram = (diagramID: string, focusComponentID?: string) => {
      if (!review) {
        requestNavigation({ kind: 'diagram', id: diagramID, focusComponentID })
        return
      }
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
    }
    const selectMapNode = (id: string) => {
      if (!activeDiagram) {
        selectComponent(id)
        return
      }
      const boundary = activeDiagram.boundaries.find((candidate) => candidate.key === id)
      if (boundary) {
        selectDiagram(boundary.home_diagram_id, boundary.component_id)
        return
      }
      selectComponent(id)
    }
    const selectRelationship = (relationship: ReviewRelationshipSelection) => {
      const relationshipSide = relationship.review_side ?? reviewSide
      const relationshipProjection = relationshipSide === 'with' ? review?.with_changes : review?.before
      const relationshipDiagram = relationshipProjection?.format_version === 2
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
    }
    const switchReviewSide = (side: ReviewSide) => {
      if (!review || side === reviewSide) return
      const nextProjection = side === 'with' ? review.with_changes : review.before
      const nextDiagram = nextProjection.format_version === 2
        ? nextProjection.diagrams?.find((diagram) => diagram.id === selectedDiagramID)
          ?? nextProjection.diagrams?.find((diagram) => diagram.id === nextProjection.root_diagram_id)
        : undefined
      const nextComponents = nextDiagram ? componentsForDiagram(nextProjection, nextDiagram) : nextProjection.components
      setReviewSelectionCleared(false)
      setReviewSide(side)
      setReviewFocus(null)
      setSelectedComponentID((current) => current && nextComponents.some((component) => component.id === current) ? current : undefined)
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
    return (
      <main className="workspace-shell">
        <header className="application-frame">
          <div>
            <p className="eyebrow">WorkBraid</p>
            <p className="workspace-context"><strong>{result.project_name}</strong><span>Architecture</span></p>
          </div>
          <div className="frame-actions">
            {result.changes ? (
              <button className="text-action" type="button" onClick={() => requestNavigation({ kind: 'changes' })}>
                Changes in progress {result.changes.components.length > 0 && <span className="change-count">{result.changes.components.length}</span>}
              </button>
            ) : null}
            <button className="text-action" type="button" disabled={architectureBusy || acceptanceUnknown} onClick={() => requestNavigation({ kind: 'refresh' })}>
              Refresh
            </button>
            <button className="text-action" type="button" disabled={architectureBusy || acceptanceUnknown} onClick={() => requestNavigation({ kind: 'open-another' })}>
              Open another project
            </button>
          </div>
        </header>
        {result.stale && <div className="stale-banner" role="alert">The current architecture could not be loaded. This earlier view is read-only.</div>}
        {architectureNotice && (
          <div className="workspace-notice" role="alert">
            <span>{architectureNotice}</span>
            <button className="notice-dismiss" type="button" aria-label="Dismiss message" onClick={() => setArchitectureNotice('')}>×</button>
          </div>
        )}
        <div className={`architecture-workbench ${review ? 'reviewing' : ''}`}>
          <nav className="component-index" aria-label={diagramProjection.format_version === 2 ? 'Diagrams and components' : 'Components'}>
            {diagramProjection.format_version === 2 && (
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
                    return (
                      <li key={diagram.id}>
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
                  return (
                  <li key={component.id}>
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
                  </li>
                  )
                })}
              </ul>
            ) : <p className="index-empty">No components</p>}
            {!review && authoringAvailable && (
              <button className="index-add" type="button" onClick={() => requestNavigation({ kind: 'add' })}>Add component</button>
            )}
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
          <section className={`map-region ${activeDiagram ? 'has-diagram' : ''}`}>
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
              </nav>
            )}
            <ArchitectureMap
              revision={`${activeProjection?.revision ?? result.revision}${activeDiagram ? `:${activeDiagram.id}` : ''}`}
              components={mapComponents}
              selectedID={selectedComponentID}
              onSelect={selectMapNode}
              emptyMessage={activeDiagram ? 'This diagram has no components.' : undefined}
              {...(review ? {
                layoutComponentIDs,
                reviewSide,
                reviewComponents: review.comparison.components,
                reviewRelationships: review.comparison.relationships,
                reviewDiagramID: activeDiagram?.id,
                selectedRelationshipKey: reviewFocus?.kind === 'relationship' ? reviewFocus.key : undefined,
                onSelectRelationship: selectRelationship,
              } : {})}
              externalReferences={externalReferences}
            />
          </section>
          <aside className="working-pane" aria-label="Architecture task">
            {review && result.changes ? (
              <ChangesTask
                result={result}
                busy={architectureBusy}
                acceptanceUnknown={acceptanceUnknown}
                discardConfirming={discardConfirming}
                reviewSide={reviewSide}
                selectedReviewComponent={selected}
                reviewFocus={reviewFocus}
                onReviewSide={switchReviewSide}
                onContinueEditing={() => {
                  setReviewVisible(false)
                  setWorkspaceTask('changes')
                }}
                onClearReviewFocus={() => {
                  setReviewSelectionCleared(true)
                  setSelectedComponentID(undefined)
                  setReviewFocus(null)
                }}
                onFocusDiagram={(focus) => {
                  setReviewSelectionCleared(false)
                  setSelectedDiagramID(focus.diagramID)
                  setSelectedComponentID(undefined)
                  setReviewFocus(focus)
                }}
                onEdit={(component) => editPending(component, undefined, result.stale || result.changes?.stale)}
                onFixRelationship={(component) => editPending(component, {
                  position: result.changes?.validation_relationship_position ?? 0,
                  field: result.changes?.validation_relationship_field ?? 'target',
                })}
                onReview={() => reviewChanges(result)}
                onUpdate={() => updateArchitecture(result)}
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
                result={result}
                busy={architectureBusy}
                acceptanceUnknown={acceptanceUnknown}
                discardConfirming={discardConfirming}
                onReturnToReview={currentReview && !acceptanceUnknown ? () => setReviewVisible(true) : undefined}
                onEdit={(component) => editPending(component, undefined, result.stale || result.changes?.stale)}
                onFixRelationship={(component) => editPending(component, {
                  position: result.changes?.validation_relationship_position ?? 0,
                  field: result.changes?.validation_relationship_field ?? 'target',
                })}
                onAddComponent={result.format_version === 2 ? (diagramID) => addComponent(diagramID) : undefined}
                onCreateDetail={result.format_version === 2 ? (componentID) => setDiagramEditor({ kind: 'detail', componentID, title: '', initialTitle: '' }) : undefined}
                onEditDiagramTitle={result.format_version === 2 ? (diagramID, title, invalid) => setDiagramEditor({ kind: 'title', diagramID, title, initialTitle: title, invalid }) : undefined}
                onMoveHome={result.format_version === 2 ? beginHomeMove : undefined}
                onShowComponent={(diagramID, componentID) => changeReference(result, diagramID, componentID, true)}
                onStopShowing={(diagramID, componentID) => changeReference(result, diagramID, componentID, false)}
                onReview={() => reviewChanges(result)}
                onUpdate={() => updateArchitecture(result)}
                onBeginDiscard={() => setDiscardConfirming(true)}
                onCancelDiscard={() => setDiscardConfirming(false)}
                onDiscard={() => discardChanges(result)}
              />
            ) : selected ? (
              <article className="component-documentation">
                <div className="pane-heading pane-heading-with-action"><div><p className="eyebrow">Component</p><h2>{selected.title}</h2></div><button className="text-action" type="button" onClick={() => requestNavigation({ kind: 'clear' })}>Clear selection</button></div>
                <MarkdownBody source={selected.description} />
                {(authoringAvailable || selectedAppearance?.detail_diagram_id) && (
                  <div className="component-documentation-actions">
                    {authoringAvailable && <button className="inline-action" type="button" onClick={() => editAccepted(selected, result)}>Edit component</button>}
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
                            <button className="text-action diagram-composition-link" type="button" onClick={() => setDiagramEditor({ kind: 'detail', componentID: selected.id, title: '', initialTitle: '' })}>Create detail diagram</button>
                          )}
                          {selectedAppearance.role === 'home' && result.home_move_destinations?.find((destinations) => destinations.component_id === selected.id)?.diagram_ids.length ? <button className="text-action diagram-composition-link" type="button" onClick={() => beginHomeMove(selected.id)}>Change where {componentAuthoringLabel(compositionProjection.components, selected.id)} lives</button> : null}
                          {selectedAppearance.role === 'reference' && activeDiagram && <button className="text-action diagram-composition-link" type="button" onClick={() => changeReference(result, activeDiagram.id, selected.id, false)}>Stop showing here</button>}
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
            <details className="technical-details">
              <summary>Technical details</summary>
              <dl><dt>Project slug</dt><dd>{result.project_slug}</dd><dt>Revision</dt><dd>{result.revision}</dd></dl>
              {result.parent_diff && <div className="accepted-diff"><h3>Parent diff</h3><pre>{result.parent_diff}</pre></div>}
            </details>
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

function ChangesTask({
  result,
  busy,
  acceptanceUnknown,
  discardConfirming,
  reviewSide,
  selectedReviewComponent,
  reviewFocus,
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
  onBeginDiscard,
  onCancelDiscard,
  onDiscard,
}: {
  result: ArchitectureResult
  busy: boolean
  acceptanceUnknown: boolean
  discardConfirming: boolean
  reviewSide?: ReviewSide
  selectedReviewComponent?: AuthoringComponent
  reviewFocus?: ReviewFocus | null
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
  onBeginDiscard: () => void
  onCancelDiscard: () => void
  onDiscard: () => void
}) {
  const changes = result.changes
  if (!changes) return null
  const readOnly = Boolean(result.stale || changes.stale)
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
  const discardAction = !acceptanceUnknown
    ? <button className="discard-action" type="button" disabled={busy} onClick={onBeginDiscard}>Discard changes</button>
    : null
  if (changes.review && !readOnly && reviewSide && onReviewSide && onClearReviewFocus) {
    return (
      <section className="changes-in-progress review-workspace-pane" aria-labelledby="review-heading">
        <div className="review-heading-row">
          <div className="pane-heading"><p className="eyebrow">Architecture</p><h2 id="review-heading">Review changes</h2></div>
          <div className="review-side-toggle" role="group" aria-label="Review side">
            <button type="button" aria-pressed={reviewSide === 'with'} onClick={() => onReviewSide('with')}>With changes</button>
            <button type="button" aria-pressed={reviewSide === 'before'} onClick={() => onReviewSide('before')}>Before changes</button>
          </div>
        </div>
        <p className="review-introduction">Inspect the visual change and complete exact diff before updating the architecture.</p>
        {(changes.review.comparison.diagrams?.length || changes.review.comparison.appearances?.length) ? (
          <section className="diagram-review-summary" aria-label="Diagram changes">
            <h3>Diagram changes</h3>
            <ul>
              {changes.review.comparison.diagrams?.map((diagram) => <li key={`${diagram.diagram_id}:${diagram.status}`}><button className="text-action" type="button" onClick={() => onFocusDiagram?.({ kind: 'diagram', key: `diagram:${diagram.diagram_id}`, diagramID: diagram.diagram_id, title: diagram.title, path: diagram.path, status: diagram.status })}><strong>{diagram.title}</strong> {diagram.status === 'added' ? 'added' : 'title changed'}</button></li>)}
              {changes.review.comparison.appearances?.map((appearance, index) => {
                const projection = changes.review?.with_changes.components.find((component) => component.id === appearance.component_id)
                  ?? changes.review?.before.components.find((component) => component.id === appearance.component_id)
                const preferredDiagrams = appearance.status === 'removed' ? changes.review?.before.diagrams : changes.review?.with_changes.diagrams
                const fallbackDiagrams = appearance.status === 'removed' ? changes.review?.with_changes.diagrams : changes.review?.before.diagrams
                const diagram = preferredDiagrams?.find((candidate) => candidate.id === appearance.diagram_id)
                  ?? fallbackDiagrams?.find((candidate) => candidate.id === appearance.diagram_id)
                const description = appearanceReviewDescription(appearance.role, appearance.status, diagram?.title ?? 'Diagram')
                return <li key={`${appearance.diagram_id}:${appearance.component_id}:${appearance.status}:${index}`}><button className="text-action" type="button" onClick={() => onFocusDiagram?.({ kind: 'diagram', key: `appearance:${appearance.diagram_id}:${appearance.component_id}:${index}`, diagramID: appearance.diagram_id, title: diagram?.title ?? 'Diagram', path: appearance.path, status: 'appearance_changed' })}><strong>{projection?.title ?? 'Component'}</strong> {description}</button></li>
              })}
            </ul>
          </section>
        ) : null}
        <ReviewContext
          side={reviewSide}
          component={selectedReviewComponent}
          components={reviewSide === 'with' ? changes.review.with_changes.components : changes.review.before.components}
          focus={reviewFocus}
          onClear={onClearReviewFocus}
        />
        <section className="raw-diff-region" aria-labelledby="raw-diff-heading">
          <div className="review-section-heading"><h3 id="raw-diff-heading">Complete change</h3><span>Raw unified diff</span></div>
          <RawDiff
            diff={changes.review.diff}
            focusPath={reviewFocus?.path}
            focusToken={reviewFocus?.key}
          />
        </section>
        <details className="review-details">
          <summary>Review details</summary>
          <dl>
            <dt>Base revision</dt><dd>{changes.review.base_revision}</dd>
            <dt>Candidate tree</dt><dd>{changes.review.candidate_tree}</dd>
            <dt>Change version</dt><dd>{changes.review.generation}</dd>
          </dl>
        </details>
        <div className="change-actions">
          <button className="inline-action" type="button" disabled={busy} onClick={onUpdate}>{busy ? 'Updating…' : 'Update architecture'}</button>
          {onContinueEditing && <button className="secondary-action" type="button" disabled={busy} onClick={onContinueEditing}>Continue editing</button>}
          {discardAction}
        </div>
        {discardConfirming && <DiscardChangesDialog busy={busy} onCancel={onCancelDiscard} onDiscard={onDiscard} />}
      </section>
    )
  }
  return (
    <section className="changes-in-progress" aria-labelledby="changes-heading">
      <div className="pane-heading"><p className="eyebrow">Architecture</p><h2 id="changes-heading">Changes in progress</h2></div>
      <p>{changes.stale ? 'These changes started from an older architecture and are read-only.' : 'These changes have not updated the architecture yet.'}</p>
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
          {!result.stale && !changes.stale && !changes.review && (
            <button className="inline-action" type="button" disabled={busy} onClick={onReview}>{busy ? 'Preparing…' : 'Review changes'}</button>
          )}
          {discardAction}
        </div>
      )}
      {discardConfirming && <DiscardChangesDialog busy={busy} onCancel={onCancelDiscard} onDiscard={onDiscard} />}
    </section>
  )
}

function appearanceReviewDescription(role: 'home' | 'reference', status: 'added' | 'removed' | 'detail_changed', diagramTitle: string) {
  if (status === 'detail_changed') return `detail diagram link changed in ${diagramTitle}`
  if (role === 'home') return status === 'added' ? `now lives in ${diagramTitle}` : `no longer lives in ${diagramTitle}`
  return status === 'added' ? `shown in ${diagramTitle}` : `no longer shown in ${diagramTitle}`
}

function ReviewContext({
  side,
  component,
  components,
  focus,
  onClear,
}: {
  side: ReviewSide
  component?: AuthoringComponent
  components: AuthoringComponent[]
  focus?: ReviewFocus | null
  onClear: () => void
}) {
  const titles = new Map(components.map((candidate) => [candidate.id, candidate.title]))
  if (focus?.kind === 'diagram') {
    return (
      <section className="review-context" aria-label="Review context">
        <div className="review-context-heading"><div><p className="eyebrow">Diagram composition</p><h3>{focus.title}</h3></div><button className="text-action" type="button" onClick={onClear}>Clear focus</button></div>
        <p>{focus.status === 'added' ? 'This diagram is added with the changes.' : focus.status === 'title_changed' ? 'This diagram title changes.' : 'A component placement changes in this diagram.'}</p>
      </section>
    )
  }
  if (focus?.kind === 'relationship' && !component) {
    return (
      <section className="review-context" aria-label="Review context">
        <div className="review-context-heading">
          <div><p className="eyebrow">{relationshipStatusText(focus.status)} relationship</p><h3>Relationship</h3></div>
          <button className="text-action" type="button" onClick={onClear}>Clear focus</button>
        </div>
        <p className={`review-relationship-summary review-${focus.status}`}>
          <span>{focus.source_title}</span><strong>{focus.label}</strong><span>{focus.target_title}</span>
        </p>
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
  return (
    <section className="review-context" aria-label="Review context">
      <div className="review-context-heading">
        <div>
          <p className="eyebrow">{focus?.kind === 'relationship' ? `${relationshipStatusText(focus.status)} relationship` : focus?.kind === 'component' ? componentStatusText(focus.status) : side === 'with' ? 'With changes' : 'Before changes'}</p>
          <h3>{component.title}</h3>
        </div>
        <button className="text-action" type="button" onClick={onClear}>Clear focus</button>
      </div>
      {focus?.kind === 'relationship' && (
        <p className={`review-relationship-summary review-${focus.status}`}>
          <span>{focus.source_title}</span><strong>{focus.label}</strong><span>{focus.target_title}</span>
        </p>
      )}
      <MarkdownBody source={component.description} />
      {component.relationships.length > 0 && (
        <details className="review-relationships">
          <summary>Outgoing relationships ({component.relationships.length})</summary>
          <ul>{component.relationships.map((relationship, index) => <li key={relationship.projection_key ?? `${relationship.target_id}:${index}`}><span>{relationship.label}</span> → <span>{titles.get(relationship.target_id) ?? 'Component unavailable'}</span></li>)}</ul>
        </details>
      )}
    </section>
  )
}

function DiscardChangesDialog({ busy, onCancel, onDiscard }: { busy: boolean; onCancel: () => void; onDiscard: () => void }) {
  return (
    <div className="navigation-guard" role="dialog" aria-modal="true" aria-labelledby="discard-heading">
      <div className="discard-confirmation">
        <h2 id="discard-heading">Discard changes?</h2>
        <p>This clears every change in progress. The accepted architecture will not change.</p>
        <div className="button-group">
          <button className="secondary-action" type="button" onClick={onCancel}>Keep changes</button>
          <button className="destructive-action" type="button" disabled={busy} onClick={onDiscard}>Discard changes</button>
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

function decodeProjectSlug(pathname: string) {
  const match = /^\/projects\/([^/]+)\/?$/.exec(pathname)
  if (!match) return undefined
  try { return decodeURIComponent(match[1]) } catch { return undefined }
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
