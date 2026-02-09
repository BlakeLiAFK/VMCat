/// <reference types="vite/client" />

declare module '@novnc/novnc/lib/rfb.js' {
    export default class RFB {
        constructor(target: HTMLElement, url: string, options?: object)
        scaleViewport: boolean
        resizeSession: boolean
        addEventListener(type: string, listener: (e: any) => void): void
        disconnect(): void
        sendCredentials(credentials: { password?: string; username?: string; target?: string }): void
        sendCtrlAltDel(): void
    }
}

declare module '*.vue' {
    import type {DefineComponent} from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}
