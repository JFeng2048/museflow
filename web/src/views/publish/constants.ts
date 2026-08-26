export const PUBLISH_STATUS_META: Record<'connected' | 'disconnected', { labelKey: string; type: 'success' | 'default' }> = {
  connected: { labelKey: 'publish.connected', type: 'success' },
  disconnected: { labelKey: 'publish.disconnected', type: 'default' },
}
