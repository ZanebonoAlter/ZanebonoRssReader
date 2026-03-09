<script setup lang="ts">
import ArticleListPanelView from '~/features/shell/components/ArticleListPanelView.vue'
import type { Article } from '~/types'

interface Props {
  articles: Article[]
  selectedCategory?: string | null
  selectedFeed?: string | null
  selectedArticle?: Article | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  selectedCategory: null,
  selectedFeed: null,
  selectedArticle: null,
  loading: false,
})
</script>

<template>
  <ArticleListPanelView
    v-bind="$attrs"
    :articles="props.articles"
    :selected-category="props.selectedCategory"
    :selected-feed="props.selectedFeed"
    :selected-article="props.selectedArticle"
    :loading="props.loading"
  />
</template>
