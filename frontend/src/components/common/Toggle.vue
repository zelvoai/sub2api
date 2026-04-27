<template>
  <label class="flex items-start justify-between gap-3">
    <div v-if="label || hint" class="min-w-0 flex-1">
      <div v-if="label" class="input-label">{{ label }}</div>
      <div v-if="hint" class="input-hint mt-1">{{ hint }}</div>
    </div>

    <button
      type="button"
      @click="toggle"
      class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-dark-800"
      :class="[modelValue ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600']"
      role="switch"
      :aria-checked="modelValue"
      :aria-label="label || undefined"
    >
      <span
        class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
        :class="[modelValue ? 'translate-x-5' : 'translate-x-0']"
      />
    </button>
  </label>
</template>

<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  label?: string
  hint?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  emit('update:modelValue', !props.modelValue)
}
</script>
