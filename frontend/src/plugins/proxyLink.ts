/**
 * 把标准 SOCKS5 URI 转成常见的四段式代理格式。
 *
 * 标准 URI 仍用于订阅和二维码；四段式只用于面板展示与手动复制，
 * 避免破坏小火箭、v2rayN 等客户端的 URI 导入兼容性。
 */
export function socks5FourPart(uri: string): string | null {
  try {
    const parsed = new URL(uri)
    if (parsed.protocol !== 'socks5:' || !parsed.hostname || !parsed.port) return null

    // IP:PORT:USER:PASS 无法无歧义表达 IPv6，IPv6 保留标准 URI。
    if (parsed.hostname.includes(':')) return null

    const username = decodeURIComponent(parsed.username)
    const password = decodeURIComponent(parsed.password)
    if (!username || !password) return null

    return `${parsed.hostname}:${parsed.port}:${username}:${password}`
  } catch {
    return null
  }
}

export function displayProxyLink(uri: string): string {
  return socks5FourPart(uri) ?? uri
}
