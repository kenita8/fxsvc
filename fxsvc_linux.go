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
	"github.com/go-logr/logr"
	"go.uber.org/fx"
)

type FxServiceLin struct {
	app    appRunner
	name   string
	logger logr.Logger
}

func NewFxService(app *fx.App, name string, logger logr.Logger) FxService {
	return &FxServiceLin{
		app:    app,
		name:   name,
		logger: logger.WithName("fxsvc"),
	}
}

func (s *FxServiceLin) Run() error {
	s.app.Run()
	return nil
}

func (s *FxServiceLin) SetDebug(isDebug bool) {
}
