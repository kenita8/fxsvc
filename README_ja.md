# fxsvc

fxsvc は、Uber Go の Fx フレームワークを用いて開発された Go アプリケーションを Windows サービスとして簡単に実行できるようにする軽量ライブラリです。

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## 目次

1. [はじめに](#はじめに)
2. [なぜ `fxsvc` を使うのか？](#なぜ-fxsvc-を使うのか)
3. [主な特徴](#主な特徴)
4. [インストール](#インストール)
5. [基本的な使い方](#基本的な使い方)
6. [Windows サービスへの登録と管理](#windows-サービスへの登録と管理)
7. [デバッグモードについて](#デバッグモードについて)
8. [ライセンス](#ライセンス)

## はじめに

`fxsvc` は、[Uber Go の `Fx`](https://github.com/uber-go/fx) フレームワークで構築された Go アプリケーションを Windows サービスとして簡単に実行できるようにするための軽量なライブラリです。

Go で開発されたアプリケーションを Windows 環境でバックグラウンドサービスとして安定的に運用したい場合、通常は Windows サービス API を直接扱う必要があり、複雑な実装が求められます。`fxsvc` は、`Fx` アプリケーションの持つ強力な機能（依存性注入、ライフサイクル管理など）を活かしながら、Windows サービスとしての実行に必要な処理を抽象化し、開発者の負担を軽減します。

## なぜ `fxsvc` を使うのか？

- **Windows サービス開発の簡略化:** 複雑な Windows サービス固有の実装を意識することなく、`Fx` アプリケーションを Windows サービスとして実行できます。
- **`Fx` のメリットを享受:** `Fx` フレームワークの提供する依存性注入やライフサイクル管理の仕組みをそのまま利用できます。
- **容易なデバッグ:** デバッグモードを備えており、開発中は通常のコンソールアプリケーションとして実行できるため、テストや問題の切り分けが容易です。
- **構造化ロギング:** [go-logr](https://github.com/go-logr/logr) をサポートしており、サービスの状態やエラーを構造化されたログとして出力できます。

## 主な特徴

- `Fx` アプリケーションをラップするだけで Windows サービス化が可能
- 開発に便利なデバッグモードを搭載
- [go-logr](https://github.com/go-logr/logr) による構造化ログ出力
- 正常なシャットダウンのためのシグナルハンドリング

## インストール

以下のコマンドで `fxsvc` ライブラリをインストールします。

```bash
go get github.com/kenita8/fxsvc
```

## 基本的な使い方

`fxsvc` を使用するには、まず `Fx` でアプリケーションを構築し、その `fx.App` オブジェクトを `fxsvc.NewFxService` に渡して `FxService` インスタンスを作成します。その後、`Run` メソッドを実行することで、アプリケーションが Windows サービスとして動作します。

以下は基本的な使用例です。詳細は、リポジトリの `examples/main.go` を参照してください。

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

**重要なポイント:**

- `fx.New` で作成した `fx.App` オブジェクトを `fxsvc.NewFxService` に渡します。
- 2 番目の引数には、Windows サービスの名前を指定します。
- 3 番目の引数には、ロガーのインスタンス（`go-logr` の `logr.Logger` インターフェースを満たすもの）を指定します。
- `svc.Run()` を呼び出すことで、アプリケーションが Windows サービスとして起動します。

## Windows サービスへの登録と管理

ビルドした実行ファイルを Windows サービスとして登録するには、管理者権限でコマンドプロンプトまたは PowerShell を開き、以下のコマンドを実行します。

```powershell
New-Service -Name "your-service-name" -BinaryPathName "C:\path\to\your-service-executable.exe" -Description "Your service description" -StartupType Automatic
```

- `"your-service-name"`: 登録する Windows サービスの名前です。
- `"C:\path\to\your-service-executable.exe"`: ビルドした実行ファイルのフルパスです。
- `"Your service description"`: サービスの説明です。
- `-StartupType Automatic`: サービスの起動タイプ（例: 自動起動）を設定します。

サービスを登録後、Windows のサービス管理ツール（`services.msc`）を開き、登録したサービス（上記の例では "your-service-name"）を見つけて、起動、停止、再起動などの操作を行うことができます。

## デバッグモードについて

開発中は、Windows サービスとして登録せずに、通常のコンソールアプリケーションとして `fxsvc` アプリケーションを実行できます。これにより、ログ出力を直接確認したり、デバッガーを使用したりすることが容易になります。

デバッグモードを有効にするには、`FxService` オブジェクトの `SetDebug(true)` メソッドを呼び出します。

```go
svc := fxsvc.NewFxService(app, "your-service-name", logger)
svc.SetDebug(true) // デバッグモードを有効にする
if err := svc.Run(); err != nil {
	logger.Error(err, "Failed to run service")
	os.Exit(1)
}
```

デバッグモードでは、サービスはフォアグラウンドで実行され、Ctrl+C などの割り込みシグナルで停止できます。

## ライセンス

本プロジェクトは Apache License 2.0 の下で提供されています。詳細については、[LICENSE](LICENSE) ファイルをご確認ください。
