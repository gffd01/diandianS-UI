/**
 * 把标准 SOCKS5 / HTTP(S) URI 转成常见的四段式代理格式。
 *
 * 标准 URI 仍用于订阅和二维码；四段式只用于面板展示与手动复制，
 * 避免破坏小火箭、v2rayN 等客户端的 URI 导入兼容性。
 */
export type FourPartProxyType = 'socks5' | 'http' | 'https'

export interface FourPartProxy {
  type: FourPartProxyType
  value: string
}

export function proxyFourPart(uri: string): FourPartProxy | null {
  try {
    const parsed = new URL(uri)
    const protocol = parsed.protocol.slice(0, -1) as FourPartProxyType
    if (!['socks5', 'http', 'https'].includes(protocol) || !parsed.hostname) return null

    // URL 会省略 HTTP/HTTPS 默认端口，四段式中需要补回。
    const port = parsed.port || (protocol === 'http' ? '80' : protocol === 'https' ? '443' : '')
    if (!port) return null

    // IP:PORT:USER:PASS 无法无歧义表达 IPv6，IPv6 保留标准 URI。
    if (parsed.hostname.includes(':')) return null

    const username = decodeURIComponent(parsed.username)
    const password = decodeURIComponent(parsed.password)
    if (!username || !password) return null

    return {
      type: protocol,
      value: `${parsed.hostname}:${port}:${username}:${password}`,
    }
  } catch {
    return null
  }
}

export function displayProxyLink(uri: string): string {
  return proxyFourPart(uri)?.value ?? uri
}
