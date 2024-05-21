package main

import (
	"fmt"
	"sync"
	"time"
	"workerpool/workerpool"
)

var (
	minWorkers       = 2
	maxWorkers       = 10
	maxQueueSize     = 20
	expandThreshold  = 0.8 // Expand when the queue is 80% full
	shrinkThreshold  = 0.2 // Shrink when the queue is 20% full
	expandStep       = 2   // Number of workers to add when expanding
	shrinkStep       = 1   // Number of workers to remove when shrinking
	monitorFrequency = 1 * time.Second

	mu            sync.Mutex
	queueSize     int
	activeWorkers int
	jobCount      int
)

func main() {

	pool := workerpool.New(minWorkers, maxQueueSize)

	// Initialize activeWorkers to minWorkers
	activeWorkers = minWorkers

	// Function to adjust pool size dynamically based on average queue size
	adjustPoolSize := func() {
		mu.Lock()
		defer mu.Unlock()

		// Check if we need to expand the pool
		if queueSize > int(float64(maxQueueSize)*expandThreshold) && activeWorkers < maxWorkers {
			newWorkers := activeWorkers + expandStep
			if newWorkers > maxWorkers {
				newWorkers = maxWorkers
			}
			if pool.Expand(newWorkers-activeWorkers, 0, nil) {
				activeWorkers = newWorkers
				fmt.Printf("Pool expanded to %d workers\n", activeWorkers)
			}
			return
		}

		// Check if we need to shrink the pool
		if queueSize < int(float64(maxQueueSize)*shrinkThreshold) && activeWorkers > minWorkers {
			newWorkers := activeWorkers - shrinkStep
			if newWorkers < minWorkers {
				newWorkers = minWorkers
			}
			// Prepare a quit channel to signal workers to stop
			quit := make(chan struct{})
			if pool.Expand(-shrinkStep, 0, quit) {
				activeWorkers = newWorkers
				fmt.Printf("Pool shrunk to %d workers\n", activeWorkers)
				// Close the quit channel after sending the signal to stop
				close(quit)
			}
			return
		}
	}
	// Monitor the queue size and adjust pool size accordingly
	go func() {
		for {
			time.Sleep(monitorFrequency)
			mu.Lock()
			queueSize = pool.PoolSize()
			jobCount = pool.CompletedJobs()
			mu.Unlock()

			// Adjust the pool size based on the average queue size and workload characteristics
			adjustPoolSize()
		}
	}()

	// Queue some jobs
	for i := 0; i < 200; i++ {
		jobID := i
		pool.Queue(func() {
			fmt.Printf("Job %d started\n", jobID)
			time.Sleep(time.Second)
			fmt.Printf("Job %d finished\n", jobID)
		}, 0) // No timeout
	}

	time.Sleep(10 * time.Second) // Let it run for a while
	pool.Stop()
	fmt.Printf("Total jobs processed: %d\n", jobCount)
}
