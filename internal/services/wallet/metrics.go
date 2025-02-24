package wallet

import "time"

// NoopMetricsCollector is a no-op implementation of MetricsCollector
type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) RecordOperationDuration(string, time.Duration) {}
func (n *NoopMetricsCollector) RecordOperationResult(string, string)          {}
func (n *NoopMetricsCollector) RecordCacheHit(string)                         {}
func (n *NoopMetricsCollector) RecordCacheMiss(string)                        {}
func (n *NoopMetricsCollector) RecordBalanceChange(uint, float64, float64)    {}
func (n *NoopMetricsCollector) RecordError(string, string)                    {}
func (n *NoopMetricsCollector) RecordTransactionVolume(float64)               {}
func (n *NoopMetricsCollector) RecordDailyVolume(uint, float64)               {}
