package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Dashboard provides real-time metrics visualization
type Dashboard struct {
	clients   sync.Map // map[string]chan DashboardUpdate
	broadcast chan DashboardUpdate
	stopChan  chan struct{}
	wg        sync.WaitGroup

	metricsProvider MetricsProvider
}

// MetricsProvider provides metrics for the dashboard
type MetricsProvider interface {
	GetMetrics() interface{}
}

// DashboardUpdate represents an update to send to clients
type DashboardUpdate struct {
	Timestamp time.Time   `json:"timestamp"`
	Type      string      `json:"type"` // "stats", "task", "alert"
	Data      interface{} `json:"data"`
}

// NewDashboard creates a new dashboard
func NewDashboard(provider MetricsProvider) *Dashboard {
	return &Dashboard{
		broadcast:       make(chan DashboardUpdate, 100),
		stopChan:        make(chan struct{}),
		metricsProvider: provider,
	}
}

// Start starts the dashboard
func (d *Dashboard) Start(addr string) error {
	// Start broadcast loop
	d.wg.Add(1)
	go d.broadcastLoop()

	// Start metrics loop
	d.wg.Add(1)
	go d.metricsLoop()

	// Setup HTTP handlers
	http.HandleFunc("/api/metrics", d.handleMetrics)
	http.HandleFunc("/api/ws", d.handleWebSocket)
	http.HandleFunc("/dashboard", d.serveDashboard)

	return http.ListenAndServe(addr, nil)
}

// Stop stops the dashboard
func (d *Dashboard) Stop() {
	close(d.stopChan)
	d.wg.Wait()
}

// broadcastLoop broadcasts updates to all clients
func (d *Dashboard) broadcastLoop() {
	defer d.wg.Done()

	for {
		select {
		case <-d.stopChan:
			return
		case update := <-d.broadcast:
			d.clients.Range(func(key, value interface{}) bool {
				ch := value.(chan DashboardUpdate)
				select {
				case ch <- update:
				default:
					// Client slow or disconnected, skip
				}
				return true
			})
		}
	}
}

// metricsLoop periodically sends metrics updates
func (d *Dashboard) metricsLoop() {
	defer d.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			if d.metricsProvider != nil {
				metrics := d.metricsProvider.GetMetrics()
				update := DashboardUpdate{
					Timestamp: time.Now(),
					Type:      "stats",
					Data:      metrics,
				}
				d.broadcast <- update
			}
		}
	}
}

// handleMetrics returns current metrics as JSON
func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if d.metricsProvider == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "no metrics provider"})
		return
	}

	metrics := d.metricsProvider.GetMetrics()
	json.NewEncoder(w).Encode(metrics)
}

// handleWebSocket handles WebSocket connections
func (d *Dashboard) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Simplified WebSocket implementation
	// In production, use a proper WebSocket library like gorilla/websocket

	clientID := r.RemoteAddr
	updateChan := make(chan DashboardUpdate, 10)
	d.clients.Store(clientID, updateChan)

	defer func() {
		d.clients.Delete(clientID)
		close(updateChan)
	}()

	// Send updates to client
	// This is a placeholder - implement proper WebSocket handling
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for update := range updateChan {
		data, _ := json.Marshal(update)
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}
}

// serveDashboard serves the dashboard HTML
func (d *Dashboard) serveDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Worker Pool Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .metric-box {
            background: white;
            padding: 20px;
            margin: 10px 0;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .metric-title { font-size: 14px; color: #666; }
        .metric-value { font-size: 32px; font-weight: bold; color: #333; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Worker Pool Dashboard</h1>
        <div class="grid">
            <div class="metric-box">
                <div class="metric-title">Active Workers</div>
                <div class="metric-value" id="active-workers">0</div>
            </div>
            <div class="metric-box">
                <div class="metric-title">Queued Tasks</div>
                <div class="metric-value" id="queued-tasks">0</div>
            </div>
            <div class="metric-box">
                <div class="metric-title">Completed Tasks</div>
                <div class="metric-value" id="completed-tasks">0</div>
            </div>
            <div class="metric-box">
                <div class="metric-title">Rejected Tasks</div>
                <div class="metric-value" id="rejected-tasks">0</div>
            </div>
        </div>
        <div class="metric-box">
            <h3>Real-time Metrics</h3>
            <pre id="metrics"></pre>
        </div>
    </div>
    <script>
        const eventSource = new EventSource('/api/ws');
        eventSource.onmessage = function(event) {
            const update = JSON.parse(event.data);
            if (update.type === 'stats') {
                document.getElementById('metrics').textContent = JSON.stringify(update.data, null, 2);
                // Update individual metrics if available
                if (update.data.WorkersActive !== undefined) {
                    document.getElementById('active-workers').textContent = update.data.WorkersActive;
                }
                if (update.data.QueuedTasks !== undefined) {
                    document.getElementById('queued-tasks').textContent = update.data.QueuedTasks;
                }
                if (update.data.TasksCompleted !== undefined) {
                    document.getElementById('completed-tasks').textContent = update.data.TasksCompleted;
                }
                if (update.data.TasksRejected !== undefined) {
                    document.getElementById('rejected-tasks').textContent = update.data.TasksRejected;
                }
            }
        };
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// SendUpdate sends a custom update to the dashboard
func (d *Dashboard) SendUpdate(updateType string, data interface{}) {
	update := DashboardUpdate{
		Timestamp: time.Now(),
		Type:      updateType,
		Data:      data,
	}

	select {
	case d.broadcast <- update:
	default:
		// Broadcast channel full, skip
	}
}
