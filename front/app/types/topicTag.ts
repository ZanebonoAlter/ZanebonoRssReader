export interface TagHierarchyNode {
  id: number
  label: string
  slug: string
  category: string
  icon: string
  feedCount: number
  similarityScore?: number
  children: TagHierarchyNode[]
}

export interface TagHierarchyResponse {
  nodes: TagHierarchyNode[]
  total: number
}

export interface UpdateAbstractNameRequest {
  newName: string
}

export interface DetachChildRequest {
  childId: number
}
