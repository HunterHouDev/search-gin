package model

// ── SSE 事件类型 ─────────────────────────────────────────────────

const (
	SSEScanStart    = "scan_start"
	SSEScanComplete = "scan_complete"
	SSEScanOneDone  = "scan_one_done"
	SSEScanError    = "scan_error"
	SSEFileChanged  = "file_changed"
	SSEIndexUpdate  = "index_update"
	SSEIndexHealth  = "index_health"
	SSETaskLog      = "task_log"
)

// ── WebSocket 消息类型 ────────────────────────────────────────────

const (
	WSChat      = "chat"
	WSOnline    = "online"
	WSSystem    = "system"
	WSSignal    = "signal"
	WSSignalAll = "signal-all"
)

// ── WebSocket / WebRTC 信令 action ─────────────────────────────────

const (
	SignalActionJoin   = "join"
	SignalActionLeave  = "leave"
	SignalActionOffer  = "offer"
	SignalActionAnswer = "answer"
	SignalActionICE    = "ice"
)
