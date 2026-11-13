# hccd

[English](README.md) | 日本語

Herdr のタブを「ディレクトリー単位」で開く／切り替えるツール。

```
hccd [options] [<dir>|<file>]
```

- 引数なし: カレントディレクトリー（エクスプローラーから起動したときは、そのエクスプローラーで表示中のフォルダー）
- `<dir>`: そのディレクトリー
- `<file>`: 親ディレクトリーのタブを開き、そのシェルでファイルを実行する

そのディレクトリーにいるシェルのタブがあればそれを表示し、無ければ新しいタブを作る。Herdr のウィンドウが無ければ起動し、最後に前面へ出す。

| オプション | 意味 |
|---|---|
| `-s`, `--space` | タブが無いとき、新しいスペースを作る |
| `-n`, `--new` | 既存タブを探さず、常に新しく作る |
| `--name <name>` | 新しいタブ／スペースの名前（既定: フォルダー名） |
| `-v`, `--verbose` | 何をしたかを表示 |
| `--version` | バージョンを表示 |
| `--register` / `--unregister` | (Windows) エクスプローラーの右クリックメニューに「Open in Herdr」を登録／削除（`-s` と併用可） |

## 必要なもの

- [herdr-core](https://github.com/herdrdev/herdr-core) の `hc` と [herdr-app](https://github.com/herdrdev/herdr-app) の `herdr-app` が PATH にあること

## ビルド

```powershell
go build -o bin\hccd.exe ./cmd/hccd                             # 端末用
go build -ldflags "-H=windowsgui" -o bin\hccdw.exe ./cmd/hccd   # エクスプローラー用（コンソール窓なし）
```

Linux / macOS は `go build -o bin/hccd ./cmd/hccd`。

## エクスプローラーから使う (Windows)

- `bin\hccd.exe --register` で右クリックメニューに登録する（`hccdw.exe` が同じフォルダーにあればそれを登録）。フォルダー、フォルダーの背景、ドライブ、ファイルのメニューに出る。Windows 11 では「その他のオプションを確認」の中。
- `hccdw.exe` をダブルクリック、またはそのショートカットから起動すると、前面のエクスプローラーで表示中のフォルダーを対象にする。
  前面がエクスプローラーでないとき（デスクトップのショートカットから起動した場合など）は、カレントディレクトリー（ショートカットの「作業フォルダー」、既定は exe のあるフォルダー）を対象にする。

## 制約

- ディレクトリーの判定は Herdr 既定のシェル（Windows では pwsh）の現在位置で行う。タブ内でさらに cmd などを起動して移動した場合は判定できない。
- 「空いているシェル」の判定は、エージェントがいないこととシェルに子プロセスが無いことで行う。pwsh 内部コマンド（`Start-Sleep` など）の実行中は空きと判定される。
- Linux / macOS は未検証。
