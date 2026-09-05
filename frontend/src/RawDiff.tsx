import { useEffect, useMemo, useRef } from 'react'

type RawDiffProps = {
  diff: string
  focusPath?: string
  focusToken?: string
}

type DiffLine = {
  text: string
  kind: 'added' | 'removed' | 'header' | 'hunk' | 'context'
  path?: string
  anchor?: string
}

export function RawDiff({ diff, focusPath, focusToken }: RawDiffProps) {
  const lines = useMemo(() => rawDiffLines(diff), [diff])
  const fileAnchors = useRef(new Map<string, HTMLSpanElement>())
  const container = useRef<HTMLPreElement>(null)

  useEffect(() => {
    if (!focusPath) return
    const target = fileAnchors.current.get(focusPath)
    if (!target) return
    // Follow the selected file inside the diff without scrolling the whole
    // working pane away from the Component or comment the person selected.
    const diffBox = container.current
    target.focus({ preventScroll: true })
    if (diffBox) diffBox.scrollTop += target.getBoundingClientRect().top - diffBox.getBoundingClientRect().top
  }, [focusPath, focusToken, diff])

  return (
    <pre ref={container} className="raw-diff" data-testid="raw-diff" aria-label="Complete architecture diff">
      {lines.map((line, index) => (
        <span
          className={`diff-line diff-${line.kind}`}
          data-diff-path={line.path}
          id={line.anchor}
          key={`${index}:${line.anchor ?? ''}`}
          tabIndex={line.path || line.kind === 'hunk' ? -1 : undefined}
          aria-label={diffLineLabel(line.kind)}
          ref={line.path ? (element) => {
            if (element) fileAnchors.current.set(line.path as string, element)
            else fileAnchors.current.delete(line.path as string)
          } : undefined}
        >{line.text}</span>
      ))}
    </pre>
  )
}

export function rawDiffLines(diff: string): DiffLine[] {
  const sourceLines = diff.match(/[^\n]*\n|[^\n]+$/g) ?? []
  let currentPath = ''
  let hunk = 0
  let inHunk = false
  return sourceLines.map((text) => {
    if (text.startsWith('diff --git ')) {
      currentPath = diffHeaderPath(text) ?? ''
      hunk = 0
      inHunk = false
      return currentPath
        ? { text, kind: 'header', path: currentPath, anchor: `diff-file-${stableAnchor(currentPath)}` }
        : { text, kind: 'header' }
    }
    if (text.startsWith('@@')) {
      hunk += 1
      inHunk = true
      return { text, kind: 'hunk', anchor: `diff-hunk-${stableAnchor(currentPath)}-${hunk}` }
    }
    if (!inHunk && (isFileMetadata(text, '---', 'a', currentPath) || isFileMetadata(text, '+++', 'b', currentPath))) {
      return { text, kind: 'header' }
    }
    if (text.startsWith('+')) return { text, kind: 'added' }
    if (text.startsWith('-')) return { text, kind: 'removed' }
    if (text.startsWith('index ') || text.startsWith('new file ') || text.startsWith('deleted file ')) return { text, kind: 'header' }
    return { text, kind: 'context' }
  })
}

function diffHeaderPath(text: string) {
  const line = withoutLineEnding(text)
  const argumentsText = presentationUnescapeBackslashes(line.slice('diff --git '.length))
  if (argumentsText === undefined) return undefined
  if (argumentsText.startsWith('"')) {
    const before = readGitQuotedPath(argumentsText, 0)
    if (!before) return undefined
    let next = before.end
    while (argumentsText[next] === ' ') next += 1
    const after = readGitQuotedPath(argumentsText, next)
    if (!after || after.end !== argumentsText.length) return undefined
    if (!before.value.startsWith('a/') || !after.value.startsWith('b/')) return undefined
    const beforePath = before.value.slice(2)
    const afterPath = after.value.slice(2)
    return beforePath === afterPath ? afterPath : undefined
  }
  if (!argumentsText.startsWith('a/')) return undefined
  for (let separator = argumentsText.indexOf(' b/', 2); separator >= 0; separator = argumentsText.indexOf(' b/', separator + 1)) {
    const beforePath = argumentsText.slice(2, separator)
    const afterPath = argumentsText.slice(separator + 3)
    if (beforePath === afterPath) return afterPath
  }
  return undefined
}

function isFileMetadata(text: string, marker: '---' | '+++', side: 'a' | 'b', currentPath: string) {
  if (!currentPath) return false
  const line = withoutLineEnding(text)
  if (!line.startsWith(`${marker} `)) return false
  const token = line.slice(marker.length + 1)
  if (token === '/dev/null') return true
  const gitToken = presentationUnescapeBackslashes(token)
  if (gitToken === undefined) return false
  const decoded = decodeWholeGitPathToken(gitToken)
  return decoded === `${side}/${currentPath}`
}

function presentationUnescapeBackslashes(value: string) {
  let unescaped = ''
  for (let index = 0; index < value.length; index += 1) {
    if (value[index] !== '\\') {
      unescaped += value[index]
      continue
    }
    if (value[index + 1] !== '\\') return undefined
    unescaped += '\\'
    index += 1
  }
  return unescaped
}

function decodeWholeGitPathToken(token: string) {
  if (!token.startsWith('"')) return token
  const decoded = readGitQuotedPath(token, 0)
  return decoded?.end === token.length ? decoded.value : undefined
}

function readGitQuotedPath(input: string, start: number): { value: string; end: number } | undefined {
  if (input[start] !== '"') return undefined
  const bytes: number[] = []
  const encoder = new TextEncoder()
  let index = start + 1
  while (index < input.length) {
    const character = input[index]
    if (character === '"') {
      try {
        return { value: new TextDecoder('utf-8', { fatal: true }).decode(Uint8Array.from(bytes)), end: index + 1 }
      } catch {
        return undefined
      }
    }
    if (character === '\\') {
      index += 1
      const escaped = input[index]
      if (escaped === undefined) return undefined
      if (/[0-7]/.test(escaped)) {
        let octal = escaped
        while (octal.length < 3 && /[0-7]/.test(input[index + 1] ?? '')) {
          index += 1
          octal += input[index]
        }
        bytes.push(Number.parseInt(octal, 8))
        index += 1
        continue
      }
      const escapeBytes: Record<string, number> = {
        a: 0x07, b: 0x08, t: 0x09, n: 0x0a, v: 0x0b, f: 0x0c, r: 0x0d,
        '"': 0x22, '\\': 0x5c,
      }
      if (!(escaped in escapeBytes)) return undefined
      bytes.push(escapeBytes[escaped])
      index += 1
      continue
    }
    const codePoint = input.codePointAt(index)
    if (codePoint === undefined) return undefined
    const literal = String.fromCodePoint(codePoint)
    bytes.push(...encoder.encode(literal))
    index += literal.length
  }
  return undefined
}

function withoutLineEnding(text: string) {
  return text.endsWith('\n') ? text.slice(0, -1) : text
}

function stableAnchor(value: string) {
  let hash = 2166136261
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return (hash >>> 0).toString(36)
}

function diffLineLabel(kind: DiffLine['kind']) {
  if (kind === 'added') return 'Added line'
  if (kind === 'removed') return 'Removed line'
  if (kind === 'hunk') return 'Diff hunk'
  if (kind === 'header') return 'Diff header'
  return undefined
}
