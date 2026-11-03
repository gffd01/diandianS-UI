import assert from 'node:assert/strict'
import { displayProxyLink, proxyFourPart } from '../src/plugins/proxyLink.ts'

const cases = [
  ['socks5://user:pass@154.36.148.30:59976', { type: 'socks5', value: '154.36.148.30:59976:user:pass' }],
  ['socks5://user%20name:p%40ss@example.com:1080', { type: 'socks5', value: 'example.com:1080:user name:p@ss' }],
  ['http://user:pass@example.com:8080', { type: 'http', value: 'example.com:8080:user:pass' }],
  ['http://user:pass@example.com:80', { type: 'http', value: 'example.com:80:user:pass' }],
  ['https://user:pass@example.com:8443', { type: 'https', value: 'example.com:8443:user:pass' }],
  ['https://user:pass@example.com:443', { type: 'https', value: 'example.com:443:user:pass' }],
  ['socks5://user:pass@[2001:db8::1]:1080', null],
  ['vless://uuid@example.com:443', null],
]

for (const [input, expected] of cases) {
  assert.deepEqual(proxyFourPart(input), expected, input)
}
assert.equal(displayProxyLink('not-a-url'), 'not-a-url')

console.log('SOCKS5 / HTTP(S) 四段式转换检查通过')
