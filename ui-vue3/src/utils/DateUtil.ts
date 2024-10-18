import dayjs from 'dayjs'

// Format time
export const formattedDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}
