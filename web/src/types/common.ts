/** 跨模块共用的分页与删除结果结构。 */

/** 列表接口统一的分页结果（网关 data 信封内的形态）。 */
export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

/** 删除接口统一返回的结果。 */
export interface DeleteResult {
  success: boolean
}
