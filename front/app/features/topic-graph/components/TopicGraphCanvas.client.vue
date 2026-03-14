<script setup lang="ts">
import { shallowRef, onMounted, onBeforeUnmount, ref, watch, computed } from 'vue'
import type { TopicGraphSceneEdge, TopicGraphSceneNode } from '~/features/topic-graph/utils/buildTopicGraphViewModel'

interface Props {
  nodes: TopicGraphSceneNode[]
  edges: TopicGraphSceneEdge[]
  activeNodeId?: string | null
  featuredNodeIds?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  activeNodeId: null,
  featuredNodeIds: () => [],
})

const emit = defineEmits<{
  nodeClick: [node: TopicGraphSceneNode]
}>()

const containerRef = ref<HTMLDivElement | null>(null)
const graphInstance = shallowRef<any>(null)
const resizeObserver = shallowRef<ResizeObserver | null>(null)
const spriteTextCtor = shallowRef<any>(null)
const threeLib = shallowRef<any>(null)
const graphReady = ref(false)

// We compute the adjacency set for the active node to distinguish branch vs peripheral
const activeAdjacency = computed(() => {
  const set = new Set<string>()
  if (!props.activeNodeId) return set
  
  props.edges.forEach(edge => {
    const sourceId = resolveLinkNodeId(edge.source)
    const targetId = resolveLinkNodeId(edge.target)
    if (sourceId === props.activeNodeId) set.add(targetId)
    if (targetId === props.activeNodeId) set.add(sourceId)
  })
  return set
})

function getNodeEmphasis(nodeId: string): 'trunk' | 'branch' | 'peripheral' {
  if (!props.activeNodeId) return 'branch' // Default state when nothing is selected
  if (nodeId === props.activeNodeId) return 'trunk'
  if (activeAdjacency.value.has(nodeId)) return 'branch'
  return 'peripheral'
}

async function setupGraph() {
  if (!containerRef.value) return

  const [{ default: ForceGraph3D }, { default: SpriteText }, threeModule] = await Promise.all([
    import('3d-force-graph'),
    import('three-spritetext'),
    import('three'),
  ])

  spriteTextCtor.value = SpriteText
  threeLib.value = threeModule

  const graph = (ForceGraph3D as any)()(containerRef.value)
    .backgroundColor('rgba(0,0,0,0)')
    .nodeRelSize(4)
    .linkOpacity(0.08)
    .linkWidth((link: TopicGraphSceneEdge) => buildLinkWidth(link))
    .linkColor((link: TopicGraphSceneEdge) => buildLinkColor(link))
    .nodeThreeObject((node: TopicGraphSceneNode) => buildNodeObject(node))
    .nodeLabel((node: TopicGraphSceneNode) => `${node.label} · ${node.kind}`)
    .onNodeClick((node: TopicGraphSceneNode) => emit('nodeClick', node))
    .d3VelocityDecay(0.2)
    .cooldownTicks(160)

  graph.d3Force('charge').strength((node: TopicGraphSceneNode) => node.kind === 'feed' ? -260 : -420)
  graph.d3Force('link').distance((link: TopicGraphSceneEdge) => link.kind === 'topic_topic' ? 132 : 184)
  graph.cameraPosition({ z: 260 })
  graphInstance.value = graph
  graphReady.value = true
  applyGraphData()

  resizeObserver.value = new ResizeObserver(() => {
    if (!containerRef.value || !graphInstance.value) return
    graphInstance.value.width(containerRef.value.clientWidth)
    graphInstance.value.height(containerRef.value.clientHeight)
  })
  resizeObserver.value.observe(containerRef.value)
}

function applyGraphData() {
  if (!graphInstance.value || !spriteTextCtor.value || !threeLib.value) return

  graphInstance.value.graphData({
    nodes: props.nodes,
    links: props.edges,
  })
}

function buildNodeObject(node: TopicGraphSceneNode) {
  if (!spriteTextCtor.value || !threeLib.value) return null

  const THREE = threeLib.value
  const SpriteText = spriteTextCtor.value
  const group = new THREE.Group()
  
  const emphasis = getNodeEmphasis(node.id)
  const isTrunk = emphasis === 'trunk'
  const isBranch = emphasis === 'branch'
  const isPeripheral = emphasis === 'peripheral'
  
  const isFeatured = isTrunk || props.featuredNodeIds.includes(node.id)
  
  // Base radius calculation
  let radius = node.kind === 'feed'
    ? Math.max(2.4, node.size * 0.12)
    : Math.max(3.2, node.size * 0.14)
    
  // Apply emphasis scaling
  if (isTrunk) {
    radius *= 1.8 // Trunk is significantly larger
  } else if (isPeripheral) {
    radius *= 0.6 // Peripheral nodes shrink
  }

  // Opacity based on emphasis
  let opacity = 0.72
  if (isTrunk) opacity = 0.98
  else if (isBranch) opacity = 0.85
  else if (isPeripheral) opacity = 0.25

  const sphere = new THREE.Mesh(
    new THREE.SphereGeometry(radius, 24, 24),
    new THREE.MeshBasicMaterial({
      color: node.accent,
      transparent: true,
      opacity,
    }),
  )
  group.add(sphere)

  // Trunk gets a strong, pulsing-like halo
  if (isTrunk) {
    const halo = new THREE.Mesh(
      new THREE.SphereGeometry(radius * 2.2, 24, 24),
      new THREE.MeshBasicMaterial({
        color: node.accent,
        transparent: true,
        opacity: 0.15,
      }),
    )
    group.add(halo)
    
    const innerHalo = new THREE.Mesh(
      new THREE.SphereGeometry(radius * 1.4, 24, 24),
      new THREE.MeshBasicMaterial({
        color: '#ffffff',
        transparent: true,
        opacity: 0.2,
      }),
    )
    group.add(innerHalo)
  }

  // Labels: Always show for trunk and branch, or if featured (when no active node)
  const showLabel = isTrunk || (props.activeNodeId && isBranch) || (!props.activeNodeId && isFeatured)
  
  if (showLabel) {
    const label = new SpriteText(compactLabel(node.label, isTrunk ? 40 : 20))
    label.color = isTrunk ? '#ffffff' : (isBranch ? '#f8f4ec' : 'rgba(248,244,236,0.6)')
    label.textHeight = isTrunk ? 10 : (isBranch ? 5.5 : 4.8)
    label.backgroundColor = isTrunk ? 'rgba(15,24,33,0.85)' : 'rgba(15,24,33,0.45)'
    label.padding = isTrunk ? 4 : 2
    label.borderRadius = 10
    label.position.set(0, radius + (isTrunk ? 12 : 8), 0)
    group.add(label)
  }

  return group
}

function buildLinkWidth(link: TopicGraphSceneEdge) {
  if (!props.activeNodeId) {
    return link.kind === 'topic_topic' ? Math.max(0.5, link.weight * 0.34) : Math.max(0.35, link.weight * 0.18)
  }

  return isFocusedEdge(link)
    ? Math.max(1.8, link.weight * 0.8) // Stronger focused edges
    : 0.1 // Thinner unfocused edges
}

function buildLinkColor(link: TopicGraphSceneEdge) {
  if (!props.activeNodeId) {
    return link.kind === 'topic_topic' ? 'rgba(240,138,75,0.34)' : 'rgba(126,151,173,0.16)'
  }

  return isFocusedEdge(link) ? 'rgba(240,138,75,0.95)' : 'rgba(255,255,255,0.03)'
}

function isFocusedEdge(link: TopicGraphSceneEdge) {
  return resolveLinkNodeId(link.source) === props.activeNodeId || resolveLinkNodeId(link.target) === props.activeNodeId
}

function resolveLinkNodeId(node: string | TopicGraphSceneNode) {
  return typeof node === 'string' ? node : node.id
}

function compactLabel(label: string, maxLength: number) {
  return label.length > maxLength ? `${label.slice(0, maxLength)}…` : label
}

function focusActiveNode() {
  if (!graphInstance.value || !props.activeNodeId) return
  const node = props.nodes.find(item => item.id === props.activeNodeId)
  if (!node) return

  const distance = 92
  const x = node.x || 0
  const y = node.y || 0
  const z = node.z || 1
  const distRatio = 1 + distance / Math.hypot(x || 1, y || 1, z || 1)

  graphInstance.value.cameraPosition(
    {
      x: x * distRatio,
      y: y * distRatio,
      z: z * distRatio,
    },
    node,
    900,
  )
}

watch(() => [props.nodes, props.edges, props.featuredNodeIds], applyGraphData, { deep: true })
watch(() => props.activeNodeId, () => {
  applyGraphData()
  focusActiveNode()
})

onMounted(() => {
  void setupGraph()
})

onBeforeUnmount(() => {
  resizeObserver.value?.disconnect()
})
</script>

<template>
  <div
    ref="containerRef"
    class="topic-canvas w-full rounded-[34px]"
    data-testid="topic-graph-canvas"
    :data-state="graphReady ? 'ready' : 'initializing'"
  />
</template>

<style scoped>
.topic-canvas {
  position: relative;
  width: 100%;
  height: clamp(34rem, 68vh, 58rem);
  min-height: 34rem;
  overflow: hidden;
  background:
    radial-gradient(circle at 18% 20%, rgba(240, 138, 75, 0.12), transparent 28%),
    radial-gradient(circle at 76% 18%, rgba(63, 124, 255, 0.12), transparent 26%),
    linear-gradient(180deg, rgba(12, 18, 25, 0.98), rgba(17, 29, 39, 0.98));
}

.topic-canvas :deep(canvas) {
  display: block;
}

@media (max-width: 1024px) {
  .topic-canvas {
    height: clamp(28rem, 62vh, 44rem);
    min-height: 28rem;
  }
}
</style>
