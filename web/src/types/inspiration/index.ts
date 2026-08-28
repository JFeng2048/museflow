/** 灵感热梗（今日热搜）。工作区可按需取用。 */
export interface TrendingTopic {
  id: string
  title: string
  /** 热度（如 98万）。 */
  heat: string
  description: string
  /** 建议写作场景。 */
  tags: string[]
  /** 是否上热搜榜。 */
  trending: boolean
}
