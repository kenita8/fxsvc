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
package main

import (
	"context"
	"time"

	"github.com/kenita8/fxsvc"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ComponentA struct {
	logger logr.Logger
}

func NewComponentA(lc fx.Lifecycle, logger logr.Logger) *ComponentA {
	ca := &ComponentA{
		logger: logger.WithName("ComponentA"),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("ComponentA starting")
			time.Sleep(2 * time.Second)
			logger.Info("ComponentA started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("ComponentA stopping")
			time.Sleep(2 * time.Second)
			logger.Info("ComponentA stopped")
			return nil
		},
	})
	return ca
}

type ComponentB struct {
	logger logr.Logger
}

func NewComponentB(lc fx.Lifecycle, logger logr.Logger) *ComponentB {
	cb := &ComponentB{
		logger: logger.WithName("ComponentB"),
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("ComponentB starting")
			time.Sleep(2 * time.Second)
			logger.Info("ComponentB started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("ComponentB stopping")
			time.Sleep(2 * time.Second)
			logger.Info("ComponentB stopped")
			return nil
		},
	})
	return cb
}

func main() {
	zapLogger, _ := zap.NewProduction()
	logger := zapr.NewLogger(zapLogger)

	app := fx.New(
		fx.Provide(
			func() logr.Logger { return logger },
			NewComponentA,
			NewComponentB,
		),
		fx.Invoke(func(lc fx.Lifecycle, compA *ComponentA, compB *ComponentB) {}),
	)

	svc := fxsvc.NewFxService(app, "FxSvcExample", logger)
	svc.Run()
}
