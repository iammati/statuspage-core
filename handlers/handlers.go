package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"infraops.dev/statuspage-core/global"
	"infraops.dev/statuspage-core/utils"
)

type ServiceState struct {
	Host            string
	IsUp            bool
	LastChange      time.Time
	UpdatetimeStart time.Time
	LastRequestTime time.Time
}

type ServiceStates struct {
	states map[string]*ServiceState
	mu     sync.Mutex
}

type ServiceStateChange struct {
	StartTime time.Time
	EndTime   time.Time
	Reason    string
}

var serviceStates = ServiceStates{states: make(map[string]*ServiceState)}

type UpdatetimeEvent struct {
	StartTime time.Time
	EndTime   time.Time
	Reason    string
}

func LogUpdatetimeEvent(event UpdatetimeEvent) {
	file, err := os.OpenFile(global.LOGFILENAME, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file '%s': %v", global.LOGFILENAME, err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("Start: %v, End: %v, Reason: %s\n", event.StartTime, event.EndTime, event.Reason)
	if _, err := file.WriteString(logEntry); err != nil {
		log.Printf("Failed to write to log file '%s': %v", global.LOGFILENAME, err)
	}
}

func (ss *ServiceStates) UpdateServiceState(host string, isCurrentlyUp bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	currentState, exists := ss.states[host]
	if !exists {
		log.Printf("Added host '%s' to the list of monitored hosts", host)
		LogUpdatetimeEvent(UpdatetimeEvent{
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Reason:    fmt.Sprintf("Added host '%s' to the list of monitored hosts", host),
		})

		ss.states[host] = &ServiceState{
			Host:            host,
			IsUp:            isCurrentlyUp,
			LastChange:      time.Now(),
			UpdatetimeStart: time.Now(),
			LastRequestTime: time.Now(),
		}
		if isCurrentlyUp {
			ss.states[host].UpdatetimeStart = time.Time{}
		}
		LogServiceStateChange(host, isCurrentlyUp, currentState)
		return
	}

	currentState.LastRequestTime = time.Now()

	if currentState.IsUp != isCurrentlyUp {
		LogServiceStateChange(host, isCurrentlyUp, currentState)
		if !isCurrentlyUp {
			currentState.UpdatetimeStart = time.Now()
		} else if !currentState.UpdatetimeStart.IsZero() {
			currentState.UpdatetimeStart = time.Time{}
		}

		currentState.IsUp = isCurrentlyUp
		currentState.LastChange = time.Now()
	}
}

func (ss *ServiceStates) RemoveInactiveHosts() {
	for {
		time.Sleep(5 * time.Second)

		ss.mu.Lock()
		currentTime := time.Now()
		for host, state := range ss.states {
			if currentTime.Sub(state.LastRequestTime) > 5*time.Second {
				delete(ss.states, host)
				log.Printf("Automatically removed host '%s' from the list of monitored hosts", host)
				LogUpdatetimeEvent(UpdatetimeEvent{
					StartTime: state.LastRequestTime,
					EndTime:   currentTime,
					Reason:    fmt.Sprintf("Automatically removed host '%s' from the list of monitored hosts", host),
				})
			}
		}
		ss.mu.Unlock()
	}
}

func LogServiceStateChange(host string, isUp bool, currentState *ServiceState) {
	state := "down"
	if isUp {
		state = "up"
	}
	log.Printf("Host '%s' is %s", host, state)

	var startTime time.Time = time.Now()
	if currentState == nil {
		startTime = global.BOOTUP_TIME
	}

	LogUpdatetimeEvent(UpdatetimeEvent{
		StartTime: startTime,
		EndTime:   time.Now(),
		Reason:    fmt.Sprintf("Host '%s' is %s", host, state),
	})
}

func CleanupInactiveHosts(timeout time.Duration) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			serviceStates.mu.Lock()
			for host, state := range serviceStates.states {
				if now.Sub(state.LastRequestTime) > timeout {
					delete(serviceStates.states, host)

					log.Printf("Removed host '%s' from the list of monitored hosts after %s of inactivty", host, timeout)
					LogUpdatetimeEvent(UpdatetimeEvent{
						StartTime: state.LastRequestTime,
						EndTime:   now,
						Reason:    fmt.Sprintf("Removed host '%s' from the list of monitored hosts after %s of inactivty", host, timeout),
					})
				}
			}
			serviceStates.mu.Unlock()
		}
	}
}

func ensurePort(host string) string {
	if !strings.Contains(host, ":") {
		return host + ":443"
	}
	return host
}

func HandleUp(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		utils.HttpError(w, "Host parameter is required", http.StatusBadRequest)
		return
	}

	hostWithPort := ensurePort(host)
	dnsTime, tcpTime, reachable := utils.HostMetrics(hostWithPort)

	serviceStates.UpdateServiceState(host, reachable)

	responseData := map[string]interface{}{
		"reachable":         reachable,
		"dnsResolutionTime": dnsTime.String(),
		"tcpConnectionTime": tcpTime.String(),
	}

	utils.JsonResponse(w, responseData)
}

func HandleCertInfo(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	if host == "" {
		utils.HttpError(w, "Host parameter is required", http.StatusBadRequest)
		return
	}

	hostWithPort := ensurePort(host)
	_, _, reachable := utils.HostMetrics(hostWithPort)

	if !reachable {
		utils.HttpError(w, "Host is not reachable", http.StatusServiceUnavailable)
		return
	}

	certInfo, err := utils.FetchCertInfo(hostWithPort)
	if err != nil {
		utils.HttpError(w, "Failed to fetch cert info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	utils.JsonResponse(w, certInfo)
}
