import assert from 'node:assert/strict'
import { displayProxyLink, socks5FourPart } from '../src/plugins/proxyLink.ts'

const cases = [
  ['socks5://user:pass@154.36.148.30:59976', '154.36.148.30:59976:user:pass'],
  ['socks5://user%20name:p%40ss@example.com:1080', 'example.com:1080:user name:p@ss'],
  ['http://user:pass@example.com:8080', null],
  ['socks5://user:pass@[2001:db8::1]:1080', null],
]

for (const [input, expected] of cases) {
  assert.equal(socks5FourPart(input), expected, input)
}
assert.equal(displayProxyLink('not-a-url'), 'not-a-url')

console.log('SOCKS5 四段式转换检查通过')
