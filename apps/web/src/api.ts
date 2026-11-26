import { parse, stringify } from 'lossless-json'
import { ref } from 'vue'

export type ID = number | bigint
export type Value = string | ID | boolean | null
export type Row = { id: ID } & Record<string, Value | Row[] | undefined>
export const pageSize = 25

const storageKey = 'lathe.session'
export const token = ref(sessionStorage.getItem(storageKey) ?? '')

export function setToken(value: string) {
  token.value = value
  if (value) sessionStorage.setItem(storageKey, value)
  else sessionStorage.removeItem(storageKey)
}

export class RequestError extends Error {
  constructor(message: string, public status: number) {
    super(message)
  }
}

export async function request<T>(path: string, body?: unknown, method = body === undefined ? 'GET' : 'POST', credential = token.value): Promise<T> {
  let response: Response
  try {
    response = await fetch('/api/v1/' + path, {
      method,
      cache: 'no-store',
      headers: {
        Authorization: 'Bearer ' + credential,
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
      },
      body: body === undefined ? undefined : stringify(body),
    })
  } catch {
    throw new RequestError('无法连接到服务，请检查网络后重试。', 0)
  }
  if (response.status === 401) {
    if (credential === token.value) setToken('')
    throw new RequestError('访问令牌无效或已失效。', 401)
  }
  let envelope: { code: number; data: T; message?: string }
  try {
    envelope = parse(await response.text(), null, {
      parseNumber(value) {
        const number = Number(value)
        return /^-?\d+$/.test(value) && !Number.isSafeInteger(number) ? BigInt(value) : number
      },
    }) as typeof envelope
  } catch {
    throw new RequestError('服务返回了无法读取的响应。', response.status)
  }
  if (!response.ok || envelope.code !== 0) {
    throw new RequestError(envelope.message || '请求失败。', response.status)
  }
  return envelope.data
}

export function page<T extends { id: ID }>(path: string, after: string = '0', limit = pageSize) {
  return request<T[]>(`${path}?after_id=${after}&limit=${limit}`)
}

export async function all<T extends { id: ID }>(path: string): Promise<T[]> {
  const rows: T[] = []
  let after = '0'
  for (;;) {
    const next = await page<T>(path, after, 200)
    rows.push(...next)
    if (next.length < 200) return rows
    after = String(next[next.length - 1].id)
  }
}

export function quantity(value: ID) {
  const integer = BigInt(value)
  const whole = integer / 1000n
  const fraction = (integer % 1000n).toString().padStart(3, '0').replace(/0+$/, '')
  return fraction ? `${whole}.${fraction}` : String(whole)
}

export function dateToday() {
  const date = new Date()
  return [date.getFullYear(), date.getMonth() + 1, date.getDate()]
    .map(value => String(value).padStart(2, '0')).join('-')
}
