package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	externalAPIRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "external_api_requests_total",
			Help: "Total number of external API requests",
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(externalAPIRequests)
}

func initTracer() (*trace.TracerProvider, error) {
	// Create gRPC exporter to connect to Tempo
	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint("tempo:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Failed to create trace exporter: %v", err)
		return nil, err
	}

	// Create TracerProvider with the exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("go-platform-api"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	log.Println("Tracer initialized with gRPC connection to Tempo at tempo:4317")
	return tp, nil
}

func getTraceID(c *fiber.Ctx) string {
	span := oteltrace.SpanFromContext(c.UserContext())
	if span.SpanContext().IsValid() {
		traceID := span.SpanContext().TraceID().String()
		log.Printf("Generated traceID: %s", traceID)
		return traceID
	}
	log.Println("No valid span context found")
	// For testing, return a test traceID
	return "test-trace-id-12345"
}

func prometheusMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())

		requestsTotal.WithLabelValues(c.Method(), c.Path(), status).Inc()
		requestDuration.WithLabelValues(c.Method(), c.Path(), status).Observe(duration)

		return err
	}
}

func loggerMiddleware() fiber.Handler {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	return logger.New(logger.Config{
		Format: "${time} ${method} ${path} ${status} ${latency} ${ip} ${user_agent}\n",
		Output: os.Stdout,
	})
}

func simulateExternalAPICall(c *fiber.Ctx) error {
	tracer := otel.Tracer("external-api")
	_, span := tracer.Start(c.UserContext(), "external_api_call")
	defer span.End()

	scenarios := []int{200, 401, 500, 0} // 0 represents timeout
	weights := []int{60, 15, 20, 5}      // probability weights

	rand.Seed(time.Now().UnixNano())
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}

	r := rand.Intn(totalWeight)
	cumulative := 0
	selectedScenario := 200

	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			selectedScenario = scenarios[i]
			break
		}
	}

	span.SetAttributes(
		attribute.String("external.api.scenario", fmt.Sprintf("%d", selectedScenario)),
		attribute.String("external.api.url", "https://httpbin.org/status/"+fmt.Sprintf("%d", selectedScenario)),
	)

	traceID := getTraceID(c)

	switch selectedScenario {
	case 0:
		time.Sleep(6 * time.Second)
		externalAPIRequests.WithLabelValues("timeout").Inc()
		span.SetAttributes(attribute.String("error", "timeout"))
		return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
			"error":   "External API timeout",
			"code":    "TIMEOUT",
			"traceId": traceID,
		})
	case 401:
		externalAPIRequests.WithLabelValues("401").Inc()
		span.SetAttributes(attribute.String("error", "unauthorized"))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "External API unauthorized",
			"code":    "UNAUTHORIZED",
			"traceId": traceID,
		})
	case 500:
		externalAPIRequests.WithLabelValues("500").Inc()
		span.SetAttributes(attribute.String("error", "internal_server_error"))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "External API internal server error",
			"code":    "INTERNAL_ERROR",
			"traceId": traceID,
		})
	default:
		delay := rand.Intn(3000) + 100
		time.Sleep(time.Duration(delay) * time.Millisecond)
		externalAPIRequests.WithLabelValues("200").Inc()
		span.SetAttributes(attribute.Int("response.delay_ms", delay))
		return c.JSON(fiber.Map{
			"message": "External API call successful",
			"traceId": traceID,
			"data": fiber.Map{
				"timestamp": time.Now().Unix(),
				"delay_ms":  delay,
				"random_id": rand.Intn(10000),
			},
		})
	}
}

func healthCheck(c *fiber.Ctx) error {
	log.Println("Health check called")
	traceID := getTraceID(c)
	log.Printf("TraceID for health check: %s", traceID)
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "go-platform-api",
		"version":   "1.0.0",
		"traceId":   traceID,
		"test":      "added-field",
	})
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatal("Failed to initialize tracer:", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	app := fiber.New(fiber.Config{
		AppName:      "Go Platform API",
		ServerHeader: "Fiber",
	})

	app.Use(recover.New())
	app.Use(loggerMiddleware())
	app.Use(prometheusMiddleware())
	app.Use(otelfiber.Middleware())

	app.Get("/health", healthCheck)
	app.Get("/api/external", simulateExternalAPICall)

	app.Get("/metrics", func(c *fiber.Ctx) error {
		metricFamilies, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			return c.Status(500).SendString("Error gathering metrics: " + err.Error())
		}

		var buf bytes.Buffer
		for _, mf := range metricFamilies {
			if _, err := expfmt.MetricFamilyToText(&buf, mf); err != nil {
				return c.Status(500).SendString("Error formatting metrics: " + err.Error())
			}
		}

		c.Set("Content-Type", string(expfmt.FmtText))
		return c.SendString(buf.String())
	})

	go func() {
		if err := app.Listen(":8080"); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exited")
}
