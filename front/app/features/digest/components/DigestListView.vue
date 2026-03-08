<script setup lang="ts">
import DigestList from '~/components/digest/DigestList.vue'
import type { DigestType } from '~/api/digest'

const props = withDefaults(defineProps<{
  lockedType?: DigestType | null
  showDigestBack?: boolean
}>(), {
  lockedType: null,
  showDigestBack: false,
})
</script>

<template>
  <DigestList :locked-type="props.lockedType" :show-digest-back="props.showDigestBack" />
</template>
