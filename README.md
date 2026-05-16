# GoNES - Nintendo Entertainment System Emulator

Go言語とSDL2で開発されたNESエミュレーターです。

## 特徴

- **6502 CPU**: 公式命令セット + 主要な非公式命令、サイクル精度のタイミング
- **PPU**: スキャンライン単位のレンダリング、スプライト0ヒット、スクロール、MMC3 IRQ対応
- **APU**: 5チャンネル（矩形波x2, 三角波, ノイズ, DMC）+ アナログ風フィルタチェーン
- **Mapper**: 0 (NROM), 1 (MMC1), 2 (UxROM), 3 (CNROM), 4 (MMC3), 10 (MMC4)
- **入力**: キーボード + ゲームパッド/ジョイスティック（ホットプラグ対応、最大2P）
- **セーブ機能**: バッテリーバックアップ（.sav 自動保存）、セーブステート（10スロット）
- **その他**: Game Genieチートコード、WAV録音、スクリーンショット、ターボ（早送り）
- **クロスプラットフォーム**: Windows、macOS、Linux対応

## 必要な環境

- Go 1.21以上
- SDL2開発ライブラリ

### SDL2のインストール

#### Ubuntu/Debian
```bash
sudo apt-get install libsdl2-dev
```

#### Fedora/CentOS
```bash
sudo dnf install SDL2-devel
```

#### macOS (Homebrew)
```bash
brew install sdl2
```

#### Windows
go-sdl2パッケージにバンドルされたSDL2ライブラリが使用されます。クロスコンパイル手順は [README-BUILD.md](README-BUILD.md) を参照。

## ビルドと実行

```bash
# 依存関係のダウンロード
go mod tidy

# ビルド（現在のプラットフォーム向け）
go build -o gones ./cmd/gones

# 実行
./gones <rom_file.nes>
```

Makefileを使う場合は `make build` / `make build-linux` / `make build-windows` / `make build-all` などが利用可能です。

### コマンドラインオプション

```
Usage: gones [options] <rom_file>

  -log-level string    ログレベル (off, error, warn, info, debug, trace) (default "info")
  -log-file string     ログ出力先ファイル（空ならstdout）
  -cpu-log             CPU命令ログを有効化
  -ppu-log             PPUログを有効化
  -apu-log             APUログを有効化
  -mapper-log          Mapperログを有効化
  -headless            ヘッドレスモード（GUIなし、テスト用）
  -test-frames int     ヘッドレスモードで実行するフレーム数 (default 600)
  -debug               追加のデバッグ出力を有効化
```

## 操作方法

### ゲーム入力（キーボード）

| キー | NESボタン |
|------|-----------|
| Z | A |
| X | B |
| A | SELECT |
| S | START |
| ↑↓←→ | 十字キー |

ゲームパッドもSDL2のGameController API経由で自動認識されます（A/B/X/Y → A/B、Back → SELECT、Start → START、D-pad・左スティック → 方向）。

### エミュレータホットキー

| キー | 動作 |
|------|------|
| Tab | ターボ（早送り）トグル |
| Ctrl+R | NESリセット |
| Ctrl+H | チートコード全体のON/OFF |
| Ctrl+E | WAV録音の開始/停止 |
| F1〜F10 | ステートをスロット1〜10へ保存 |
| Ctrl+F1〜F10 | スロット1〜10からステートをロード |
| F11 | FPS表示トグル |
| F12 | スクリーンショット保存 |
| 1〜5 | APUチャンネル1〜5をミュート切替（矩形1, 矩形2, 三角, ノイズ, DMC） |
| 6 | APUアナログフィルタチェーンのON/OFF |
| ESC | 終了 |

### コンパニオンファイル

ROMと同じディレクトリに次のファイルが自動的に読み書きされます：

- `<rom>.sav` — バッテリーバックアップRAM（対応カートリッジのみ、終了時に自動保存）
- `<rom>.state1` 〜 `<rom>.state10` — セーブステート
- `<rom>.cht` — Game Genieチートコード（起動時に読み込み）
- `<rom>.<timestamp>.wav` — Ctrl+Eで録音した音声

## 重要な注意事項

### リージョン対応

本エミュレーターは現在 **NTSC（北米/日本）仕様** で動作します。CPU周波数1.789773 MHz、60.0988 FPSです。PAL版ROMを実行するとタイミングが約7-8%速くなります。

**今後の対応予定**:
- [ ] ROMヘッダーからのPAL/NTSC自動検出
- [ ] PAL仕様のタイミング実装（1.662607 MHz CPU、50 FPS）

## アーキテクチャ

```
cmd/
├── gones/             # メインエミュレータ
├── headless_debug/    # ヘッドレスデバッグツール
└── rom_analyzer/      # ROM解析ツール

pkg/
├── cpu/               # 6502 CPU
├── ppu/               # Picture Processing Unit
├── apu/               # Audio Processing Unit (チャンネル, フィルタ)
├── memory/            # CPU/PPUメモリマップ
├── cartridge/         # iNESローダ
│   └── mapper/        # Mapper 0/1/2/3/4/10
├── input/             # NESコントローラ抽象
├── cheat/             # Game Genie パーサ・マネージャ
├── logger/            # 構造化ログ
├── gui/               # SDL2 GUI（描画・音声・入力・録音）
└── nes/               # 全体統合、Step/StepFrame、セーブステート

test/                  # 統合テスト
tools/wavstat/         # WAV統計ツール
```

## テスト

```bash
go test ./...
```

CPU、PPU、APU、Mapper（0〜4）、カートリッジ、セーブステート、統合テストが含まれています。

## ライセンス

MIT License

## 参考資料

- [NESdev Wiki](https://www.nesdev.org/)
- [6502 Instruction Set](http://www.6502.org/)
- [iNES Format Specification](https://www.nesdev.org/wiki/INES)
