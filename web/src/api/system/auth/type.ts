import type { User } from '@/types/system/auth'

export interface AuthResult {
  token: string
  user: User
}
