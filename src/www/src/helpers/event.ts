export const track = (eventName: string, data: any) => {
  // @ts-ignore
  if (typeof window === 'undefined' || !window.umami) {
    return
  }

  // @ts-ignore
  umami.track(eventName, data)
}
