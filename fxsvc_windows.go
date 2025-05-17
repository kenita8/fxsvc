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
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/windows/svc"
)

type svcRunner interface {
	Run(name string, handler svc.Handler) error
}

type DefaultSvcRunner struct{}

func (d *DefaultSvcRunner) Run(name string, handler svc.Handler) error {
	return svc.Run(name, handler)
}

type eventChannel struct {
	ch   chan struct{}
	once sync.Once
}

func newEventChannel() *eventChannel {
	return &eventChannel{
		ch: make(chan struct{}),
	}
}

func (e *eventChannel) wait() {
	<-e.ch
}

func (e *eventChannel) signal() {
	e.once.Do(func() {
		close(e.ch)
	})
}

type serviceProgress struct {
	startPending *eventChannel
	running      *eventChannel
	stopPending  *eventChannel
}

func newServiceProgress() *serviceProgress {
	return &serviceProgress{
		startPending: newEventChannel(),
		running:      newEventChannel(),
		stopPending:  newEventChannel(),
	}
}

func (s *serviceProgress) waitForStartPending() {
	s.startPending.wait()
}

func (s *serviceProgress) waitForStopInitiation() {
	s.running.wait()
}

func (s *serviceProgress) waitForStopPending() {
	s.stopPending.wait()
}

func (s *serviceProgress) signalStartPending() {
	s.startPending.signal()
}

func (s *serviceProgress) signalStopInitiation() {
	s.running.signal()
}

func (s *serviceProgress) signalStopPending() {
	s.stopPending.signal()
}

func (s *serviceProgress) signalAll() {
	s.signalStartPending()
	s.signalStopInitiation()
	s.signalStopPending()
}

type FxServiceWin struct {
	app           appRunner
	name          string
	isDebug       bool
	logger        logr.Logger
	svc           svcRunner
	progress      *serviceProgress
	eg            *errgroup.Group
	signalHandler func(chan os.Signal)
	osSignalChan  chan os.Signal
}

func NewFxService(app appRunner, name string, logger logr.Logger) FxService {
	return newFxService(app, name, &DefaultSvcRunner{}, logger, defaultSignalHandler)
}

func newFxService(app appRunner, name string, svc svcRunner, logger logr.Logger, signalHandler func(chan os.Signal)) FxService {
	return &FxServiceWin{
		app:           app,
		name:          name,
		logger:        logger.WithName("fxsvc"),
		svc:           svc,
		progress:      newServiceProgress(),
		eg:            &errgroup.Group{},
		signalHandler: signalHandler,
		osSignalChan:  make(chan os.Signal, 1),
	}
}

func (s *FxServiceWin) start(ctx context.Context) error {
	s.logger.Info("Starting service", "name", s.name)
	err := s.app.Start(ctx)
	if err != nil {
		s.progress.signalAll()
		s.logger.Error(err, "Failed to start application")
		return ErrStartApplication.WithDetails("err", err, "service", s.name)
	}
	s.progress.signalStartPending()
	s.logger.Info("Service started")
	return nil
}

func (s *FxServiceWin) stop(ctx context.Context) error {
	s.logger.Info("Stopping service", "name", s.name)

	err := s.app.Stop(ctx)
	if err != nil {
		s.progress.signalAll()
		s.logger.Error(err, "Failed to stop application")
		return ErrStopApplication.WithDetails("err", err, "service", s.name)
	}

	s.progress.signalStopPending()
	s.logger.Info("Service stopped")
	return nil
}

func (s *FxServiceWin) Run() error {
	if s.isDebug {
		s.eg.Go(func() error {
			s.debug()
			return nil
		})
	} else {
		sigs := make(chan os.Signal, 1)
		s.signalHandler(sigs)
		go func() {
			for {
				<-sigs
			}
		}()

		s.eg.Go(func() error {
			err := s.svc.Run(s.name, s)
			if err != nil {
				s.progress.signalAll()
				s.logger.Error(err, "Failed to run service")
				return nil
			}
			return nil
		})
	}
	defer s.eg.Wait()

	startCtx, cancelStartCtx := context.WithTimeout(context.Background(), s.app.StartTimeout())
	defer cancelStartCtx()
	err := s.start(startCtx)
	if err != nil {
		return err
	}

	s.progress.waitForStopInitiation()

	stopCtx, cancelStopCtx := context.WithTimeout(context.Background(), s.app.StopTimeout())
	defer cancelStopCtx()
	err = s.stop(stopCtx)
	if err != nil {
		return err
	}
	return nil
}

func (s *FxServiceWin) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	s.progress.waitForStartPending()
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
loop:
	for {
		select {
		case c := <-requests:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			default:
				s.logger.Info("Unexpected control request", "cmd", c.Cmd)
			}
		case <-s.progress.running.ch:
			break loop
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	s.progress.signalStopInitiation()
	s.progress.waitForStopPending()
	return false, 0
}

func (s *FxServiceWin) debug() {
	s.logger.Info("Running in debug mode")
	s.progress.waitForStartPending()
	s.signalHandler(s.osSignalChan)
	select {
	case <-s.osSignalChan:
		break
	case <-s.progress.running.ch:
		break
	}
	s.progress.signalStopInitiation()
	s.progress.waitForStopPending()
}

func (s *FxServiceWin) SetDebug(isDebug bool) {
	s.isDebug = isDebug
}

func defaultSignalHandler(sigCh chan os.Signal) {
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
}
