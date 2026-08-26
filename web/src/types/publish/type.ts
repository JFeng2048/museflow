export interface PublishChannel {
  id: string
  name: string
  enabled: boolean
  status: 'connected' | 'disconnected'
}
