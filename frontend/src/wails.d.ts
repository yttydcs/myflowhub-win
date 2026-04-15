// 本文件声明 Win 前端访问 `window.go` 时使用的类型。

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
