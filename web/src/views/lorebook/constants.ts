export const ROLE_META: Record<string, { labelKey: string; type: 'error' | 'info' | 'warning' | 'default' }> = {
  主角: { labelKey: 'lore.roleProtagonist', type: 'error' },
  配角: { labelKey: 'lore.roleDeuteragonist', type: 'info' },
  反派: { labelKey: 'lore.roleAntagonist', type: 'warning' },
  龙套: { labelKey: 'lore.roleMinor', type: 'default' },
}

export const FORESHADOW_META: Record<string, { labelKey: string; type: 'default' | 'info' | 'success' }> = {
  planted: { labelKey: 'lore.fsPlanted', type: 'default' },
  revealing: { labelKey: 'lore.fsRevealing', type: 'info' },
  resolved: { labelKey: 'lore.fsResolved', type: 'success' },
}
