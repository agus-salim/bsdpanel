<template>
  <component
    :is="resolvedIcon"
    v-if="isLucideComponent"
    :size="size"
    :color="color"
    :stroke-width="strokeWidth"
    class="app-icon"
  />
  <Icon
    v-else
    :icon="iconifyName"
    :width="size"
    :height="size"
    :color="color"
    class="app-icon"
  />
</template>

<script setup>
import { computed } from 'vue'
import { Icon } from '@iconify/vue'
import * as LucideIcons from '@lucide/vue'

const props = defineProps({
  name: {
    type: String,
    required: true
  },
  size: {
    type: [Number, String],
    default: 18
  },
  color: {
    type: String,
    default: 'currentColor'
  },
  strokeWidth: {
    type: [Number, String],
    default: 2
  }
})

// Normalize name: e.g. "server" -> "Server", "globe" -> "Globe"
const pascalName = computed(() => {
  if (!props.name) return 'HelpCircle'
  const clean = props.name.replace(/[^a-zA-Z0-9]/g, '')
  return clean.charAt(0).toUpperCase() + clean.slice(1)
})

const isLucideComponent = computed(() => {
  return Boolean(LucideIcons[pascalName.value])
})

const resolvedIcon = computed(() => {
  return LucideIcons[pascalName.value] || LucideIcons.HelpCircle
})

const iconifyName = computed(() => {
  if (props.name.includes(':')) {
    return props.name
  }
  return `lucide:${props.name.toLowerCase().replace(/[A-Z]/g, m => `-${m.toLowerCase()}`).replace(/^-/, '')}`
})
</script>

<style scoped>
.app-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
}
</style>
