import { reactive } from 'vue'

const state = reactive({
  open: false,
  tasks: [],
})

export const useTaskCenter = () => state

export const openTaskCenter = () => {
  state.open = true
}

export const closeTaskCenter = () => {
  state.open = false
}

export const addTask = (task) => {
  state.tasks.unshift({
    id: task.id,
    name: task.name,
    type: task.type,
    progress: task.progress ?? 0,
    status: task.status ?? 'running',
    sizeLabel: task.sizeLabel ?? '',
  })
}

export const updateTask = (id, updates) => {
  const item = state.tasks.find((task) => task.id === id)
  if (!item) return
  Object.assign(item, updates)
}

export const completeTask = (id) => {
  updateTask(id, { progress: 100, status: 'done' })
}

export const failTask = (id) => {
  updateTask(id, { status: 'error' })
}

export const clearCompleted = () => {
  state.tasks = state.tasks.filter((task) => task.status !== 'done')
}
