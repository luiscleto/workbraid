import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { MarkdownBody } from './MarkdownBody'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

it('renders the approved Markdown features without changing source', () => {
  const source = '## Notes\n\n| A | B |\n| - | - |\n| 1 | 2 |\n\n- [x] done\n\n~~old~~\n\nhttps://example.test\n\n```mermaid\ngraph TD\n```\n'
  const exactSource = source
  const { container } = render(<MarkdownBody source={source} />)

  expect(screen.getByRole('heading', { name: 'Notes' })).toBeInTheDocument()
  const markdown = container.querySelector<HTMLElement>('.markdown-body')!
  const table = screen.getByRole('table')
  const checkbox = screen.getByRole('checkbox')
  expect(markdown).toContainElement(table)
  expect(table.parentElement).toHaveClass('markdown-table-scroll')
  expect(markdown.querySelector('ul')).toContainElement(checkbox)
  expect(checkbox).toBeChecked()
  expect(container.querySelector('del')).toHaveTextContent('old')
  expect(screen.getByRole('link', { name: 'https://example.test' })).toBeInTheDocument()
  expect(screen.getByText('graph TD')).toBeInTheDocument()
  expect(source).toBe(exactSource)
})

it('presents representative raw HTML literally and creates no active authored nodes', () => {
  const source = '<script>alert(1)</script>\n\n<img src="/secret">\n\n<iframe src="https://example.test"></iframe>\n\n<object data="x"></object>\n\n<embed src="x">\n\n<svg onload="x"></svg>\n\n<div onclick="x">literal</div>\n\n<style>body{display:none}</style>'
  const { container } = render(<MarkdownBody source={source} />)

  expect(container).toHaveTextContent('<script>alert(1)</script>')
  expect(container).toHaveTextContent('<img src="/secret">')
  expect(container).toHaveTextContent('<iframe src="https://example.test"></iframe>')
  expect(container).toHaveTextContent('<object data="x"></object>')
  expect(container).toHaveTextContent('<embed src="x">')
  expect(container).toHaveTextContent('<svg onload="x"></svg>')
  expect(container).toHaveTextContent('<div onclick="x">literal</div>')
  expect(container).toHaveTextContent('<style>body{display:none}</style>')
  expect(container.querySelectorAll('script,img,iframe,object,embed,svg,style')).toHaveLength(0)
  expect(container.querySelector('[onclick]')).toBeNull()
})

it('turns inline and reference images into inert text without requesting resources', () => {
  const request = vi.spyOn(globalThis, 'fetch')
  const source = '![remote](http://127.0.0.1:49152/watch)\n\n![local](file:///etc/passwd)\n\n![reference][asset]\n\n[asset]: /private/image.png\n'
  const { container } = render(<MarkdownBody source={source} />)

  expect(screen.getByText('[Image: remote]')).toBeInTheDocument()
  expect(screen.getByText('[Image: local]')).toBeInTheDocument()
  expect(screen.getByText('[Image: reference]')).toBeInTheDocument()
  expect(container.querySelectorAll('img,video,audio,iframe,object,embed,source')).toHaveLength(0)
  expect(request).not.toHaveBeenCalled()
})

it('keeps ordinary links deliberate and disables executable schemes', () => {
  render(<MarkdownBody source={'[Read more](https://example.test) and [do not run](javascript:alert(1))'} />)

  expect(screen.getByRole('link', { name: 'Read more' })).toHaveAttribute('href', 'https://example.test')
  expect(screen.queryByRole('link', { name: 'do not run' })).not.toBeInTheDocument()
  expect(screen.getByText('do not run')).toBeInTheDocument()
})
