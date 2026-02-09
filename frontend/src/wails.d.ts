export {}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          ProfileState: () => Promise<{
            profiles: string[]
            current: string
            baseDir: string
            settingsPath: string
            keysPath: string
          }>
          SetCurrentProfile: (name: string) => Promise<{
            profiles: string[]
            current: string
            baseDir: string
            settingsPath: string
            keysPath: string
          }>
        }
      }
    }
  }
}
