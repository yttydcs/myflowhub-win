// Context: declares the window.go typings used by the Win frontend Wails bindings.

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
