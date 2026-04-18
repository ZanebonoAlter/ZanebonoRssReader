<script setup lang="ts">
import { ref, watch, onBeforeUnmount, shallowRef } from 'vue'
import type { NarrativeItem, NarrativeTimelineDay } from '~/api/topicGraph'

interface Props {
  days: NarrativeTimelineDay[]
  selectedId: number | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  select: [id: number]
  hover: [id: number | null]
}>()

const containerRef = ref<HTMLDivElement | null>(null)
const p5Instance = shallowRef<any>(null)

onBeforeUnmount(() => {
  if (p5Instance.value) {
    p5Instance.value.remove()
    p5Instance.value = null
  }
})

type PaletteEntry = { dot: number[]; line: number[]; bg: number[]; badge: number[]; glow: number[] }

const PALETTE: Record<string, PaletteEntry> = {
  emerging:   { dot: [52, 211, 153],  line: [52, 211, 153],  bg: [20, 50, 40],  badge: [52, 211, 153, 40],  glow: [52, 211, 153, 30] },
  continuing: { dot: [96, 165, 250],  line: [96, 165, 250],  bg: [20, 35, 55],  badge: [96, 165, 250, 40],  glow: [96, 165, 250, 30] },
  splitting:  { dot: [251, 146, 60],  line: [251, 146, 60],  bg: [50, 35, 18],  badge: [251, 146, 60, 40], glow: [251, 146, 60, 30] },
  merging:    { dot: [192, 132, 252], line: [192, 132, 252], bg: [40, 28, 55],  badge: [192, 132, 252, 40], glow: [192, 132, 252, 30] },
  ending:     { dot: [107, 114, 128], line: [107, 114, 128], bg: [30, 30, 35],  badge: [107, 114, 128, 40], glow: [107, 114, 128, 30] },
}

const STATUS_LABELS: Record<string, string> = {
  emerging: '新兴', continuing: '持续', splitting: '分化', merging: '融合', ending: '终结',
}

function pal(status: string): PaletteEntry {
  return PALETTE[status] ?? PALETTE.ending!
}

function formatDate(dateStr: string) {
  const d = new Date(dateStr + 'T00:00:00')
  const m = d.getMonth() + 1
  const day = d.getDate()
  const weekdays = ['日', '一', '二', '三', '四', '五', '六']
  return `${m}/${day} ${weekdays[d.getDay()]}`
}

async function initP5() {
  if (!containerRef.value) return
  if (p5Instance.value) {
    p5Instance.value.remove()
    p5Instance.value = null
  }

  const p5Module = await import('p5')
  const P5 = p5Module.default

  const sketch = (p: any) => {
    const CARD_W = 200
    const CARD_H = 64
    const COL_W = 240
    const HEADER_H = 44
    const CARD_GAP = 10
    const PAD_LEFT = 20
    const PAD_TOP = 16
    const ACCENT_W = 4

    interface LayoutNode {
      narrative: NarrativeItem
      col: number
      row: number
      x: number
      y: number
      w: number
      h: number
    }

    interface LayoutEdge {
      from: number
      to: number
      fx: number; fy: number
      tx: number; ty: number
      status: string
      childStatus: string
      isContinuation: boolean
    }

    let nodes = new Map<number, LayoutNode>()
    let edges: LayoutEdge[] = []
    let canvasW = 800
    let canvasH = 400
    let hoveredId: number | null = null
    let highlightedIds: Set<number> | null = null
    let highlightedEdgeSet = new Set<number>()
    let dashOffset = 0
    let canvasEl: HTMLCanvasElement

    function computeLayout() {
      const days = props.days.filter((d: NarrativeTimelineDay) => d.narratives.length > 0)
      if (!days.length) {
        nodes = new Map()
        edges = []
        canvasW = 800
        canvasH = 400
        return
      }

      const allNarratives: NarrativeItem[] = []
      for (const day of days) allNarratives.push(...day.narratives)

      const nm = new Map<number, LayoutNode>()
      let maxRows = 0

      days.forEach((day: NarrativeTimelineDay, col: number) => {
        day.narratives.forEach((n, row) => {
          const x = PAD_LEFT + col * COL_W + COL_W / 2
          const y = PAD_TOP + HEADER_H + row * (CARD_H + CARD_GAP) + CARD_H / 2
          nm.set(n.id, {
            narrative: n,
            col, row,
            x, y,
            w: CARD_W, h: CARD_H,
          })
        })
        maxRows = Math.max(maxRows, day.narratives.length)
      })

      const es: LayoutEdge[] = []
      for (const n of allNarratives) {
        const child = nm.get(n.id)
        if (!child) continue
        for (const pid of n.parent_ids) {
          const parent = nm.get(pid)
          if (!parent) continue
          es.push({
            from: pid, to: n.id,
            fx: parent.x + parent.w / 2, fy: parent.y,
            tx: child.x - child.w / 2, ty: child.y,
            status: parent.narrative.status,
            childStatus: child.narrative.status,
            isContinuation: child.narrative.status === 'continuing',
          })
        }
      }

      nodes = nm
      edges = es
      canvasW = PAD_LEFT + days.length * COL_W + PAD_LEFT
      canvasH = PAD_TOP + HEADER_H + maxRows * (CARD_H + CARD_GAP) + PAD_TOP
    }

    function hitTest(mx: number, my: number): number | null {
      for (const [id, node] of nodes) {
        if (mx >= node.x - node.w / 2 && mx <= node.x + node.w / 2 &&
            my >= node.y - node.h / 2 && my <= node.y + node.h / 2) {
          return id
        }
      }
      return null
    }

    function computeHighlight(nodeId: number | null) {
      if (nodeId === null) {
        highlightedIds = null
        highlightedEdgeSet = new Set()
        return
      }
      const connected = new Set<number>([nodeId])
      const allN = props.days.flatMap((d: NarrativeTimelineDay) => d.narratives)
      const nMap = new Map(allN.map((n: NarrativeItem) => [n.id, n]))

      const upQ = [nodeId]
      const upV = new Set<number>()
      while (upQ.length) {
        const cur = upQ.shift()!
        if (upV.has(cur)) continue
        upV.add(cur); connected.add(cur)
        const n = nMap.get(cur)
        if (n) for (const pid of n.parent_ids) if (!upV.has(pid)) upQ.push(pid)
      }
      const downQ = [nodeId]
      const downV = new Set<number>()
      while (downQ.length) {
        const cur = downQ.shift()!
        if (downV.has(cur)) continue
        downV.add(cur); connected.add(cur)
        const n = nMap.get(cur)
        if (n) for (const cid of n.child_ids) if (!downV.has(cid)) downQ.push(cid)
      }

      highlightedIds = connected
      highlightedEdgeSet = new Set<number>()
      edges.forEach((e, i) => {
        if (connected.has(e.from) && connected.has(e.to)) highlightedEdgeSet.add(i)
      })
    }

    /* ── bezier helpers ── */
    function drawBezier(x1: number, y1: number, x2: number, y2: number) {
      const midX = (x1 + x2) / 2
      p.noFill()
      p.beginShape()
      for (let t = 0; t <= 1; t += 0.02) {
        p.vertex(p.bezierPoint(x1, midX, midX, x2, t), p.bezierPoint(y1, y1, y2, y2, t))
      }
      p.endShape()
    }

    function drawBezierPartial(x1: number, y1: number, x2: number, y2: number, fromT: number, toT: number) {
      const midX = (x1 + x2) / 2
      p.noFill()
      p.beginShape()
      for (let t = fromT; t <= toT; t += 0.02) {
        p.vertex(p.bezierPoint(x1, midX, midX, x2, t), p.bezierPoint(y1, y1, y2, y2, t))
      }
      p.endShape()
    }

    function drawDashedBezier(x1: number, y1: number, x2: number, y2: number, dashLen: number, gapLen: number, offset: number) {
      const midX = (x1 + x2) / 2
      const pts: Array<{ x: number; y: number }> = []
      for (let t = 0; t <= 1; t += 0.01) {
        pts.push({
          x: p.bezierPoint(x1, midX, midX, x2, t),
          y: p.bezierPoint(y1, y1, y2, y2, t),
        })
      }
      let dist = offset % (dashLen + gapLen)
      if (dist < 0) dist += dashLen + gapLen
      let drawing = true
      p.noFill()
      p.beginShape()
      for (let i = 1; i < pts.length; i++) {
        const dx = pts[i]!.x - pts[i - 1]!.x
        const dy = pts[i]!.y - pts[i - 1]!.y
        dist += Math.sqrt(dx * dx + dy * dy)
        if (drawing) p.vertex(pts[i]!.x, pts[i]!.y)
        const period = drawing ? dashLen : gapLen
        if (dist >= period) {
          dist = 0
          drawing = !drawing
          if (drawing) { p.endShape(); p.beginShape(); p.vertex(pts[i]!.x, pts[i]!.y) }
          else { p.endShape() }
        }
      }
      p.endShape()
    }

    function wrapText(text: string, maxW: number, maxLines: number): string[] {
      const lines: string[] = []
      let current = ''
      for (const ch of text) {
        const test = current + ch
        if (p.textWidth(test) > maxW && current.length > 0) {
          lines.push(current)
          current = ch
          if (lines.length >= maxLines) {
            lines[lines.length - 1] += '…'
            return lines
          }
        } else {
          current = test
        }
      }
      if (current) lines.push(current)
      return lines
    }

    /* ── p5 lifecycle ── */
    p.setup = () => {
      computeLayout()
      canvasEl = p.createCanvas(canvasW, canvasH).parent(containerRef.value!).elt as HTMLCanvasElement
      canvasEl.style.cursor = 'default'
      p.textFont('system-ui, -apple-system, sans-serif')
      p.frameRate(30)
    }

    p.draw = () => {
      p.background(10, 15, 22)
      dashOffset -= 0.4

      /* dot grid */
      p.noStroke()
      p.fill(186, 206, 226, 10)
      for (let gx = 0; gx < canvasW; gx += 20) {
        for (let gy = 0; gy < canvasH; gy += 20) {
          p.circle(gx, gy, 1.5)
        }
      }

      /* date columns */
      const visDays = props.days.filter((d: NarrativeTimelineDay) => d.narratives.length > 0)
      for (let di = 0; di < visDays.length; di++) {
        const day = visDays[di]!
        const colX = PAD_LEFT + di * COL_W

        /* column separator */
        if (di > 0) {
          p.stroke(255, 255, 255, 8)
          p.strokeWeight(0.5)
          p.line(colX, PAD_TOP, colX, canvasH - PAD_TOP)
          p.noStroke()
        }

        /* date header */
        p.fill(186, 206, 226, 80)
        p.textSize(12)
        p.textAlign(p.CENTER, p.CENTER)
        p.text(formatDate(day.date), colX + COL_W / 2, PAD_TOP + HEADER_H / 2)

        /* header underline */
        p.stroke(255, 255, 255, 12)
        p.strokeWeight(0.5)
        p.line(colX + 10, PAD_TOP + HEADER_H, colX + COL_W - 10, PAD_TOP + HEADER_H)
        p.noStroke()
      }

      /* ── edges ── */
      for (let i = 0; i < edges.length; i++) {
        const e = edges[i]!
        const isHl = highlightedEdgeSet.has(i)
        const hasHighlight = highlightedIds !== null
        const pc = pal(e.status)
        const cc = pal(e.childStatus)

        if (hasHighlight && !isHl) {
          p.stroke(255, 255, 255, 12)
          p.strokeWeight(1)
          drawBezier(e.fx, e.fy, e.tx, e.ty)
          continue
        }

        if (e.isContinuation) {
          p.strokeWeight(1.5)
          p.stroke(pc.line[0]!, pc.line[1]!, pc.line[2]!, isHl ? 120 : 50)
          drawDashedBezier(e.fx, e.fy, e.tx, e.ty, 8, 6, dashOffset)
          if (isHl) {
            p.strokeWeight(4)
            p.stroke(pc.line[0]!, pc.line[1]!, pc.line[2]!, 20)
            drawBezier(e.fx, e.fy, e.tx, e.ty)
          }
        } else {
          const alpha = isHl ? 160 : 60
          p.strokeWeight(isHl ? 2.5 : 1.8)
          p.stroke(pc.line[0]!, pc.line[1]!, pc.line[2]!, alpha)
          drawBezierPartial(e.fx, e.fy, e.tx, e.ty, 0, 0.5)
          p.stroke(cc.line[0]!, cc.line[1]!, cc.line[2]!, alpha)
          drawBezierPartial(e.fx, e.fy, e.tx, e.ty, 0.5, 1)
          if (isHl) {
            p.strokeWeight(6)
            p.stroke(pc.line[0]!, pc.line[1]!, pc.line[2]!, 15)
            drawBezier(e.fx, e.fy, e.tx, e.ty)
          }
        }
      }

      /* ── nodes ── */
      for (const [id, node] of nodes) {
        const st = pal(node.narrative.status)
        const hasHl = highlightedIds !== null
        const isHl = highlightedIds?.has(id) ?? false
        const isDimmed = hasHl && !isHl
        const isSelected = props.selectedId === id
        const isHovered = hoveredId === id
        const alpha = isDimmed ? 40 : 255
        const nx = node.x - node.w / 2
        const ny = node.y - node.h / 2

        p.push()

        if ((isHovered || isSelected) && !isDimmed) {
          p.noStroke()
          p.fill(st.glow[0]!, st.glow[1]!, st.glow[2]!, 25)
          p.rect(nx - 4, ny - 4, node.w + 8, node.h + 8, 14)
        }

        p.fill(18, 25, 35, alpha * 0.92)
        p.stroke(255, 255, 255, isDimmed ? 10 : (isHovered || isSelected ? 35 : 15))
        p.strokeWeight(isHovered || isSelected ? 1.2 : 0.8)
        p.rect(nx, ny, node.w, node.h, 10)

        p.noStroke()
        p.fill(st.dot[0]!, st.dot[1]!, st.dot[2]!, isDimmed ? 40 : 200)
        p.rect(nx, ny + 6, ACCENT_W, node.h - 12, 2, 0, 0, 2)

        p.fill(st.badge[0]!, st.badge[1]!, st.badge[2]!, isDimmed ? 20 : 50)
        const badgeLabel = STATUS_LABELS[node.narrative.status] ?? ''
        p.textSize(9)
        const bw = p.textWidth(badgeLabel) + 12
        p.rect(nx + 10, ny + 7, bw, 16, 8)
        p.fill(st.dot[0]!, st.dot[1]!, st.dot[2]!, isDimmed ? 60 : 220)
        p.textAlign(p.CENTER, p.CENTER)
        p.text(badgeLabel, nx + 10 + bw / 2, ny + 7 + 8)

        if (node.narrative.related_tags.length > 0) {
          p.fill(186, 206, 226, isDimmed ? 30 : 60)
          p.textSize(9)
          p.textAlign(p.RIGHT, p.CENTER)
          p.text(`${node.narrative.related_tags.length}标签`, nx + node.w - 8, ny + 7 + 8)
        }

        p.fill(241, 247, 252, isDimmed ? 40 : 220)
        p.textSize(11)
        p.textAlign(p.LEFT, p.TOP)
        const titleLines = wrapText(node.narrative.title, node.w - 16, 2)
        for (let li = 0; li < titleLines.length; li++) {
          p.text(titleLines[li]!, nx + 10, ny + 26 + li * 14)
        }

        p.pop()
      }

      /* resize canvas when layout changes */
      if (p.width !== canvasW || p.height !== canvasH) {
        p.resizeCanvas(canvasW, canvasH)
      }
    }

    p.mouseMoved = () => {
      if (!canvasEl) return
      const hit = hitTest(p.mouseX, p.mouseY)
      if (hit !== hoveredId) {
        hoveredId = hit
        computeHighlight(hit)
        canvasEl.style.cursor = hit !== null ? 'pointer' : 'default'
        emit('hover', hit)
      }
    }

    p.mousePressed = () => {
      const hit = hitTest(p.mouseX, p.mouseY)
      if (hit !== null) emit('select', hit)
    }

    watch(() => props.days, () => {
      computeLayout()
      if (p5Instance.value) {
        p5Instance.value.resizeCanvas(canvasW, canvasH)
      }
      computeHighlight(hoveredId)
    }, { deep: true })

    watch(() => props.selectedId, () => {
      /* draw() reads props.selectedId directly */
    })
  }

  p5Instance.value = new P5(sketch)
}

watch(containerRef, (el) => {
  if (el) initP5()
}, { immediate: true })
</script>

<template>
  <div ref="containerRef" class="narrative-canvas-wrap" />
</template>

<style scoped>
.narrative-canvas-wrap {
  position: relative;
  width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(10, 15, 22, 0.8);
}

.narrative-canvas-wrap :deep(canvas) {
  display: block;
}
</style>
