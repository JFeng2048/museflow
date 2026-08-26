import type { MaterialType } from '@/types'

export const MATERIAL_TYPE_OPTIONS: { labelKey: string; value: MaterialType | 'all' }[] = [
  { labelKey: 'material.typeAll', value: 'all' },
  { labelKey: 'material.typeQuote', value: 'quote' },
  { labelKey: 'material.typeWorld', value: 'world' },
  { labelKey: 'material.typePlot', value: 'plot' },
  { labelKey: 'material.typeCharacter', value: 'character' },
  { labelKey: 'material.typeImage', value: 'image' },
]

export const MATERIAL_TYPE_LABEL_KEYS: Record<MaterialType, string> = {
  quote: 'material.typeQuote',
  world: 'material.typeWorld',
  plot: 'material.typePlot',
  character: 'material.typeCharacter',
  image: 'material.typeImage',
}

export const MATERIAL_TYPE_COLORS: Record<
  MaterialType,
  'default' | 'info' | 'success' | 'warning' | 'error'
> = {
  quote: 'default',
  world: 'success',
  plot: 'warning',
  character: 'info',
  image: 'error',
}
