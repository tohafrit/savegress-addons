package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"getchainlens.com/chainlens/pkg/workerpool"
	"getchainlens.com/chainlens/pkg/workerpool/enterprise/dashboard"
	"getchainlens.com/chainlens/pkg/workerpool/enterprise/observability"
)

func main() {
	// Create enterprise configuration
	config := workerpool.NewEnterpriseConfig()

	// Customize configuration
	config.Workers = 10
	config.QueueSize = 1000

	// Enable observability
	config.Telemetry.Enabled = true
	config.Telemetry.ServiceName = "my-service"
	config.Telemetry.LogLevel = "info"

	// Enable rate limiting
	config.RateLimit.Enabled = true
	config.RateLimit.Rate = 100.0 // 100 tasks per second
	config.RateLimit.Burst = 50

	// Enable retry
	config.Retry.Enabled = true
	config.Retry.MaxRetries = 3
	config.Retry.InitialDelay = 100 * time.Millisecond

	// Enable circuit breaker
	config.CircuitBreaker.Enabled = true
	config.CircuitBreaker.FailureThreshold = 5
	config.CircuitBreaker.Timeout = 30 * time.Second

	// Enable cost tracking
	config.Cost.Enabled = true
	config.Cost.CPUCostPerMs = 0.00001
	config.Cost.MemoryCostPerMB = 0.000001
	config.Cost.TaskCostBase = 0.0001

	// Enable dashboard
	config.Dashboard.Enabled = true
	config.Dashboard.Port = 8080

	// Enable multi-tenancy
	config.MultiTenancy.Enabled = true

	// Create enterprise pool
	pool, err := workerpool.NewWorkerPool(config)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Stop()

	fmt.Println("Enterprise Worker Pool started")
	fmt.Println("Dashboard available at: http://localhost:8080/dashboard")
	fmt.Println("Health check at: http://localhost:8080/api/metrics")

	// Add alert channels
	alertManager := pool.GetDashboard()
	if alertManager != nil {
		// Add a log channel for alerts
		logChannel := &dashboard.LogChannel{}
		// Note: alertManager doesn't directly expose AddChannel in current implementation
		// This is just an example of how it could be used
		_ = logChannel
	}

	// Example 1: Submit simple tasks
	fmt.Println("\n=== Example 1: Simple Task Submission ===")
	for i := 0; i < 5; i++ {
		taskID := i
		err := pool.SubmitWithContext(context.Background(), func() error {
			fmt.Printf("Task %d executing\n", taskID)
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		if err != nil {
			log.Printf("Failed to submit task %d: %v", taskID, err)
		}
	}

	// Example 2: Multi-tenant tasks
	if config.MultiTenancy.Enabled {
		fmt.Println("\n=== Example 2: Multi-Tenant Tasks ===")

		tenants := []string{"tenant-a", "tenant-b", "tenant-c"}
		for _, tenant := range tenants {
			for i := 0; i < 3; i++ {
				tenantID := tenant
				taskID := i
				err := pool.SubmitWithTenant(context.Background(), tenantID, func() error {
					fmt.Printf("Task %d for tenant %s executing\n", taskID, tenantID)
					time.Sleep(50 * time.Millisecond)
					return nil
				})
				if err != nil {
					log.Printf("Failed to submit task for tenant %s: %v", tenantID, err)
				}
			}
		}
	}

	// Example 3: Tasks with context cancellation
	fmt.Println("\n=== Example 3: Tasks with Context ===")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = pool.SubmitWithContext(ctx, func() error {
		fmt.Println("Long-running task started")
		time.Sleep(5 * time.Second)
		fmt.Println("Long-running task completed (should not reach here)")
		return nil
	})
	if err != nil {
		log.Printf("Task submission failed: %v", err)
	}

	// Wait a bit for tasks to complete
	time.Sleep(2 * time.Second)

	// Example 4: Get metrics
	fmt.Println("\n=== Example 4: Metrics ===")
	metrics := pool.GetMetrics().(observability.Metrics)
	fmt.Printf("Active Workers: %d\n", metrics.WorkersActive)
	fmt.Printf("Queued Tasks: %d\n", metrics.QueueSize)
	fmt.Printf("Completed Tasks: %d\n", metrics.TasksCompleted)
	fmt.Printf("Rejected Tasks: %d\n", metrics.TasksRejected)
	fmt.Printf("Task Duration P95: %.2fms\n", metrics.TaskDurationP95*1000)

	// Example 5: Health checks
	fmt.Println("\n=== Example 5: Health Checks ===")
	healthChecker := pool.GetHealthChecker()
	fmt.Printf("Liveness: %v\n", healthChecker.Liveness())
	fmt.Printf("Readiness: %v\n", healthChecker.Readiness())
	fmt.Printf("Startup: %v\n", healthChecker.Startup())

	healthStatus := healthChecker.GetStatus()
	fmt.Printf("Health Status: %s\n", healthStatus.Status)
	fmt.Printf("Uptime: %s\n", healthStatus.Uptime)

	// Example 6: Cost tracking
	if config.Cost.Enabled {
		fmt.Println("\n=== Example 6: Cost Tracking ===")
		costTracker := pool.GetCostTracker()

		// Get cost summary
		now := time.Now()
		summary := costTracker.GetCostSummary(now.Add(-1*time.Hour), now)
		fmt.Printf("Total Tasks: %d\n", summary.TotalTasks)
		fmt.Printf("Total Cost: $%.6f\n", summary.TotalCost)
		fmt.Printf("CPU Cost: $%.6f\n", summary.CPUCost)
		fmt.Printf("Memory Cost: $%.6f\n", summary.MemoryCost)

		if len(summary.PerTenant) > 0 {
			fmt.Println("\nPer-Tenant Costs:")
			for tenant, cost := range summary.PerTenant {
				fmt.Printf("  %s: $%.6f\n", tenant, cost)
			}
		}
	}

	// Example 7: Submit many tasks to test performance
	fmt.Println("\n=== Example 7: Performance Test ===")
	numTasks := 100
	start := time.Now()

	for i := 0; i < numTasks; i++ {
		taskID := i
		pool.SubmitWithContext(context.Background(), func() error {
			// Simulate work
			time.Sleep(10 * time.Millisecond)
			if taskID%50 == 0 {
				fmt.Printf("Processed %d tasks...\n", taskID)
			}
			return nil
		})
	}

	// Wait for all tasks to complete
	pool.Wait()
	elapsed := time.Since(start)

	fmt.Printf("\nProcessed %d tasks in %v\n", numTasks, elapsed)
	fmt.Printf("Throughput: %.2f tasks/second\n", float64(numTasks)/elapsed.Seconds())

	// Final metrics
	fmt.Println("\n=== Final Metrics ===")
	finalMetrics := pool.GetMetrics().(observability.Metrics)
	fmt.Printf("Total Completed: %d\n", finalMetrics.TasksCompleted)
	fmt.Printf("Total Rejected: %d\n", finalMetrics.TasksRejected)
	fmt.Printf("Average Task Duration: %.2fms\n", finalMetrics.TaskDurationP50*1000)
	fmt.Printf("P99 Task Duration: %.2fms\n", finalMetrics.TaskDurationP99*1000)

	fmt.Println("\nPress Ctrl+C to exit (or wait for 30 seconds to see dashboard updates)")
	time.Sleep(30 * time.Second)
}
