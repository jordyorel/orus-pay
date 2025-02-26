package wallet

import (
	"time"
)

// NoopMetricsCollector provides a no-op implementation of MetricsCollector
type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) RecordTransaction(txType string, amount float64)                   {}
func (n *NoopMetricsCollector) RecordError(errType string, errMsg string)                         {}
func (n *NoopMetricsCollector) RecordBalanceChange(walletID uint, oldBalance, newBalance float64) {}
func (n *NoopMetricsCollector) RecordCacheHit(key string)                                         {}
func (n *NoopMetricsCollector) RecordCacheMiss(key string)                                        {}
func (n *NoopMetricsCollector) RecordDailyVolume(userID uint, amount float64)                     {}
func (n *NoopMetricsCollector) RecordOperationDuration(operation string, duration time.Duration)  {}
func (n *NoopMetricsCollector) RecordOperationResult(operation, result string)                    {}
func (n *NoopMetricsCollector) RecordTransactionVolume(amount float64)                            {}
