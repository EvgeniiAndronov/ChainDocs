package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics коллектор метрик ChainDocs
type Metrics struct {
	BlocksTotal       prometheus.Gauge
	ActiveKeys        prometheus.Gauge
	RegisteredKeys    prometheus.Gauge
	RevokedKeys       prometheus.Gauge
	DocumentsTotal    prometheus.Gauge
	SignaturesTotal   prometheus.Gauge
	ConsensusPercent  prometheus.Gauge
	RequestsTotal     prometheus.Counter
	RequestDuration   prometheus.Histogram
	UploadSize        prometheus.Histogram
}

// New создаёт новые метрики
func New() *Metrics {
	return &Metrics{
		BlocksTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_blocks_total",
			Help: "Total number of blocks in the blockchain",
		}),
		ActiveKeys: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_active_keys",
			Help: "Number of active keys (last 24h)",
		}),
		RegisteredKeys: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_registered_keys_total",
			Help: "Total number of registered keys",
		}),
		RevokedKeys: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_revoked_keys_total",
			Help: "Total number of revoked keys",
		}),
		DocumentsTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_documents_total",
			Help: "Total number of uploaded documents",
		}),
		SignaturesTotal: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_signatures_total",
			Help: "Total number of signatures across all blocks",
		}),
		ConsensusPercent: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "chaindocs_consensus_percent",
			Help: "Current consensus percentage for last block",
		}),
		RequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "chaindocs_requests_total",
			Help: "Total number of API requests",
		}),
		RequestDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "chaindocs_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		UploadSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "chaindocs_upload_size_bytes",
			Help:    "Uploaded document size in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760},
		}),
	}
}

// DefaultMetrics метрики по умолчанию
var DefaultMetrics *Metrics

// Init инициализирует метрики по умолчанию
func Init() {
	DefaultMetrics = New()
}

// UpdateBlocks обновляет метрики блоков
func (m *Metrics) UpdateBlocks(count int) {
	m.BlocksTotal.Set(float64(count))
}

// UpdateKeys обновляет метрики ключей
func (m *Metrics) UpdateKeys(active, registered, revoked int) {
	m.ActiveKeys.Set(float64(active))
	m.RegisteredKeys.Set(float64(registered))
	m.RevokedKeys.Set(float64(revoked))
}

// UpdateConsensus обновляет метрики консенсуса
func (m *Metrics) UpdateConsensus(percent float64, signatures int) {
	m.ConsensusPercent.Set(percent)
	m.SignaturesTotal.Set(float64(signatures))
}

// ObserveRequest записывает метрики запроса
func (m *Metrics) ObserveRequest(duration float64) {
	m.RequestDuration.Observe(duration)
	m.RequestsTotal.Inc()
}

// ObserveUpload записывает метрики загрузки
func (m *Metrics) ObserveUpload(size float64) {
	m.UploadSize.Observe(size)
	m.DocumentsTotal.Inc()
}
