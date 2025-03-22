# fxsvc

fxsvc is a lightweight library designed to simplify running Go applications developed with Uber Go's Fx framework as Windows services.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Table of Contents

1. [Introduction](#introduction)
2. [Why use `fxsvc`?](#why-use-fxsvc)
3. [Main Features](#main-features)
4. [Installation](#installation)
5. [Basic Usage](#basic-usage)
6. [Registering and Managing as a Windows Service](#registering-and-managing-as-a-windows-service)
7. [About Debug Mode](#about-debug-mode)
8. [License](#license)

## Introduction

`fxsvc` is a lightweight library that makes it easy to run Go applications built with the [Uber Go's `Fx`](<[https://github.com/uber-go/fx](https://github.com/uber-go/fx)>) framework as Windows services.

When you want to stably operate Go applications as background services in a Windows environment, you typically need to interact directly with the Windows Service API, which requires complex implementations for service lifecycle, error handling, concurrency, and more.

`fxsvc` solves these problems by integrating the lifecycle of your `Fx` application with the lifecycle of a Windows service. You can benefit from `Fx`'s powerful dependency injection and lifecycle management while significantly simplifying the deployment and operation as a Windows service.

## Why use `fxsvc`?

- **Simplifies Windows service development:** You can run your `Fx` application as a Windows service without being aware of complex Windows service-specific implementations.
- **Leverage the benefits of `Fx`:** You can continue to use the dependency injection and lifecycle management mechanisms provided by the `Fx` framework.
- **Easy debugging:** Equipped with a debug mode, it can be run as a normal console application during development, making testing and issue isolation easy.
- **Structured logging:** Supports [go-logr](https://github.com/go-logr/logr), allowing you to output service status and errors as structured logs.

## Main Features

- Enables Windows service functionality simply by wrapping your `Fx` application
- Equipped with a debug mode convenient for development
- Structured log output using [go-logr](https://github.com/go-logr/logr)
- Signal handling for graceful shutdown

## Installation

Install the `fxsvc` library with the following command:

```bash
go get github.com/kenita8/fxsvc
```

## Basic Usage

To use `fxsvc`, first build your application with `Fx`, then pass its `fx.App` object to `fxsvc.NewFxService` to create an `FxService` instance. After that, by executing the `Run` method, your application will operate as a Windows service.

Below is a basic usage example. For more details, please refer to `examples/main.go` in the repository.

```go
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

	svc := fxsvc.NewFxService(app, "your-service-name", logger)
	svc.Run()
}
```

**Important Points:**

- Pass the `fx.App` object created with `fx.New` to `fxsvc.NewFxService`.
- Specify the name of the Windows service as the second argument.
- Specify an instance of the logger (one that satisfies the `logr.Logger` interface of `go-logr`) as the third argument.
- By calling `svc.Run()`, your application will start as a Windows service.

## Registering and Managing as a Windows Service

To register the built executable as a Windows service, open Command Prompt or PowerShell with administrator privileges and execute the following command:

```powershell
New-Service -Name "your-service-name" -BinaryPathName "C:\path\to\your-service-executable.exe" -Description "Your service description" -StartupType Automatic
```

- `"your-service-name"`: The name of the Windows service to be registered.
- `"C:\path\to\your-service-executable.exe"`: The full path to your built executable file.
- `"Your service description"`: The description of the service.
- `-StartupType Automatic`: Sets the service's startup type (e.g., automatic startup).

After registering the service, open the Windows Services management tool (`services.msc`), find the registered service (in the example above, "your-service-name"), and you can perform operations such as starting, stopping, and restarting it.

## About Debug Mode

During development, you can run your `fxsvc` application as a normal console application without registering it as a Windows service. This makes it easy to directly check log output and use a debugger.

To enable debug mode, call the `SetDebug(true)` method of the `FxService` object.

```go
svc := fxsvc.NewFxService(app, "your-service-name", logger)
svc.SetDebug(true) // Enable debug mode
if err := svc.Run(); err != nil {
	logger.Error(err, "Failed to run service")
	os.Exit(1)
}
```

In debug mode, the service runs in the foreground and can be stopped with interrupt signals such as Ctrl+C.

## License

This project is provided under the Apache License 2.0. For details, please see the [LICENSE](LICENSE) file.
