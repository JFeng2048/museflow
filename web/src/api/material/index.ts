import request from '@/utils/request'
import { materials as mockMaterials } from '@/mock'
import type { Material } from '@/types/material'

let store: Material[] = [...mockMaterials]

export function fetchMaterials(): Promise<Material[]> {
  return request.get<Material[]>('/materials').catch(() => store)
}

export function importMaterial(id: string): Promise<Material | undefined> {
  store = store.map((m) => (m.id === id ? { ...m, imported: true } : m))
  return request
    .post<Material | undefined>(`/materials/${id}/import`)
    .catch(() => store.find((m) => m.id === id))
}
