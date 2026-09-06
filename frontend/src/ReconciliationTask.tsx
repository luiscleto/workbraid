import { ReactNode, useId, useState } from 'react'

type Home = { component_id: string; diagram_id: string }
type Reference = Home & { present: boolean }
type Anchor = { diagram_id: string; anchor_component_id: string }
type Count = { source_id: string; target_id: string; label: string; count: number }
export type ReconciliationLocator = { kind: string; component_id?: string; diagram_id?: string; source_id?: string; target_id?: string; label?: string; reason?: string; component_ids?: string[]; diagram_ids?: string[] }
type Value = { text?: string; count?: number; present?: boolean; diagram_id?: string; anchor_component_id?: string; homes?: Home[]; references?: Reference[]; detail_anchors?: Anchor[]; relationship_counts?: Count[] }
type Side = Value & { exists: boolean; component?: { title: string; description: string }; diagram?: { title: string } }
export type ReconciliationResolution = { locator: ReconciliationLocator; choice: 'accepted' | 'proposed' | 'manual'; value?: Value }
type Conflict = { locator: ReconciliationLocator; original: Side; accepted: Side; proposed: Side; choices: string[]; unsupported?: Record<string, string>; resolved: boolean; eligible_anchor_component_ids?: string[]; eligible_parents?: { component_id: string; title: string; home_diagram_id: string; home_diagram_title: string }[] }
export type ReconciliationSnapshot = {
  revision: string; format_version: number; component_count: number; component_titles: string[]; root_diagram_id?: string
  components: { id: string; title: string; description: string; filename: string; relationships: { target_id: string; label: string }[] }[]
  diagrams?: { id: string; title: string; filename: string; parent_anchor_component_id?: string; appearances: { component_id: string; role: string; detail_diagram_id?: string }[] }[]
}
export type ReconciliationPreview = {
  status: 'ready' | 'needs_resolution' | 'blocked' | 'not_required'
  inputs: { store_id: string; change_set_id: string; generation: number; change_set_state: string; base_revision: string; candidate_tree: string; accepted_revision: string }
  original: ReconciliationSnapshot; accepted: ReconciliationSnapshot; proposed: ReconciliationSnapshot
  automatic_changes: { locator: ReconciliationLocator; reason: string; original: Side; accepted: Side; proposed: Side }[]
  conflicts: Conflict[]
  result_candidate?: ReconciliationSnapshot & { candidate_tree: string }
  remaining_changes?: boolean
}
const key = (locator: ReconciliationLocator) => JSON.stringify(locator)
function expandsComposition(next: ReconciliationLocator, prior: ReconciliationLocator): boolean {
  return next.kind === 'composition' && prior.kind === 'composition' && next.reason === prior.reason
    && (next.component_ids?.length ?? 0) + (next.diagram_ids?.length ?? 0) > (prior.component_ids?.length ?? 0) + (prior.diagram_ids?.length ?? 0)
    && (prior.component_ids ?? []).every((id) => next.component_ids?.includes(id))
    && (prior.diagram_ids ?? []).every((id) => next.diagram_ids?.includes(id))
}
function rebindComposition(locator: ReconciliationLocator, prior: ReconciliationResolution[]): ReconciliationResolution {
  const homes = new Map<string, Home>(), references = new Map<string, Reference>(), anchors = new Map<string, Anchor>(), counts = new Map<string, Count>()
  for (const { value } of prior) {
    for (const home of value?.homes ?? []) {
      if (locator.component_ids?.includes(home.component_id)) homes.set(JSON.stringify([home.component_id, home.diagram_id]), home)
    }
    for (const ref of value?.references ?? []) {
      if (locator.component_ids?.includes(ref.component_id) && locator.diagram_ids?.includes(ref.diagram_id)) references.set(JSON.stringify([ref.diagram_id, ref.component_id, ref.present]), ref)
    }
    for (const anchor of value?.detail_anchors ?? []) {
      if (locator.diagram_ids?.includes(anchor.diagram_id)) anchors.set(JSON.stringify([anchor.diagram_id, anchor.anchor_component_id]), anchor)
    }
    for (const count of value?.relationship_counts ?? []) {
      if (locator.component_ids?.includes(count.source_id) && locator.component_ids?.includes(count.target_id)) counts.set(JSON.stringify([count.source_id, count.target_id, count.label, count.count]), count)
    }
  }
  // Deduplicate identical fact/value pairs only. Contradictory values remain
  // explicit for Check to reject; no row matching or last-writer-wins choice.
  return { locator, choice: prior.every((item) => item.choice === prior[0].choice) ? prior[0].choice : 'manual', value: {
    ...(homes.size ? { homes: [...homes.values()] } : {}),
    ...(references.size ? { references: [...references.values()] } : {}),
    ...(anchors.size ? { detail_anchors: [...anchors.values()] } : {}),
    ...(counts.size ? { relationship_counts: [...counts.values()] } : {}),
  } }
}
const reasons: Record<string, string> = { competing_children: 'Detail diagrams need distinct parents', home_reference_overlap: 'A home and reference overlap', missing_home: 'Choose an existing home', missing_target: 'A proposed fact needs a missing object', missing_parent: 'A detail diagram needs a parent', unreachable_diagram: 'Connect this diagram to the main diagram', hierarchy_cycle: 'Resolve the circular hierarchy' }
const unsupported: Record<string, string> = { replace_identity: 'These independently added objects use the same ID but have different content or dependent facts. Identity replacement is unavailable. Return to the proposal to correct the collision.', restore_component: 'Restoring this removed Component is unavailable. Accepted can keep it absent; dependent proposed facts need explicit choices.', restore_diagram: 'Restoring this removed Diagram is unavailable. Accepted can keep it absent; dependent proposed facts need explicit choices.', reassign_root: 'The main Diagram cannot be reassigned as a child.', restore_source: 'This source or path cannot be restored through ordinary authoring.', delete_component: 'Deleting an Accepted Component is unavailable.', delete_diagram: 'Deleting an Accepted Diagram is unavailable.' }

export function ReconciliationTask({ name, initial, onCheck, onApply, onLeave, onDirty, renderCandidate }: {
  name: string; initial: ReconciliationPreview
  onCheck: (resolutions: ReconciliationResolution[]) => Promise<ReconciliationPreview>
  onApply: (resolutions: ReconciliationResolution[]) => Promise<void>
  onLeave: () => void; onDirty: (dirty: boolean) => void
  renderCandidate: (snapshot: ReconciliationSnapshot, diagramID: string | undefined, selectedComponentID: string | undefined, onSelect: (id: string) => void) => ReactNode
}) {
  const [preview, setPreview] = useState(initial)
  const [choices, setChoices] = useState<Record<string, ReconciliationResolution>>({})
  const [selected, setSelected] = useState(initial.conflicts[0] ? key(initial.conflicts[0].locator) : '')
  const [checked, setChecked] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [applyAttempted, setApplyAttempted] = useState(false)
  const [showResult, setShowResult] = useState(false)
  const [diagramID, setDiagramID] = useState<string>()
  const [componentID, setComponentID] = useState<string>()
  const components = new Map([...preview.original.components, ...preview.proposed.components, ...preview.accepted.components].map((component) => [component.id, component]))
  const diagrams = new Map([...(preview.original.diagrams ?? []), ...(preview.proposed.diagrams ?? []), ...(preview.accepted.diagrams ?? [])].map((diagram) => [diagram.id, diagram]))
  const componentName = (id?: string) => id ? components.get(id)?.title ?? `Missing Component (${id})` : 'No Component'
  const diagramName = (id?: string) => id ? diagrams.get(id)?.title ?? `Missing Diagram (${id})` : 'No Diagram'
  const availableComponents = [...components.values()].filter((component) => preview.accepted.components.some((item) => item.id === component.id) || !preview.original.components.some((item) => item.id === component.id))
  const availableDiagrams = [...diagrams.values()].filter((diagram) => preview.accepted.diagrams?.some((item) => item.id === diagram.id) || !preview.original.diagrams?.some((item) => item.id === diagram.id))
  const title = (locator: ReconciliationLocator) => locator.kind === 'composition' ? reasons[locator.reason ?? ''] ?? 'Composition needs a decision'
    : locator.kind === 'relationship_count' ? `${componentName(locator.source_id)} → ${componentName(locator.target_id)}`
      : `${locator.component_id ? componentName(locator.component_id) : diagramName(locator.diagram_id)} · ${locator.kind.replace('component_', '').replace('diagram_', '').replaceAll('_', ' ')}`
  const current = preview.conflicts.find((conflict) => key(conflict.locator) === selected) ?? preview.conflicts[0]
  const resolution = current ? choices[key(current.locator)] : undefined
  function choose(conflict: Conflict, choice: ReconciliationResolution['choice'], value?: Value) {
    setChoices((previous) => ({ ...previous, [key(conflict.locator)]: { locator: conflict.locator, choice, ...(value === undefined ? {} : { value }) } }))
    setChecked(false); setShowResult(false); setError(''); onDirty(true)
  }
  function sideText(side: Side): string {
    if (!side.exists) return 'Absent'
    if (side.text !== undefined) return side.text === '' ? '(Empty)' : side.text
    if (side.count !== undefined) return `${side.count} occurrence${side.count === 1 ? '' : 's'}`
    if (side.present !== undefined) return side.present ? 'Shown by reference' : 'Not shown by reference'
    if (side.diagram_id) return `Lives in ${diagramName(side.diagram_id)}`
    if (side.anchor_component_id) return `Parent component: ${componentName(side.anchor_component_id)}`
    if (side.component) return `${side.component.title}\n${side.component.description}`
    return side.diagram?.title ?? 'Present'
  }
  function sideValue(conflict: Conflict, side: 'accepted' | 'proposed'): Value {
    const value = conflict[side]
    return { ...(value.homes?.length ? { homes: value.homes } : {}), ...(value.references?.length ? { references: value.references } : {}), ...(value.detail_anchors?.length ? { detail_anchors: value.detail_anchors } : {}), ...(value.relationship_counts?.length ? { relationship_counts: value.relationship_counts } : {}) }
  }
  function choiceLabel(conflict: Conflict, choice: ReconciliationResolution['choice']): string {
    if (choice === 'manual') return 'Choose manually'
    if (conflict.locator.reason === 'competing_children') {
      const other = conflict[choice === 'accepted' ? 'proposed' : 'accepted']
      const retained = conflict[choice].detail_anchors?.find((anchor) => other.detail_anchors?.some((otherAnchor) => otherAnchor.anchor_component_id === anchor.anchor_component_id && otherAnchor.diagram_id !== anchor.diagram_id))
      if (retained) return `Keep ${diagramName(retained.diagram_id)} on ${componentName(retained.anchor_component_id)}`
    }
    return `Use ${choice === 'accepted' ? 'Accepted' : 'proposed'}`
  }
  async function check() {
    setBusy(true); setError('')
    try {
      const next = await onCheck(Object.values(choices))
      // The response may also describe the original invalid subset. Its
      // expanded group is now the decision context and supersedes that locator.
      const conflicts = next.conflicts.filter((conflict) => !next.conflicts.some((other) => expandsComposition(other.locator, conflict.locator)))
      // Expanded groups retain values by stable fact identity. They must be
      // checked again with the newly returned locator, never old row indexes.
      const retained: Record<string, ReconciliationResolution> = {}
      let expanded = false
      for (const conflict of conflicts) {
        const exact = choices[key(conflict.locator)]
        if (exact) { retained[key(conflict.locator)] = exact; continue }
        if (conflict.locator.kind === 'composition') {
          const prior = Object.values(choices).filter((choice) => expandsComposition(conflict.locator, choice.locator))
          if (prior.length) { retained[key(conflict.locator)] = rebindComposition(conflict.locator, prior); expanded = true }
        }
      }
      setChoices(retained); setPreview({ ...next, conflicts }); setChecked(!expanded)
      if (!conflicts.some((conflict) => key(conflict.locator) === selected)) setSelected(conflicts[0] ? key(conflicts[0].locator) : '')
      if (expanded) setError('The affected group expanded. Assign every involved child and check these choices again.')
    } catch (failure) { setError(failure instanceof Error ? failure.message : 'Choices could not be checked.') }
    finally { setBusy(false) }
  }
  async function apply() {
    setBusy(true); setApplyAttempted(true); setError('')
    try { await onApply(Object.values(choices)) }
    catch (failure) { setError(`${failure instanceof Error ? failure.message : 'The response was unavailable.'} Return to the proposal and inspect its current state before preparing again.`) }
    finally { setBusy(false) }
  }
  const remaining = preview.conflicts.filter((conflict) => !conflict.resolved).length
  const candidate = checked ? preview.result_candidate : undefined
  const selectedComponent = candidate?.components.find((component) => component.id === componentID)
  return <section className="reconciliation-task" aria-labelledby="reconciliation-heading">
    <div className="pane-heading pane-heading-with-action"><div><p className="eyebrow">{name}</p><h2 id="reconciliation-heading">Reconcile with Accepted</h2></div><button className="text-action" onClick={onLeave}>Back to proposal</button></div>
    <p className="reconciliation-intro">Combine this proposal with the current Accepted Architecture. Your choices stay local until you apply them.</p>
    <div className="reconciliation-body">
      <nav className="reconciliation-index" aria-label="Reconciliation decisions">
        <h3>Combined automatically <span>{preview.automatic_changes.length}</span></h3>
        <details><summary>See combined changes</summary><ul>{preview.automatic_changes.map((change) => <li key={key(change.locator)}><strong>{title(change.locator)}</strong><small>{change.reason === 'same_result' ? 'Both made the same change' : change.reason === 'accepted_only' ? 'Changed in Accepted' : 'Changed in this proposal'}</small></li>)}</ul></details>
        <h3>Needs a decision <span>{remaining}</span></h3>
        {preview.conflicts.map((conflict) => <button key={key(conflict.locator)} type="button" className="reconciliation-context" aria-current={current === conflict ? 'true' : undefined} onClick={() => { setSelected(key(conflict.locator)); setShowResult(false) }}><strong>{title(conflict.locator)}</strong><small>{conflict.resolved ? 'Checked' : choices[key(conflict.locator)] ? 'Choice needs checking' : conflict.choices.length ? 'Choose a result' : 'Unsupported collision'}</small></button>)}
        {!preview.conflicts.length && <p>No conflicting changes.</p>}
      </nav>
      <div className="reconciliation-decision">
        {showResult && candidate ? <>
          <h3>Result preview</h3><p>One complete validated Architecture. Applying updates this proposal only.</p>
          <nav className="reconciliation-diagrams" aria-label="Result diagrams">{candidate.diagrams?.map((diagram) => <button className="secondary-action" type="button" key={diagram.id} aria-current={diagram.id === (diagramID ?? candidate.root_diagram_id) ? 'true' : undefined} onClick={() => { setDiagramID(diagram.id); setComponentID(undefined) }}>{diagram.title}</button>)}</nav>
          <div className="reconciliation-map">{renderCandidate(candidate, diagramID, componentID, setComponentID)}</div>
          {selectedComponent && <div><h4>{selectedComponent.title}</h4><pre className="exact-reconciliation-text">{selectedComponent.description}</pre></div>}
        </> : current ? <>
          <h3>{title(current.locator)}</h3>
          {current.locator.kind === 'relationship_count' && <p>Exact relationship label: <span className="exact-inline-label">{current.locator.label}</span>. Choose its final number of occurrences.</p>}
          {current.locator.kind === 'composition' && <p>{current.locator.reason === 'competing_children' ? 'All involved diagrams must survive. Choose which stays on this parent, then explicitly assign every displaced diagram. You can also choose a complete manual assignment.' : 'Choose a complete valid arrangement for the affected facts. Check choices will identify any additional dependencies.'}</p>}
          <div className="reconciliation-sides">{(['original', 'accepted', 'proposed'] as const).map((side) => <section key={side}><h4>{side === 'original' ? 'Original' : side === 'accepted' ? 'Accepted' : 'Proposed'}</h4>{current.locator.kind === 'composition' ? <ul>{current[side].detail_anchors?.map((anchor) => <li key={anchor.diagram_id}>{diagramName(anchor.diagram_id)} → {componentName(anchor.anchor_component_id)}</li>)}{current[side].homes?.map((home) => <li key={home.component_id}>{componentName(home.component_id)} lives in {diagramName(home.diagram_id)}</li>)}{current[side].references?.map((ref) => <li key={`${ref.diagram_id}:${ref.component_id}`}>{componentName(ref.component_id)}: {ref.present ? 'reference in' : 'no reference in'} {diagramName(ref.diagram_id)}</li>)}</ul> : <pre className="exact-reconciliation-text">{sideText(current[side])}</pre>}</section>)}</div>
          {Object.entries(current.unsupported ?? {}).map(([choice, reason]) => <p className="reconciliation-unavailable" key={choice}>{unsupported[reason] ?? `This choice is unsupported: ${reason}`}</p>)}
          {!!current.choices.length && <section className="reconciliation-choice-section"><h4>{current.locator.reason === 'competing_children' ? 'Choose which diagram stays' : 'Choose the result'}</h4><div className="reconciliation-side-choices" role="group" aria-label="Choose result">{(['accepted', 'proposed', 'manual'] as const).filter((choice) => current.choices.includes(choice)).map((choice) => <button type="button" key={choice} aria-pressed={resolution?.choice === choice} onClick={() => choose(current, choice, current.locator.kind === 'composition' ? choice === 'manual' ? resolution?.value ?? {} : sideValue(current, choice) : choice === 'manual' ? manualInitial(current) : undefined)}>{choiceLabel(current, choice)}</button>)}</div></section>}
          {resolution && current.locator.kind === 'composition' ? <CompositionChoices conflict={current} rootID={preview.accepted.root_diagram_id} resolution={resolution} components={availableComponents} diagrams={availableDiagrams} componentName={componentName} diagramName={diagramName} onChange={(value) => choose(current, resolution.choice, value)} />
            : resolution?.choice === 'manual' ? <ScalarEditor conflict={current} value={resolution.value ?? {}} components={availableComponents} diagrams={availableDiagrams} onChange={(value) => choose(current, 'manual', value)} /> : resolution && <section className="reconciliation-chosen"><h4>Chosen result</h4><pre className="exact-reconciliation-text">{sideText(current[resolution.choice === 'accepted' ? 'accepted' : 'proposed'])}</pre></section>}
          <details className="technical-details"><summary>Exact context</summary><dl><dt>Component IDs</dt><dd>{current.locator.component_id ?? current.locator.component_ids?.join(', ') ?? current.locator.source_id}</dd><dt>Diagram IDs</dt><dd>{current.locator.diagram_id ?? current.locator.diagram_ids?.join(', ')}</dd></dl></details>
        </> : <div className="reconciliation-ready"><h3>{preview.status === 'not_required' ? 'Already based on Accepted' : 'Combined without conflicts'}</h3><p>{preview.status === 'not_required' ? 'Return to the proposal to continue ordinary work.' : 'Review the complete result, then apply it to this proposal.'}</p></div>}
      </div>
    </div>
    <footer className="reconciliation-actions">
      {error && <p className="review-error" role="alert">{error}</p>}
      {checked && preview.status === 'ready' && <p>{preview.remaining_changes === false ? 'No Architecture changes remain.' : 'The complete result is valid. Accepted and submitted feedback stay unchanged.'}</p>}
      {!checked && <p>Check your local choices before applying.</p>}
      {preview.status === 'blocked' && <p>Return to the proposal to correct the unsupported collision. This attempt has changed nothing.</p>}
      <div className="button-group">{!applyAttempted && preview.status !== 'not_required' && <button className="secondary-action" disabled={busy} onClick={check}>{busy ? 'Working…' : 'Check choices'}</button>}{candidate && <button className="secondary-action" onClick={() => setShowResult(!showResult)}>{showResult ? 'Back to decisions' : 'Preview result'}</button>}{checked && preview.status === 'ready' && !applyAttempted && <button className="inline-action" disabled={busy} onClick={apply}>Apply reconciliation</button>}</div>
      <details className="technical-details"><summary>Exact inputs</summary><dl>{Object.entries(preview.inputs).map(([label, value]) => <div key={label}><dt>{label.replaceAll('_', ' ')}</dt><dd>{String(value)}</dd></div>)}</dl></details>
    </footer>
  </section>
}

function manualInitial(conflict: Conflict): Value {
  const side = conflict.proposed
  if (side.text !== undefined) return { text: side.text }
  if (side.count !== undefined) return { count: side.count }
  if (side.present !== undefined) return { present: side.present }
  if (side.diagram_id) return { diagram_id: side.diagram_id }
  if (side.anchor_component_id) return { anchor_component_id: side.anchor_component_id }
  return {}
}
type Option = { id: string; title: string; filename: string; context?: string }
function IdentityChoices({ label, options, value, onChange }: { label: string; options: Option[]; value?: string; onChange: (id: string) => void }) {
  const groupID = useId()
  return <fieldset className="reconciliation-options"><legend>{label}</legend>{options.map((option) => <label key={option.id}><input type="radio" name={groupID} checked={value === option.id} onChange={() => onChange(option.id)} /><span><strong>{option.title}</strong>{option.context && <small>{option.context}</small>}{options.filter((other) => other.title === option.title).length > 1 && <small>{option.filename}</small>}</span></label>)}</fieldset>
}
function ScalarEditor({ conflict, value, components, diagrams, onChange }: { conflict: Conflict; value: Value; components: Option[]; diagrams: Option[]; onChange: (value: Value) => void }) {
  if (conflict.locator.kind === 'home') return <IdentityChoices label="Final home diagram" options={diagrams} value={value.diagram_id} onChange={(id) => onChange({ diagram_id: id })} />
  if (conflict.locator.kind === 'detail_anchor') {
    const eligible = (conflict.eligible_parents ?? []).map((parent) => ({ id: parent.component_id, title: parent.title, filename: components.find((component) => component.id === parent.component_id)?.filename ?? '', context: `Lives in ${parent.home_diagram_title}` }))
    return <><IdentityChoices label="Final parent component" options={eligible} value={value.anchor_component_id} onChange={(id) => onChange({ anchor_component_id: id })} /><details><summary>Other parent components (check the resulting arrangement)</summary><IdentityChoices label="Other parent component" options={components.filter((component) => !eligible.some((option) => option.id === component.id))} value={value.anchor_component_id} onChange={(id) => onChange({ anchor_component_id: id })} /></details></>
  }
  if (conflict.locator.kind === 'reference') return <label className="reconciliation-check"><input type="checkbox" checked={value.present ?? false} onChange={(event) => onChange({ present: event.target.checked })} />Show by reference</label>
  if (conflict.locator.kind === 'relationship_count') return <label className="reconciliation-manual">Final occurrence count<input type="number" min={0} step={1} value={value.count ?? ''} onChange={(event) => onChange({ count: event.target.value === '' ? undefined : Number(event.target.value) })} /></label>
  return <label className="reconciliation-manual">{conflict.locator.kind === 'component_description' ? 'Exact Markdown Description' : 'Final title'}<textarea rows={conflict.locator.kind === 'component_description' ? 12 : 3} value={value.text ?? ''} onChange={(event) => onChange({ text: event.target.value })} /></label>
}
function CompositionChoices({ conflict, rootID, resolution, components, diagrams, componentName, diagramName, onChange }: { conflict: Conflict; rootID?: string; resolution: ReconciliationResolution; components: Option[]; diagrams: Option[]; componentName: (id?: string) => string; diagramName: (id?: string) => string; onChange: (value: Value) => void }) {
  const value = resolution.value ?? {}
  const anchors = value.detail_anchors ?? []
  const homes = value.homes ?? []
  const refs = value.references ?? []
  const counts = value.relationship_counts ?? []
  const competing = conflict.locator.reason === 'competing_children'
  const eligible = components.filter((component) => conflict.eligible_anchor_component_ids?.includes(component.id)).map((component) => {
    const parent = conflict.eligible_parents?.find((item) => item.component_id === component.id)
    return { ...component, context: parent ? `Lives in ${parent.home_diagram_title}` : undefined }
  })
  const relationships = [...new Map([...conflict.original.relationship_counts ?? [], ...conflict.accepted.relationship_counts ?? [], ...conflict.proposed.relationship_counts ?? []].map((rel) => [JSON.stringify([rel.source_id, rel.target_id, rel.label]), rel])).values()]
  return <div className="reconciliation-assignments"><h4>{competing ? 'Choose a parent for each diagram' : 'Choose the affected arrangement'}</h4>
    {competing && eligible.length < (conflict.locator.diagram_ids?.length ?? 0) && <p className="reconciliation-unavailable">There is no free valid parent component for every displaced diagram. Leave reconciliation and add a suitable Component to the original proposal, then prepare again. No diagram will be removed or assigned automatically.</p>}
    {conflict.locator.diagram_ids?.filter((id) => id !== rootID && diagrams.some((diagram) => diagram.id === id)).map((id) => <div key={id}><IdentityChoices label={`Parent component for ${diagramName(id)}`} options={eligible} value={anchors.find((anchor) => anchor.diagram_id === id)?.anchor_component_id} onChange={(anchorID) => onChange({ ...value, detail_anchors: [...anchors.filter((anchor) => anchor.diagram_id !== id), { diagram_id: id, anchor_component_id: anchorID }] })} />{resolution.choice === 'manual' && <details><summary>Other parent components (may expand this group)</summary><IdentityChoices label={`Other parent for ${diagramName(id)}`} options={components.filter((component) => !eligible.some((option) => option.id === component.id))} value={anchors.find((anchor) => anchor.diagram_id === id)?.anchor_component_id} onChange={(anchorID) => onChange({ ...value, detail_anchors: [...anchors.filter((anchor) => anchor.diagram_id !== id), { diagram_id: id, anchor_component_id: anchorID }] })} /></details>}</div>)}
    {!competing && conflict.locator.component_ids?.filter((id) => components.some((component) => component.id === id)).map((id) => <IdentityChoices key={id} label={`Home for ${componentName(id)}`} options={diagrams} value={homes.find((home) => home.component_id === id)?.diagram_id} onChange={(diagramID) => onChange({ ...value, homes: [...homes.filter((home) => home.component_id !== id), { component_id: id, diagram_id: diagramID }] })} />)}
    {!competing && conflict.locator.diagram_ids?.flatMap((diagramID) => conflict.locator.component_ids?.map((componentID) => <label className="reconciliation-check" key={`${diagramID}:${componentID}`}><input type="checkbox" checked={refs.find((ref) => ref.diagram_id === diagramID && ref.component_id === componentID)?.present ?? false} onChange={(event) => onChange({ ...value, references: [...refs.filter((ref) => ref.diagram_id !== diagramID || ref.component_id !== componentID), { diagram_id: diagramID, component_id: componentID, present: event.target.checked }] })} />Show {componentName(componentID)} by reference in {diagramName(diagramID)}</label>))}
    {relationships.map((rel) => <label className="reconciliation-manual" key={JSON.stringify([rel.source_id, rel.target_id, rel.label])}>{componentName(rel.source_id)} → {componentName(rel.target_id)} · <span className="exact-inline-label">{rel.label}</span><input type="number" min={0} step={1} value={counts.find((entry) => entry.source_id === rel.source_id && entry.target_id === rel.target_id && entry.label === rel.label)?.count ?? ''} placeholder="Choose final count" onChange={(event) => onChange({ ...value, relationship_counts: [...counts.filter((entry) => entry.source_id !== rel.source_id || entry.target_id !== rel.target_id || entry.label !== rel.label), { ...rel, count: Number(event.target.value) }] })} /></label>)}
  </div>
}
