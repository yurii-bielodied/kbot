package cmd

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// =====================================================
// PROMETHEUS METRICS ДЛЯ KBOT
// =====================================================
// Цей файл додає інструментацію для експорту метрик
// Prometheus збиратиме ці метрики з endpoint /metrics
// =====================================================

var (
	// messagesTotal - лічильник всіх отриманих повідомлень
	// Labels: command (hello, time, unknown), status (success, error)
	messagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kbot_messages_total",
			Help: "Total number of messages received by the bot",
		},
		[]string{"command", "status"},
	)

	// messageProcessingDuration - час обробки повідомлення
	// Гістограма для аналізу latency
	messageProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kbot_message_processing_duration_seconds",
			Help:    "Time spent processing messages",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"command"},
	)

	// activeUsers - gauge для кількості унікальних користувачів за останню хвилину
	// (опціонально - потребує додаткової логіки)
	botInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kbot_info",
			Help: "Information about kbot instance",
		},
		[]string{"version"},
	)

	// botUptime - час роботи бота
	botStartTime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "kbot_start_time_seconds",
			Help: "Start time of the bot in unix timestamp",
		},
	)
)

// StartMetricsServer запускає HTTP сервер для метрик на порту 8080
// Prometheus буде скрейпити /metrics endpoint
func StartMetricsServer(port string) {
	// Встановлюємо час запуску
	botStartTime.Set(float64(time.Now().Unix()))

	// Встановлюємо інформацію про версію
	botInfo.WithLabelValues(appVersion).Set(1)

	// Створюємо mux для роутингу
	mux := http.NewServeMux()

	// /metrics - endpoint для Prometheus
	mux.Handle("/metrics", promhttp.Handler())

	// /health - endpoint для liveness probe
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// /ready - endpoint для readiness probe
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// Запускаємо сервер в горутині
	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			// Логуємо помилку, але не падаємо - бот може працювати без метрик
			println("Metrics server error:", err.Error())
		}
	}()
}

// RecordMessage записує метрику про оброблене повідомлення
func RecordMessage(command string, status string, duration time.Duration) {
	messagesTotal.WithLabelValues(command, status).Inc()
	messageProcessingDuration.WithLabelValues(command).Observe(duration.Seconds())
}
