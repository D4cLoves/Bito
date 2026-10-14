export interface PickOption<V> {
  value: V
  label: string
  short?: string
  hint?: string
  disabled?: boolean
  chip?: number
  suit?: string
  red?: boolean
}
