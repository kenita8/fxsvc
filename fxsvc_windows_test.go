// Copyright 2025 kenita8
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fxsvc

import (
	"context"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/kenita8/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/svc"
)

// mockAppRunner is a mock implementation of appRunner for testing.
type mockAppRunner struct {
	runFunc          func()
	startFunc        func(ctx context.Context) error
	stopFunc         func(ctx context.Context) error
	startTimeoutFunc func() time.Duration
	stopTimeoutFunc  func() time.Duration
	runCalled        bool
	startCalled      bool
	stopCalled       bool
	startCalledMutex sync.Mutex
	stopCalledMutex  sync.Mutex
	runCalledMutex   sync.Mutex
}

func (m *mockAppRunner) Start(ctx context.Context) error {
	m.startCalledMutex.Lock()
	m.startCalled = true
	m.startCalledMutex.Unlock()
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	return nil
}

func (m *mockAppRunner) Stop(ctx context.Context) error {
	m.stopCalledMutex.Lock()
	m.stopCalled = true
	m.stopCalledMutex.Unlock()
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	return nil
}

func (m *mockAppRunner) Run() {
	m.runCalledMutex.Lock()
	m.runCalled = true
	m.runCalledMutex.Unlock()
	if m.runFunc != nil {
		m.runFunc()
	}
}

func (m *mockAppRunner) StartTimeout() time.Duration {
	if m.startTimeoutFunc != nil {
		return m.startTimeoutFunc()
	}
	return time.Second
}

func (m *mockAppRunner) StopTimeout() time.Duration {
	if m.stopTimeoutFunc != nil {
		return m.stopTimeoutFunc()
	}
	return time.Second
}

// mockSvcRunner is a mock implementation of svcRunner for testing.
type mockSvcRunner struct {
	runFunc func(name string, handler svc.Handler) error
}

func (m *mockSvcRunner) Run(name string, handler svc.Handler) error {
	if m.runFunc != nil {
		return m.runFunc(name, handler)
	}
	return nil
}

func TestFxServiceWin_Run(t *testing.T) {
	zapLogger, _ := zap.NewProduction()
	logger := zapr.NewLogger(zapLogger)
	err := errors.New("error occurred")
	expectChanges := []svc.State{
		svc.StartPending,
		svc.Running,
		svc.Running,
		svc.Running,
		svc.Running,
		svc.StopPending,
	}

	testcase := []struct {
		name             string
		runSleep         time.Duration
		runErr           error
		errBeforeExecute bool
		startSleep       time.Duration
		startErr         error
		stopSleep        time.Duration
		stopErr          error
		changes          []svc.State
	}{
		{
			name:     "success1",
			runErr:   nil,
			startErr: nil,
			stopErr:  nil,
			changes:  expectChanges,
		},
		{
			name:     "success2",
			runSleep: time.Second,
			runErr:   nil,
			startErr: nil,
			stopErr:  nil,
			changes:  expectChanges,
		},
		{
			name:       "success3",
			runErr:     nil,
			startSleep: time.Second,
			startErr:   nil,
			stopErr:    nil,
			changes:    expectChanges,
		},
		{
			name:      "success4",
			runErr:    nil,
			startErr:  nil,
			stopSleep: time.Second,
			stopErr:   nil,
			changes:   expectChanges,
		},
		{
			name:             "run_error_before",
			runErr:           err,
			errBeforeExecute: true,
			startErr:         nil,
			stopErr:          nil,
			changes:          []svc.State{},
		},
		{
			name:             "run_error_after",
			runErr:           err,
			errBeforeExecute: false,
			startErr:         nil,
			stopErr:          nil,
			changes:          expectChanges,
		},
		{
			name:     "start_error",
			runErr:   nil,
			startErr: err,
			stopErr:  nil,
			changes:  expectChanges,
		},
		{
			name:     "stop_error",
			runErr:   nil,
			startErr: nil,
			stopErr:  err,
			changes:  expectChanges,
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			mockApp := &mockAppRunner{
				startFunc: func(ctx context.Context) error {
					time.Sleep(tc.startSleep)
					return tc.startErr
				},
				stopFunc: func(ctx context.Context) error {
					time.Sleep(tc.stopSleep)
					return tc.stopErr
				},
				startTimeoutFunc: func() time.Duration {
					return time.Second
				},
				stopTimeoutFunc: func() time.Duration {
					return time.Second
				},
			}
			req := make(chan svc.ChangeRequest, 100)
			chg := make(chan svc.Status, 100)
			mockSvc := &mockSvcRunner{
				runFunc: func(name string, s svc.Handler) error {
					if tc.errBeforeExecute == true && tc.runErr != nil {
						return tc.runErr
					}
					time.Sleep(tc.runSleep)
					req <- svc.ChangeRequest{Cmd: svc.Interrogate, CurrentStatus: svc.Status{State: svc.Running}}
					req <- svc.ChangeRequest{Cmd: svc.Interrogate, CurrentStatus: svc.Status{State: svc.Running}}
					req <- svc.ChangeRequest{Cmd: svc.Interrogate, CurrentStatus: svc.Status{State: svc.Running}}
					req <- svc.ChangeRequest{Cmd: 999, CurrentStatus: svc.Status{State: svc.Running}}
					req <- svc.ChangeRequest{Cmd: svc.Stop}
					s.Execute(nil, req, chg)
					return tc.runErr
				},
			}
			s := newFxService(mockApp, "testService", mockSvc, logger, defaultSignalHandler)
			s.Run()
			close(req)
			close(chg)
			changes := []svc.State{}
			for val := range chg {
				changes = append(changes, val.State)
			}
			assert.Equal(t, tc.changes, changes)
		})
	}
}

func TestFxServiceWin_Run_debug(t *testing.T) {
	zapLogger, _ := zap.NewProduction()
	logger := zapr.NewLogger(zapLogger)
	err := errors.New("error occurred")

	testcase := []struct {
		name             string
		runErr           error
		errBeforeExecute bool
		startErr         error
		stopErr          error
	}{
		{
			name:     "success",
			startErr: nil,
			stopErr:  nil,
		},
		{
			name:     "start_error",
			startErr: err,
			stopErr:  nil,
		},
		{
			name:     "stop_error",
			startErr: nil,
			stopErr:  err,
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			mockApp := &mockAppRunner{
				startFunc: func(ctx context.Context) error {
					return tc.startErr
				},
				stopFunc: func(ctx context.Context) error {
					return tc.stopErr
				},
				startTimeoutFunc: func() time.Duration {
					return time.Second
				},
				stopTimeoutFunc: func() time.Duration {
					return time.Second
				},
			}
			s := newFxService(mockApp, "testService", nil, logger, defaultSignalHandler).(*FxServiceWin)
			s.SetDebug(true)
			s.osSignalChan <- syscall.SIGINT
			s.Run()
		})
	}
}
