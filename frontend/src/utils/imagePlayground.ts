import type { CustomMenuItem } from '@/types'

export const IMAGE_PLAYGROUND_MENU_ID = 'image-playground'
export const IMAGE_PLAYGROUND_PATH = '/ai-image'
export const IMAGE_PLAYGROUND_ROUTE_NAME = 'ImagePlayground'

export function findImagePlaygroundMenuItem(
  items: CustomMenuItem[],
  visibility: CustomMenuItem['visibility'],
): CustomMenuItem | undefined {
  return items.find((item) => item.id === IMAGE_PLAYGROUND_MENU_ID && item.visibility === visibility)
}

export function getCustomMenuPath(id: string): string {
  return id === IMAGE_PLAYGROUND_MENU_ID ? IMAGE_PLAYGROUND_PATH : `/custom/${id}`
}
