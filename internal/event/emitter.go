package event

// Emitter 事件发射器接口
type Emitter interface {
	Emit(event string, data ...interface{})
}

// NoopEmitter 空实现（服务端模式使用，前端通过轮询获取状态）
type NoopEmitter struct{}

func (e *NoopEmitter) Emit(event string, data ...interface{}) {}
