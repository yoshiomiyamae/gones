# GoNES - NESエミュレーター仕様書

## 1. プロジェクト概要

### 1.1 プロジェクト名
GoNES (Go Nintendo Entertainment System Emulator)

### 1.2 目的
Go言語とSDL2を使用して、Nintendo Entertainment System (NES/ファミリーコンピューター) のエミュレーターを開発する。

### 1.3 対象システム
- Nintendo Entertainment System (NES)
- ファミリーコンピューター (FC)

## 2. 技術仕様

### 2.1 開発環境
- **言語**: Go 1.21以上
- **グラフィックライブラリ**: SDL2 (github.com/veandco/go-sdl2)
- **プラットフォーム**: Windows, macOS, Linux

### 2.2 NESハードウェア仕様
- **CPU**: MOS 6502 (1.79MHz)
- **PPU**: Picture Processing Unit (映像処理)
- **APU**: Audio Processing Unit (音声処理)
- **メモリ**: 2KB RAM + カートリッジ容量
- **解像度**: 256x240ピクセル
- **カラーパレット**: 64色中同時表示可能色数制限あり

## 3. アーキテクチャ設計

### 3.1 主要コンポーネント
```
GoNES
├── CPU (6502プロセッサ)
├── PPU (映像処理ユニット)
├── APU (音声処理ユニット)
├── Memory (メモリ管理)
├── Cartridge (カートリッジ処理)
├── Input (入力処理)
└── Renderer (SDL2描画)
```

### 3.2 モジュール詳細

#### 3.2.1 CPU モジュール
- **責務**: 6502命令セットの実装
- **機能**:
  - 命令デコード・実行
  - レジスタ管理 (A, X, Y, SP, PC, P)
  - アドレッシングモード処理
  - 割り込み処理 (NMI, IRQ, BRK)

#### 3.2.2 PPU モジュール
- **責務**: 映像出力処理
- **機能**:
  - スプライト描画
  - バックグラウンド描画
  - パレット管理
  - VRAM管理
  - スクロール処理

#### 3.2.3 APU モジュール
- **責務**: 音声出力処理
- **機能**:
  - 矩形波チャンネル (2ch)
  - 三角波チャンネル (1ch)
  - ノイズチャンネル (1ch)
  - DMCチャンネル (1ch)

#### 3.2.4 Memory モジュール
- **責務**: メモリマップ管理
- **機能**:
  - CPU RAM (2KB)
  - PPU RAM (2KB)
  - カートリッジメモリアクセス
  - メモリミラーリング処理

#### 3.2.5 Cartridge モジュール
- **責務**: ROMカートリッジ処理
- **機能**:
  - iNESフォーマット対応
  - Mapper実装 (0, 1, 2, 3...)
  - PRG-ROM/CHR-ROM管理
  - SRAM/Battery backup対応

#### 3.2.6 Input モジュール
- **責務**: コントローラー入力
- **機能**:
  - 標準コントローラー (十字キー + A/B/SELECT/START)
  - キーボードマッピング
  - SDL2イベント処理

#### 3.2.7 Renderer モジュール
- **責務**: SDL2による画面描画
- **機能**:
  - フレームバッファ管理
  - ピクセル描画
  - フルスクリーン/ウィンドウモード
  - フィルタリング対応

## 4. 実装フェーズ

### Phase 1: 基盤実装
- [ ] プロジェクト構造の構築
- [ ] CPU基本実装 (6502命令セット)
- [ ] Memory管理システム
- [ ] 基本的なテストフレームワーク

### Phase 2: 映像システム
- [ ] PPU基本実装
- [ ] SDL2描画システム
- [ ] パレット・色彩管理
- [ ] スプライト・BG描画

### Phase 3: カートリッジシステム
- [ ] iNESローダー
- [ ] Mapper 0実装
- [ ] ROM/RAM管理

### Phase 4: 入力システム
- [ ] コントローラー入力
- [ ] キーボードマッピング
- [ ] SDL2イベント処理

### Phase 5: 音声システム
- [ ] APU基本実装
- [ ] 音声チャンネル実装
- [ ] SDL2音声出力

### Phase 6: 最適化・拡張
- [ ] 追加Mapper対応
- [ ] セーブステート機能
- [ ] デバッグ機能
- [ ] パフォーマンス最適化

## 5. ファイル構造
```
gones/
├── cmd/
│   └── gones/
│       └── main.go
├── pkg/
│   ├── cpu/
│   │   ├── cpu.go
│   │   ├── instructions.go
│   │   └── addressing.go
│   ├── ppu/
│   │   ├── ppu.go
│   │   ├── renderer.go
│   │   └── palette.go
│   ├── apu/
│   │   └── apu.go
│   ├── memory/
│   │   └── memory.go
│   ├── cartridge/
│   │   ├── cartridge.go
│   │   └── mapper/
│   ├── input/
│   │   └── controller.go
│   └── nes/
│       └── nes.go
├── test/
│   ├── roms/
│   └── cpu_test.go
├── go.mod
├── go.sum
├── README.md
└── SPECIFICATION.md
```

## 6. 開発ガイドライン

### 6.1 コーディング規約
- Go標準のコーディング規約に従う
- gofmt, golint使用
- 適切なコメント記述 (英語)
- テスト駆動開発推奨

### 6.2 テスト方針
- ユニットテスト必須
- CPU命令テスト用ROM使用
- 統合テスト実装
- ベンチマークテスト

### 6.3 パフォーマンス目標
- 60FPS動作
- 低レイテンシ音声出力
- メモリ効率的な実装

## 7. 依存関係

### 7.1 外部ライブラリ
```go
require (
    github.com/veandco/go-sdl2 v0.4.35
)
```

### 7.2 開発ツール
- Go 1.21+
- SDL2開発ライブラリ
- Git
- テスト用NES ROM

## 8. 参考資料
- NESdev Wiki: https://www.nesdev.org/
- 6502命令セット: http://www.6502.org/
- iNESフォーマット仕様
- PPU/APU技術資料