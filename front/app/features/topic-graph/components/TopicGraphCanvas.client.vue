<script setup lang="ts">
import { shallowRef, onMounted, onBeforeUnmount, ref, watch } from 'vue'
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
  const isActive = node.id === props.activeNodeId
  const isFeatured = isActive || props.featuredNodeIds.includes(node.id)
  const radius = node.kind === 'feed'
    ? Math.max(2.4, node.size * 0.12)
    : Math.max(isActive ? 4.5 : 3.2, node.size * (isActive ? 0.18 : 0.14))

  const sphere = new THREE.Mesh(
    new THREE.SphereGeometry(radius, 20, 20),
    new THREE.MeshBasicMaterial({
      color: node.accent,
      transparent: true,
      opacity: isActive ? 0.96 : isFeatured ? 0.88 : 0.72,
    }),
  )
  group.add(sphere)

  if (isActive) {
    const halo = new THREE.Mesh(
      new THREE.SphereGeometry(radius * 1.85, 20, 20),
      new THREE.MeshBasicMaterial({
        color: node.accent,
        transparent: true,
        opacity: 0.08,
      }),
    )
    group.add(halo)
  }

  if (isFeatured) {
    const label = new SpriteText(compactLabel(node.label, isActive ? 34 : 16))
    label.color = '#f8f4ec'
    label.textHeight = isActive ? 8 : 4.8
    label.backgroundColor = isActive ? 'rgba(15,24,33,0.68)' : 'rgba(15,24,33,0.34)'
    label.padding = isActive ? 3 : 2
    label.borderRadius = 10
    label.position.set(0, radius + (isActive ? 10 : 7), 0)
    group.add(label)
  }

  return group
}

function buildLinkWidth(link: TopicGraphSceneEdge) {
  if (!props.activeNodeId) {
    return link.kind === 'topic_topic' ? Math.max(0.5, link.weight * 0.34) : Math.max(0.35, link.weight * 0.18)
  }

  return isFocusedEdge(link)
    ? Math.max(1.25, link.weight * 0.62)
    : 0.18
}

function buildLinkColor(link: TopicGraphSceneEdge) {
  if (!props.activeNodeId) {
    return link.kind === 'topic_topic' ? 'rgba(240,138,75,0.34)' : 'rgba(126,151,173,0.16)'
  }

  return isFocusedEdge(link) ? 'rgba(240,138,75,0.88)' : 'rgba(255,255,255,0.06)'
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
  <div ref="containerRef" class="topic-canvas w-full rounded-[34px]" />
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
