import { shallowRef } from 'vue'
import { useMutationObserver } from '@vueuse/core'

// Canvas charts need resolved colors and a redraw when the document theme changes.
export function useChartTheme() {
  const root = typeof document === 'undefined' ? null : document.documentElement

  function readTheme() {
    const style = root ? getComputedStyle(root) : undefined
    return {
      text: style?.getPropertyValue('--llmr-muted').trim() || '#626262',
      grid: style?.getPropertyValue('--llmr-line').trim() || '#e4e4e4',
      tooltipBackground: style?.getPropertyValue('--llmr-elevated').trim() || '#ffffff',
      tooltipTitle: style?.getPropertyValue('--llmr-text').trim() || '#171717',
      tooltipText: style?.getPropertyValue('--llmr-secondary').trim() || '#424242',
      series: style?.getPropertyValue('--llmr-focus').trim() || '#737373'
    }
  }

  const theme = shallowRef(readTheme())
  useMutationObserver(root, () => { theme.value = readTheme() }, {
    attributes: true,
    attributeFilter: ['class']
  })
  return theme
}
