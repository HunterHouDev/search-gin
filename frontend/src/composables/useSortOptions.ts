import { computed } from 'vue'
import { FieldEnum, DescEnum } from 'components/utils'

// 统一排序选项生成器
// 替代 SearchPage/SearchPanel/ListEditDialog/ImmersivePlayer 中重复的 sortOptions computed

export function useSortOptions(separator = ' ') {
  return computed(() => {
    const options: Array<{ label: string; value: string }> = []
    for (const field of FieldEnum) {
      for (const desc of DescEnum) {
        options.push({
          label: `${field.label}${separator}${desc.label}`,
          value: `${field.value}_${desc.value}`,
        })
      }
    }
    return options
  })
}
