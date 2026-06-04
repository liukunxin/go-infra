package collab

import "fmt"

// Redis key 命名规则。
// 使用 {sid} 花括号确保同一 session 的 key 落在同一 Redis slot（集群模式友好）。

func streamKey(ns, sid string) string   { return fmt.Sprintf("%s:sess:{%s}:events", ns, sid) }
func seqKey(ns, sid string) string      { return fmt.Sprintf("%s:sess:{%s}:seq", ns, sid) }
func snapshotKey(ns, sid string) string  { return fmt.Sprintf("%s:sess:{%s}:snapshot", ns, sid) }
func metaKey(ns, sid string) string      { return fmt.Sprintf("%s:sess:{%s}:meta", ns, sid) }
func dedupKey(ns, eventID string) string { return fmt.Sprintf("%s:dedup:{%s}", ns, eventID) }
