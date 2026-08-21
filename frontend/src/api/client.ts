interface ErrorEnvelope {
  error: { message: string }
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

function isErrorEnvelope(body: unknown): body is ErrorEnvelope {
  return (
    typeof body === 'object' &&
    body !== null &&
    'error' in body &&
    typeof (body as ErrorEnvelope).error?.message === 'string'
  )
}

export async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(path)

  if (!res.ok) {
    const body: unknown = await res.json().catch(() => null)
    const message = isErrorEnvelope(body) ? body.error.message : res.statusText
    throw new ApiError(res.status, message)
  }

  return res.json() as Promise<T>
}
