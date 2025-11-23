package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL     = "http://localhost:8080"
	numRequests = 1000
	concurrent  = 50
	targetRPS   = 100
	duration    = 30 * time.Second
)

type LoadTestResult struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalDuration      time.Duration
	AvgResponseTime    time.Duration
	MinResponseTime    time.Duration
	MaxResponseTime    time.Duration
	P50ResponseTime    time.Duration
	P95ResponseTime    time.Duration
	P99ResponseTime    time.Duration
	ErrorRate          float64
	RequestsPerSecond  float64
	ResponseTimes      []time.Duration
	StatusCodes        map[int]int64
	EndpointStats      map[string]*EndpointStat
}

type EndpointStat struct {
	Total       int64
	Success     int64
	Failed      int64
	StatusCodes map[int]int64
}

type RequestResult struct {
	Success      bool
	ResponseTime time.Duration
	StatusCode   int
	Error        error
	Endpoint     string
}

func main() {
	fmt.Println("=== НАГРУЗОЧНОЕ ТЕСТИРОВАНИЕ СЕРВИСА ===")
	fmt.Printf("Базовый URL: %s\n", baseURL)
	fmt.Printf("Количество запросов: %d\n", numRequests)
	fmt.Printf("Параллелизм: %d\n", concurrent)
	fmt.Printf("Целевой RPS: %d\n", targetRPS)
	fmt.Printf("Длительность теста: %v\n\n", duration)
	fmt.Println("Запуск теста...\n")

	// Создаем тестовые данные перед запуском теста
	fmt.Println("Создание тестовых данных...")
	createTestData()
	fmt.Println("Тестовые данные созданы.\n")

	result := runLoadTest()

	printResults(result)
}

func createTestData() {
	client := &http.Client{Timeout: 5 * time.Second}

	teams := []string{"backend", "frontend", "devops", "qa", "mobile"}

	for i, teamName := range teams {
		req := map[string]interface{}{
			"team_name": teamName,
			"members": []map[string]interface{}{
				{
					"user_id":   fmt.Sprintf("user_%s_1", teamName),
					"username":  fmt.Sprintf("User1_%s", teamName),
					"team_name": teamName,
					"is_active": true,
				},
				{
					"user_id":   fmt.Sprintf("user_%s_2", teamName),
					"username":  fmt.Sprintf("User2_%s", teamName),
					"team_name": teamName,
					"is_active": true,
				},
			},
		}

		body, _ := json.Marshal(req)
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/team/add", baseURL), bytes.NewBuffer(body))
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(httpReq)
		if err == nil && resp != nil {
			resp.Body.Close()
		}

		for j := 0; j < 3; j++ {
			prReq := map[string]interface{}{
				"pull_request_id":   fmt.Sprintf("pr_%s_%d", teamName, j),
				"pull_request_name": fmt.Sprintf("PR #%d for %s", j, teamName),
				"author_id":         fmt.Sprintf("user_%s_1", teamName),
			}
			prBody, _ := json.Marshal(prReq)
			prHttpReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/pullRequest/create", baseURL), bytes.NewBuffer(prBody))
			prHttpReq.Header.Set("Content-Type", "application/json")
			prResp, err := client.Do(prHttpReq)
			if err == nil && prResp != nil {
				prResp.Body.Close()
			}
		}

		if i < len(teams)-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func runLoadTest() *LoadTestResult {
	var (
		totalRequests   int64
		successful      int64
		failed          int64
		responseTimes   = make([]time.Duration, 0, numRequests)
		statusCodes     = make(map[int]int64)
		endpointStats   = make(map[string]*EndpointStat)
		responseTimesMu sync.Mutex
		statusCodesMu   sync.Mutex
		endpointStatsMu sync.Mutex
		wg              sync.WaitGroup
	)

	semaphore := make(chan struct{}, concurrent)
	ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
	defer ticker.Stop()

	startTime := time.Now()

	stopCh := make(chan struct{})
	go func() {
		<-time.After(duration)
		close(stopCh)
	}()

	requestNum := int64(0)

	for {
		select {
		case <-stopCh:
			ticker.Stop()
			goto waitForCompletion
		case <-ticker.C:
			atomic.AddInt64(&requestNum, 1)
			if atomic.LoadInt64(&requestNum) > numRequests {
				ticker.Stop()
				goto waitForCompletion
			}

			wg.Add(1)
			semaphore <- struct{}{}

			go func(reqNum int64) {
				defer wg.Done()
				defer func() { <-semaphore }()

				result := makeRequest(reqNum)
				atomic.AddInt64(&totalRequests, 1)

				responseTimesMu.Lock()
				responseTimes = append(responseTimes, result.ResponseTime)
				responseTimesMu.Unlock()

				statusCodesMu.Lock()
				statusCodes[result.StatusCode]++
				statusCodesMu.Unlock()

				endpointStatsMu.Lock()
				if endpointStats[result.Endpoint] == nil {
					endpointStats[result.Endpoint] = &EndpointStat{
						StatusCodes: make(map[int]int64),
					}
				}
				stat := endpointStats[result.Endpoint]
				stat.Total++
				stat.StatusCodes[result.StatusCode]++
				if result.Success {
					stat.Success++
					atomic.AddInt64(&successful, 1)
				} else {
					stat.Failed++
					atomic.AddInt64(&failed, 1)
				}
				endpointStatsMu.Unlock()
			}(requestNum)
		}
	}

waitForCompletion:
	wg.Wait()
	totalTime := time.Since(startTime)

	responseTimesMu.Lock()
	sort.Slice(responseTimes, func(i, j int) bool {
		return responseTimes[i] < responseTimes[j]
	})
	responseTimesMu.Unlock()

	result := &LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successful,
		FailedRequests:     failed,
		TotalDuration:      totalTime,
		ResponseTimes:      responseTimes,
		StatusCodes:        statusCodes,
		EndpointStats:      endpointStats,
	}

	if totalRequests > 0 {
		var totalDuration int64
		for _, rt := range responseTimes {
			totalDuration += rt.Nanoseconds()
		}
		result.AvgResponseTime = time.Duration(totalDuration / totalRequests)
		result.ErrorRate = float64(failed) / float64(totalRequests)
		result.RequestsPerSecond = float64(totalRequests) / totalTime.Seconds()

		if len(responseTimes) > 0 {
			result.MinResponseTime = responseTimes[0]
			result.MaxResponseTime = responseTimes[len(responseTimes)-1]
			result.P50ResponseTime = percentile(responseTimes, 50)
			result.P95ResponseTime = percentile(responseTimes, 95)
			result.P99ResponseTime = percentile(responseTimes, 99)
		}
	}

	return result
}

func makeRequest(reqNum int64) RequestResult {
	start := time.Now()

	endpoint := selectEndpoint(reqNum)
	method, url, body := endpoint()

	endpointPath := extractEndpointPath(url)

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
		if err != nil {
			return RequestResult{
				Success:      false,
				ResponseTime: time.Since(start),
				Error:        err,
			}
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return RequestResult{
				Success:      false,
				ResponseTime: time.Since(start),
				Error:        err,
			}
		}
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return RequestResult{
			Success:      false,
			ResponseTime: responseTime,
			Error:        err,
			StatusCode:   0,
			Endpoint:     endpointPath,
		}
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return RequestResult{
		Success:      success,
		ResponseTime: responseTime,
		StatusCode:   resp.StatusCode,
		Endpoint:     endpointPath,
	}
}

func extractEndpointPath(url string) string {

	if idx := strings.Index(url, "://"); idx != -1 {
		url = url[idx+3:]
	}

	if idx := strings.Index(url, "/"); idx != -1 {
		path := url[idx:]
		if queryIdx := strings.Index(path, "?"); queryIdx != -1 {
			return path[:queryIdx]
		}
		return path
	}
	return url
}

type EndpointFunc func() (method string, url string, body []byte)

func selectEndpoint(reqNum int64) EndpointFunc {
	mod := reqNum % 10

	switch mod {
	case 0, 1:
		// GET /team/get - 20%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[reqNum%int64(len(teamNames))]
			return "GET", fmt.Sprintf("%s/team/get?team_name=%s", baseURL, teamName), nil
		}
	case 2, 3:
		// GET /users/getReview - 20%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[reqNum%int64(len(teamNames))]
			userID := fmt.Sprintf("user_%s_1", teamName)
			return "GET", fmt.Sprintf("%s/users/getReview?user_id=%s", baseURL, userID), nil
		}
	case 4:
		// POST /users/setIsActive - 10%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[reqNum%int64(len(teamNames))]
			userID := fmt.Sprintf("user_%s_1", teamName)
			req := map[string]interface{}{
				"user_id":   userID,
				"is_active": reqNum%2 == 0,
			}
			body, _ := json.Marshal(req)
			return "POST", fmt.Sprintf("%s/users/setIsActive", baseURL), body
		}
	case 5:
		// POST /team/add - 10%
		return func() (string, string, []byte) {
			req := map[string]interface{}{
				"team_name": fmt.Sprintf("team_%d", reqNum),
				"members": []map[string]interface{}{
					{"user_id": fmt.Sprintf("user_%d_1", reqNum), "username": "User1", "team_name": fmt.Sprintf("team_%d", reqNum), "is_active": true},
					{"user_id": fmt.Sprintf("user_%d_2", reqNum), "username": "User2", "team_name": fmt.Sprintf("team_%d", reqNum), "is_active": true},
				},
			}
			body, _ := json.Marshal(req)
			return "POST", fmt.Sprintf("%s/team/add", baseURL), body
		}
	case 6:
		// POST /pullRequest/create - 10%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[reqNum%int64(len(teamNames))]
			req := map[string]interface{}{
				"pull_request_id":   fmt.Sprintf("pr_load_%d", reqNum),
				"pull_request_name": fmt.Sprintf("PR #%d", reqNum),
				"author_id":         fmt.Sprintf("user_%s_1", teamName),
			}
			body, _ := json.Marshal(req)
			return "POST", fmt.Sprintf("%s/pullRequest/create", baseURL), body
		}
	case 7:
		// POST /pullRequest/merge - 10%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[(reqNum%100)%int64(len(teamNames))]
			prID := fmt.Sprintf("pr_%s_%d", teamName, (reqNum%100)%3)
			req := map[string]interface{}{
				"pull_request_id": prID,
			}
			body, _ := json.Marshal(req)
			return "POST", fmt.Sprintf("%s/pullRequest/merge", baseURL), body
		}
	case 8:
		// POST /pullRequest/reassign - 10%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[(reqNum%100)%int64(len(teamNames))]
			prID := fmt.Sprintf("pr_%s_%d", teamName, (reqNum%100)%3)
			req := map[string]interface{}{
				"pull_request_id": prID,
				"old_reviewer_id": fmt.Sprintf("user_%s_1", teamName),
			}
			body, _ := json.Marshal(req)
			return "POST", fmt.Sprintf("%s/pullRequest/reassign", baseURL), body
		}
	case 9:
		// GET /statistics - 10%
		return func() (string, string, []byte) {
			teamNames := []string{"backend", "frontend", "devops", "qa", "mobile"}
			teamName := teamNames[reqNum%int64(len(teamNames))]
			return "GET", fmt.Sprintf("%s/statistics?team_name=%s", baseURL, teamName), nil
		}
	default:
		return func() (string, string, []byte) {
			return "GET", fmt.Sprintf("%s/team/get?team_name=backend", baseURL), nil
		}
	}
}

func percentile(sortedTimes []time.Duration, p int) time.Duration {
	if len(sortedTimes) == 0 {
		return 0
	}
	index := (p * len(sortedTimes)) / 100
	if index >= len(sortedTimes) {
		index = len(sortedTimes) - 1
	}
	return sortedTimes[index]
}

func printResults(result *LoadTestResult) {
	fmt.Println("=== РЕЗУЛЬТАТЫ НАГРУЗОЧНОГО ТЕСТИРОВАНИЯ ===\n")

	fmt.Printf("Общая статистика:\n")
	fmt.Printf("  Всего запросов:        %d\n", result.TotalRequests)
	fmt.Printf("  Успешных запросов:     %d\n", result.SuccessfulRequests)
	fmt.Printf("  Неудачных запросов:    %d\n", result.FailedRequests)
	fmt.Printf("  Процент ошибок:        %.2f%%\n", result.ErrorRate*100)
	fmt.Printf("  Общая длительность:    %v\n", result.TotalDuration)
	fmt.Printf("  Запросов в секунду:    %.2f RPS\n\n", result.RequestsPerSecond)

	fmt.Printf("Время ответа:\n")
	fmt.Printf("  Минимальное:           %v\n", result.MinResponseTime)
	fmt.Printf("  Среднее:               %v\n", result.AvgResponseTime)
	fmt.Printf("  Максимальное:          %v\n", result.MaxResponseTime)
	fmt.Printf("  P50 (медиана):         %v\n", result.P50ResponseTime)
	fmt.Printf("  P95:                   %v\n", result.P95ResponseTime)
	fmt.Printf("  P99:                   %v\n\n", result.P99ResponseTime)

	fmt.Printf("Распределение HTTP статус-кодов:\n")
	sortedCodes := make([]int, 0, len(result.StatusCodes))
	for code := range result.StatusCodes {
		sortedCodes = append(sortedCodes, code)
	}
	sort.Ints(sortedCodes)
	for _, code := range sortedCodes {
		count := result.StatusCodes[code]
		percentage := float64(count) / float64(result.TotalRequests) * 100
		fmt.Printf("  %d: %d (%.2f%%)\n", code, count, percentage)
	}
	fmt.Println()

	fmt.Printf("Статистика по эндпоинтам:\n")
	sortedEndpoints := make([]string, 0, len(result.EndpointStats))
	for endpoint := range result.EndpointStats {
		sortedEndpoints = append(sortedEndpoints, endpoint)
	}
	sort.Strings(sortedEndpoints)
	for _, endpoint := range sortedEndpoints {
		stat := result.EndpointStats[endpoint]
		successRate := float64(stat.Success) / float64(stat.Total) * 100
		fmt.Printf("  %s:\n", endpoint)
		fmt.Printf("    Всего: %d, Успешно: %d, Ошибок: %d (%.2f%% успешных)\n",
			stat.Total, stat.Success, stat.Failed, successRate)
		sortedStatCodes := make([]int, 0, len(stat.StatusCodes))
		for code := range stat.StatusCodes {
			sortedStatCodes = append(sortedStatCodes, code)
		}
		sort.Ints(sortedStatCodes)
		for _, code := range sortedStatCodes {
			fmt.Printf("      %d: %d\n", code, stat.StatusCodes[code])
		}
	}
	fmt.Println()

	fmt.Println("=== ОЦЕНКА ПРОИЗВОДИТЕЛЬНОСТИ ===\n")

	// SLI для времени ответа (цель: < 300ms для P95)
	if result.P95ResponseTime < 300*time.Millisecond {
		fmt.Printf("✓ SLI Время ответа (P95 < 300ms): ПРОЙДЕНО (%.2fms)\n", float64(result.P95ResponseTime.Nanoseconds())/1e6)
	} else {
		fmt.Printf("✗ SLI Время ответа (P95 < 300ms): НЕ ПРОЙДЕНО (%.2fms)\n", float64(result.P95ResponseTime.Nanoseconds())/1e6)
	}

	// SLI для успешности (цель: > 99.9%)
	if result.ErrorRate < 0.001 {
		fmt.Printf("✓ SLI Успешность (> 99.9%%): ПРОЙДЕНО (%.4f%% ошибок)\n", result.ErrorRate*100)
	} else {
		fmt.Printf("✗ SLI Успешность (> 99.9%%): НЕ ПРОЙДЕНО (%.4f%% ошибок)\n", result.ErrorRate*100)
	}

	if result.RequestsPerSecond >= float64(targetRPS)*0.8 {
		fmt.Printf("✓ Пропускная способность: ПРОЙДЕНО (%.2f RPS)\n", result.RequestsPerSecond)
	} else {
		fmt.Printf("✗ Пропускная способность: НЕ ПРОЙДЕНО (%.2f RPS, цель: %.2f RPS)\n", result.RequestsPerSecond, float64(targetRPS))
	}

	fmt.Println("\n=== ТЕСТ ЗАВЕРШЕН ===")
}
