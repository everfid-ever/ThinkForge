export const getStatusType = (status) => {
  switch (status) {
    case 0: return 'info'    // 待处理
    case 1: return 'warning' // 处理中
    case 2: return 'success' // 已完成
    case 3: return 'danger'  // 失败
    default: return 'info'
  }
}

export const getStatusText = (status) => {
  switch (status) {
    case 0: return 'Pending processing'
    case 1: return 'Processing'
    case 2: return 'Completed'
    case 3: return 'Failed'
    default: return 'Unknown'
  }
}

export const formatDate = (date) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}