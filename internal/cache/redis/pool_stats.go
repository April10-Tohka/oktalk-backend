// Package redis 提供连接池统计信息
package redis

// PoolStatsInfo 连接池统计信息
type PoolStatsInfo struct {
	Hits       uint32 `json:"hits"`        // 命中次数
	Misses     uint32 `json:"misses"`      // 未命中次数
	Timeouts   uint32 `json:"timeouts"`    // 超时次数
	TotalConns uint32 `json:"total_conns"` // 总连接数
	IdleConns  uint32 `json:"idle_conns"`  // 空闲连接数
	StaleConns uint32 `json:"stale_conns"` // 过期连接数
}

// HitRate 计算命中率
func (s *PoolStatsInfo) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// Usage 计算连接使用率
func (s *PoolStatsInfo) Usage() float64 {
	if s.TotalConns == 0 {
		return 0
	}
	activeConns := s.TotalConns - s.IdleConns
	return float64(activeConns) / float64(s.TotalConns)
}
