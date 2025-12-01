package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL     = "http://localhost:8080"
	numRequests = 500
	duration    = 10 * time.Second
)

type TestResult struct {
	TotalRequests   int
	SuccessRequests int
	FailedRequests  int
	AvgResponseTime time.Duration
	MaxResponseTime time.Duration
}

// хранилище созданных PR ID для последующего merge
var (
	createdPRs   []string
	prMutex      sync.Mutex
	mergeCounter int
)

func main() {
	fmt.Println("=== НАГРУЗОЧНОЕ ТЕСТИРОВАНИЕ ===")
	fmt.Printf("URL: %s\n", baseURL)
	fmt.Printf("Запросов: %d\n", numRequests)
	fmt.Printf("Длительность: %v\n\n", duration)

	setupTestData()
	result := runLoadTest()
	printResults(result)
}

func setupTestData() {
	fmt.Println("Создание тестовых данных...")
	client := &http.Client{Timeout: 5 * time.Second}

	teams := []string{"backend", "frontend", "devops"}
	for _, team := range teams {
		createTeam(client, team)
		time.Sleep(100 * time.Millisecond)
	}

	for _, team := range teams {
		for i := 0; i < 100; i++ {
			createPR(client, team, i)
		}
	}

	fmt.Println("Данные созданы")
}

func createTeam(client *http.Client, teamName string) {
	req := map[string]interface{}{
		"teamName": teamName,
		"members": []map[string]interface{}{
			{
				"userId":   fmt.Sprintf("user_%s_1", teamName),
				"userName": fmt.Sprintf("User1_%s", teamName),
				"teamName": teamName,
				"isActive": true,
			},
			{
				"userId":   fmt.Sprintf("user_%s_2", teamName),
				"userName": fmt.Sprintf("User2_%s", teamName),
				"teamName": teamName,
				"isActive": true,
			},
		},
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", baseURL+"/team/add", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err == nil && resp != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// createPR создаёт Pull Request с уникальным ID и сохраняет его для последующего merge
func createPR(client *http.Client, teamName string, idx int) {
	prID := fmt.Sprintf("pr_%s_%d_%d", teamName, idx, time.Now().UnixNano()/1000000)

	req := map[string]interface{}{
		"pullRequestId":   prID,
		"pullRequestName": fmt.Sprintf("PR %d for %s", idx, teamName),
		"authorId":        fmt.Sprintf("user_%s_1", teamName),
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", baseURL+"/pullRequest/create", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err == nil && resp != nil {
		if resp.StatusCode == 200 {
			prMutex.Lock()
			createdPRs = append(createdPRs, prID)
			prMutex.Unlock()
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

func runLoadTest() TestResult {
	client := &http.Client{Timeout: 10 * time.Second}
	result := TestResult{}

	start := time.Now()
	responseTimes := make([]time.Duration, 0, numRequests)

	endpoints := []func(*http.Client) (bool, time.Duration){
		testTeamGet,
		testUserReview,
		testPRMerge,
		testStatistics,
		testUserSetActive,
	}

	requestCount := 0
	for time.Since(start) < duration && requestCount < numRequests {
		endpoint := endpoints[requestCount%len(endpoints)]
		success, responseTime := endpoint(client)

		result.TotalRequests++
		responseTimes = append(responseTimes, responseTime)

		if success {
			result.SuccessRequests++
		} else {
			result.FailedRequests++
		}

		if responseTime > result.MaxResponseTime {
			result.MaxResponseTime = responseTime
		}

		requestCount++
		time.Sleep(20 * time.Millisecond)
	}

	var totalTime int64
	for _, rt := range responseTimes {
		totalTime += rt.Nanoseconds()
	}
	if len(responseTimes) > 0 {
		result.AvgResponseTime = time.Duration(totalTime / int64(len(responseTimes)))
	}

	return result
}

// isBusinessError проверяет, является ли код бизнесовой ошибкой (не техническая проблема)
func isBusinessError(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500
}

func testTeamGet(client *http.Client) (bool, time.Duration) {
	start := time.Now()
	resp, err := client.Get(baseURL + "/team/get?team_name=backend")
	duration := time.Since(start)

	if err != nil {
		return false, duration
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// бизнес-ошибки не считаем техническими
	return resp.StatusCode >= 200 && resp.StatusCode < 300 || isBusinessError(resp.StatusCode), duration
}

func testUserReview(client *http.Client) (bool, time.Duration) {
	start := time.Now()
	resp, err := client.Get(baseURL + "/users/getReview?userId=user_backend_1")
	duration := time.Since(start)

	if err != nil {
		return false, duration
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// бизнес-ошибки не считаем техническими
	return resp.StatusCode >= 200 && resp.StatusCode < 300 || isBusinessError(resp.StatusCode), duration
}

// testPRMerge мержит PR из списка созданных
func testPRMerge(client *http.Client) (bool, time.Duration) {
	prMutex.Lock()
	if mergeCounter >= len(createdPRs) {
		prMutex.Unlock()
		return true, 0
	}
	prID := createdPRs[mergeCounter]
	mergeCounter++
	prMutex.Unlock()

	req := map[string]interface{}{
		"pullRequestId": prID,
	}
	body, _ := json.Marshal(req)

	start := time.Now()
	httpReq, _ := http.NewRequest("POST", baseURL+"/pullRequest/merge", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return false, duration
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// бизнес-ошибки не считаем техническими
	return resp.StatusCode >= 200 && resp.StatusCode < 300 || isBusinessError(resp.StatusCode), duration
}

func testStatistics(client *http.Client) (bool, time.Duration) {
	start := time.Now()
	resp, err := client.Get(baseURL + "/statistics?team_name=backend")
	duration := time.Since(start)

	if err != nil {
		return false, duration
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// бизнес-ошибки не считаем техническими
	return resp.StatusCode >= 200 && resp.StatusCode < 300 || isBusinessError(resp.StatusCode), duration
}

func testUserSetActive(client *http.Client) (bool, time.Duration) {
	req := map[string]interface{}{
		"userId":   "user_frontend_1",
		"isActive": true,
	}
	body, _ := json.Marshal(req)

	start := time.Now()
	httpReq, _ := http.NewRequest("POST", baseURL+"/users/setIsActive", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	duration := time.Since(start)

	if err != nil {
		return false, duration
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// бизнес-ошибки не считаем техническими
	return resp.StatusCode >= 200 && resp.StatusCode < 300 || isBusinessError(resp.StatusCode), duration
}

func printResults(result TestResult) {
	fmt.Println("=== РЕЗУЛЬТАТЫ ===")
	fmt.Printf("Всего запросов: %d\n", result.TotalRequests)
	fmt.Printf("Успешных: %d\n", result.SuccessRequests)
	fmt.Printf("Неудачных: %d\n", result.FailedRequests)

	var successRate float64
	if result.TotalRequests > 0 {
		successRate = float64(result.SuccessRequests) / float64(result.TotalRequests)
	}

	fmt.Printf("Процент успеха: %.2f%%\n", successRate*100)
	fmt.Printf("Среднее время ответа: %v\n", result.AvgResponseTime)
	fmt.Printf("Максимальное время ответа: %v\n\n", result.MaxResponseTime)

	fmt.Println()
	switch {
	case result.AvgResponseTime < 100*time.Millisecond:
		fmt.Println("Производительность: excellent")
	case result.AvgResponseTime < 300*time.Millisecond:
		fmt.Println("Производительность: good")
	default:
		fmt.Println("Производительность: needs optimization")
	}

	if successRate >= 0.999 {
		fmt.Println("Надёжность: high")
	} else if successRate > 0.95 {
		fmt.Println("Надёжность: good")
	} else {
		fmt.Println("Надёжность: low")
	}
}
