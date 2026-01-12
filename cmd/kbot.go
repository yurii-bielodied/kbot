package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	telebot "gopkg.in/telebot.v4"
)

var (
	// Teletoken bot
	TeleToken = os.Getenv("TELE_TOKEN")
	// MetricsPort - порт для metrics сервера (default: 8080)
	MetricsPort = getEnvOrDefault("METRICS_PORT", "8080")
)

// getEnvOrDefault повертає значення змінної середовища або default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// kbotCmd represents the kbot command
var kbotCmd = &cobra.Command{
	Use:     "kbot",
	Aliases: []string{"start"},
	Short:   "Telegram bot with observability",
	Long: `A Telegram bot with full observability stack.

Features:
  - Prometheus metrics on /metrics endpoint
  - OpenTelemetry distributed tracing
  - Health checks on /health and /ready

Commands:
  hello     - Get a greeting from the bot
  time      - Get system time`,
	Run: func(cmd *cobra.Command, args []string) {
		// Створюємо контекст з можливістю скасування
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Обробка сигналів для graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		if TeleToken == "" {
			log.Fatal("TELE_TOKEN environment variable is not set")
		}

		fmt.Printf("kbot %s starting...\n", appVersion)

		// =====================================================
		// ЗАПУСК METRICS SERVER
		// =====================================================
		StartMetricsServer(MetricsPort)
		fmt.Printf("Metrics server started on port %s\n", MetricsPort)

		// =====================================================
		// ІНІЦІАЛІЗАЦІЯ OPENTELEMETRY TRACING
		// =====================================================
		var shutdownTracer func(context.Context) error
		if IsTracingEnabled() {
			var err error
			shutdownTracer, err = InitTracer(ctx)
			if err != nil {
				log.Printf("Warning: Failed to initialize tracer: %v", err)
			} else {
				fmt.Println("OpenTelemetry tracing enabled")
			}
		} else {
			fmt.Println("OpenTelemetry tracing disabled (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
		}
		// =====================================================
		// ІНІЦІАЛІЗАЦІЯ TELEGRAM BOT
		// =====================================================
		kbot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  TeleToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			log.Fatalf("Failed to create bot: %s", err)
		}

		// =====================================================
		// ОБРОБНИК ПОВІДОМЛЕНЬ З TRACING ТА METRICS
		// =====================================================
		kbot.Handle(telebot.OnText, func(m telebot.Context) error {
			// Створюємо span для всієї обробки повідомлення
			ctx, span := StartSpan(ctx, "handle_message",
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			// Записуємо час початку для метрик
			startTime := time.Now()

			// Отримуємо TraceID для логування
			traceID := GetTraceID(ctx)

			// Додаємо атрибути до span
			span.SetAttributes(
				attribute.Int64("telegram.user_id", m.Sender().ID),
				attribute.String("telegram.username", m.Sender().Username),
				attribute.String("telegram.message", m.Text()),
			)

			// Логуємо з TraceID для кореляції
			if traceID != "" {
				log.Printf("[TraceID: %s] Received message from %s: %s",
					traceID, m.Sender().Username, m.Text())
			} else {
				log.Printf("Received message from %s: %s",
					m.Sender().Username, m.Text())
			}

			command := strings.TrimSpace(m.Message().Payload)
			if command == "" {
				command = strings.TrimPrefix(m.Text(), "/")
			}
			command = strings.ToLower(command)

			// payload := m.Message().Payload
			var sendErr error
			// var command string
			var response string

			// Обробка команд з вкладеними spans
			// switch payload {
			switch command {
			case "hello":
				command = "hello"
				_, cmdSpan := StartSpan(ctx, "command_hello")
				response = fmt.Sprintf("Hello I'm Kbot %s!", appVersion)
				sendErr = m.Send(response)
				if sendErr != nil {
					cmdSpan.RecordError(sendErr)
					cmdSpan.SetStatus(codes.Error, sendErr.Error())
				}
				cmdSpan.End()

			case "time":
				command = "time"
				_, cmdSpan := StartSpan(ctx, "command_time")
				response = fmt.Sprintf("Current time is %v", time.Now().Format("2006-01-02 15:04:05"))
				sendErr = m.Send(response)
				if sendErr != nil {
					cmdSpan.RecordError(sendErr)
					cmdSpan.SetStatus(codes.Error, sendErr.Error())
				}
				cmdSpan.End()

			default:
				command = "unknown"
				_, cmdSpan := StartSpan(ctx, "command_default")
				response = "Hello from Kbot! Try hello or time"
				sendErr = m.Send(response)
				if sendErr != nil {
					cmdSpan.RecordError(sendErr)
					cmdSpan.SetStatus(codes.Error, sendErr.Error())
				}
				cmdSpan.End()
			}

			// Додаємо результат до головного span
			span.SetAttributes(
				attribute.String("command", command),
				attribute.String("response", response),
			)

			// =====================================================
			// ЗАПИС МЕТРИК
			// =====================================================
			duration := time.Since(startTime)
			status := "success"
			if sendErr != nil {
				status = "error"
				span.RecordError(sendErr)
				span.SetStatus(codes.Error, sendErr.Error())
			} else {
				span.SetStatus(codes.Ok, "")
			}
			RecordMessage(command, status, duration)

			// Логуємо завершення з TraceID
			if traceID != "" {
				log.Printf("[TraceID: %s] Processed %s command in %v (status: %s)",
					traceID, command, duration, status)
			}

			return sendErr
		})

		fmt.Printf("kbot %s started successfully\n", appVersion)

		// Запускаємо бота в горутині
		go kbot.Start()

		// Чекаємо на сигнал завершення
		<-sigChan
		fmt.Println("\nShutting down...")

		// Graceful shutdown
		if kbot != nil {
			kbot.Stop()
		}

		if shutdownTracer != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := shutdownTracer(shutdownCtx); err != nil {
				log.Printf("Error shutting down tracer: %v", err)
			}
		}

		fmt.Println("Goodbye!")
	},
}

func init() {
	rootCmd.AddCommand(kbotCmd)
}
