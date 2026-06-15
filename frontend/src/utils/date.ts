// 统一时间格式化工具
// 替代 SearchPage/SearchPanel/ImmersivePlayer 中各自定义的 getTimeAgo

export function getTimeAgo(dateStr: string | Date): string {
  if (!dateStr) return ''
  const diff = Date.now() - new Date(dateStr).getTime()
  const days = Math.floor(diff / 86400000)
  if (days < 1) return '今天'
  if (days < 30) return `${days}天前`
  if (days < 365) return `${Math.floor(days / 30)}月前`
  return `${Math.floor(days / 365)}年前`
}

// 备选：不显示「前」字的短格式
export function getTimeAgoShort(dateStr: string | Date): string {
  if (!dateStr) return ''
  const diff = Date.now() - new Date(dateStr).getTime()
  const days = Math.floor(diff / 86400000)
  if (days < 1) return '今天'
  if (days < 30) return `${days}天`
  if (days < 365) return `${Math.floor(days / 30)}月`
  return `${Math.floor(days / 365)}年`
}
