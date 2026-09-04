import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

type MarkdownNode = {
  type: string
  value?: string
  alt?: string
  children?: MarkdownNode[]
}

export function MarkdownBody({ source }: { source: string }) {
  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, inertAuthoredResources]}
        urlTransform={(url, key) => (key === 'href' && safeLink(url) ? url : '')}
        components={{
          a: ({ href, children }) => safeLink(href) ? <a href={href}>{children}</a> : <span>{children}</span>,
          img: ({ alt }) => <span className="inert-resource">[Image: {alt || 'untitled'}]</span>,
          table: ({ children }) => <div className="markdown-table-scroll"><table>{children}</table></div>,
        }}
      >
        {source}
      </ReactMarkdown>
    </div>
  )
}

// This WorkBraid-owned AST boundary makes authored raw HTML and resource
// syntax inert before ReactMarkdown can turn it into DOM or URL attributes.
function inertAuthoredResources() {
  return (tree: MarkdownNode) => rewriteChildren(tree)
}

function rewriteChildren(node: MarkdownNode) {
  if (!node.children) return
  node.children = node.children.map((child) => {
    if (child.type === 'html') {
      return { type: 'text', value: child.value ?? '' }
    }
    if (child.type === 'image' || child.type === 'imageReference') {
      return { type: 'text', value: `[Image: ${child.alt || 'untitled'}]` }
    }
    rewriteChildren(child)
    return child
  })
}

function safeLink(href?: string) {
  if (!href) return false
  const value = href.trim()
  if (/^(?:https?:|mailto:)/i.test(value)) return true
  if (/^[a-z][a-z\d+.-]*:/i.test(value)) return false
  return true
}
