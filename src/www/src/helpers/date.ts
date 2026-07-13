export function dateFormat(date: string) {
  const [year, month, day] = date.split('T')[0].split('-')
  const d = new Date()
  d.setFullYear(+year)
  d.setMonth(+month - 1)
  d.setDate(+day)

  return d.toLocaleDateString('en-US', {
    day: '2-digit',
    year: 'numeric',
    month: 'short',
  })
}
