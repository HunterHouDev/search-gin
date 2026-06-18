// Package consts 遗留 — 应用常量请使用 internal/service 中对应函数
//
// 主要迁移说明：
//   OSSetting / GetOSSetting / SetOSSetting / UpdateOSSetting → service.GetOSSetting 等
//   TokenStore / SetToken / ValidateTokenWithInfo           → service.SetToken 等
//   TypeMenu / TagMenu / SeriesCount / LogMem               → service.TypeMenu 等
//   ScanProgress (Sp)                                       → service.Sp
//   TransferTask / TransferTaskMutex                         → service.TransferTask
//
// 本包仅保留端口、文件类型等纯常量。
package consts
