package accel

import (
	"bytes"
	"errors"
	"log"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func captureAccelLogs(t *testing.T, level insyra.LogLevel) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	previousPrefix := log.Prefix()
	previousLevel := insyra.Config.GetLogLevel()
	log.SetOutput(&output)
	log.SetFlags(0)
	log.SetPrefix("")
	insyra.Config.SetLogLevel(level)
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		log.SetPrefix(previousPrefix)
		insyra.Config.SetLogLevel(previousLevel)
	})
	return &output
}

func loggingFixture(t *testing.T) (*Session, *Dataset, [][]float64) {
	t.Helper()
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &shortlistExecutor{}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(91))
	return session, exactDataset(1024, 3, rnd), exactQueries(40, 3, rnd)
}

func TestExecutionLoggingIsOncePerSessionUnderConcurrentUse(t *testing.T) {
	output := captureAccelLogs(t, insyra.LogLevelDebug)
	session, dataset, queries := loggingFixture(t)

	const executions = 5
	var workers sync.WaitGroup
	workers.Add(executions)
	for i := 0; i < executions; i++ {
		go func() {
			defer workers.Done()
			result, err := session.ExecuteNearestExact(dataset, queries, 2, WorkloadEstimate{})
			if err != nil || !result.Accelerated {
				t.Errorf("execution failed: accelerated=%t err=%v", result.Accelerated, err)
			}
		}()
	}
	workers.Wait()

	logs := output.String()
	if got := strings.Count(logs, "acceleration engaged"); got != 1 {
		t.Fatalf("accelerated info lines = %d, want 1; logs:\n%s", got, logs)
	}
	if !strings.Contains(logs, "device=Stub CUDA Device") ||
		!strings.Contains(logs, "backend=cuda") ||
		!strings.Contains(logs, "mode=auto") ||
		!strings.Contains(logs, "shard_strategy=auto") {
		t.Fatalf("accelerated info line lacks session details:\n%s", logs)
	}
	if got := strings.Count(logs, "Execution operation=nearest-shortlist"); got != executions {
		t.Fatalf("debug execution lines = %d, want %d; logs:\n%s", got, executions, logs)
	}
	if !strings.Contains(logs, "rows=1024") || !strings.Contains(logs, "chunks=1") || !strings.Contains(logs, "placement=device=Stub CUDA Device") {
		t.Fatalf("debug detail lacks rows, chunks, or placement:\n%s", logs)
	}
}

func TestExecutionLoggingReportsFirstQualifyingFallbackOnce(t *testing.T) {
	output := captureAccelLogs(t, insyra.LogLevelInfo)
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &failingExecutor{err: errors.New("device failed")}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(92))
	dataset := exactDataset(1024, 3, rnd)
	queries := exactQueries(40, 3, rnd)
	for i := 0; i < 2; i++ {
		if result, err := session.ExecuteNearestExact(dataset, queries, 2, WorkloadEstimate{}); err != nil || result.FallbackReason != FallbackReasonExecutionFailed {
			t.Fatalf("fallback execution %d: result=%+v err=%v", i, result, err)
		}
	}

	logs := output.String()
	if got := strings.Count(logs, "acceleration fallback"); got != 1 {
		t.Fatalf("fallback info lines = %d, want 1; logs:\n%s", got, logs)
	}
	if !strings.Contains(logs, "reason=execution-failed") {
		t.Fatalf("fallback reason missing from info line:\n%s", logs)
	}
}

func TestExecutionLoggingKeepsCallerIneligibleFallbackAtDebug(t *testing.T) {
	output := captureAccelLogs(t, insyra.LogLevelDebug)
	session := singleDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(93))
	dataset := exactDataset(64, 3, rnd)
	queries := exactQueries(40, 3, rnd)
	result, err := session.ExecuteNearestExact(dataset, queries, 2, WorkloadEstimate{})
	if err != nil || result.FallbackReason != FallbackReasonWorkloadNotProfitable {
		t.Fatalf("ineligible execution: result=%+v err=%v", result, err)
	}

	logs := output.String()
	if strings.Contains(logs, "acceleration fallback") {
		t.Fatalf("caller-ineligible fallback was promoted to info:\n%s", logs)
	}
	if !strings.Contains(logs, "fallback_reason=workload-not-profitable") {
		t.Fatalf("debug fallback reason missing:\n%s", logs)
	}
}

func TestExecutionLoggingSilencesInfoWithRootLogLevel(t *testing.T) {
	output := captureAccelLogs(t, insyra.LogLevelWarning)
	session, dataset, queries := loggingFixture(t)
	result, err := session.ExecuteNearestExact(dataset, queries, 2, WorkloadEstimate{})
	if err != nil || !result.Accelerated {
		t.Fatalf("execution failed: accelerated=%t err=%v", result.Accelerated, err)
	}
	if output.Len() != 0 {
		t.Fatalf("silenced info level produced log output:\n%s", output.String())
	}
}
