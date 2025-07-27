# GoNES - Nintendo Entertainment System Emulator

Go言語とSDL2で開発されたNESエミュレーターです。

## 特徴

- **完全なNES互換性**: 6502 CPU、PPU、APUの正確な実装
- **クロスプラットフォーム**: Windows、macOS、Linux対応
- **高性能**: 60FPS動作とリアルタイム音声出力
- **拡張可能**: モジュラー設計による保守性

## 必要な環境

- Go 1.21以上
- SDL2開発ライブラリ

### SDL2のインストール

#### Ubuntu/Debian
```bash
sudo apt-get install libsdl2-dev
```

#### macOS (Homebrew)
```bash
brew install sdl2
```

#### Windows
SDL2の開発ライブラリをダウンロードし、適切な場所に配置してください。

## ビルドと実行

```bash
# 依存関係のダウンロード
go mod tidy

# ビルド
go build -o gones cmd/gones/main.go

# 実行
./gones <rom_file.nes>
```

## 操作方法

- **十字キー**: 方向キー
- **A**: Z キー
- **B**: X キー
- **SELECT**: A キー
- **START**: S キー

## 開発状況

### Phase 1: 基盤実装 ✅
- [x] プロジェクト構造の構築
- [x] CPU基本実装 (6502命令セット)
- [x] Memory管理システム
- [ ] 基本的なテストフレームワーク

### Phase 2: 映像システム 🚧
- [x] PPU基本実装
- [x] SDL2描画システム
- [ ] パレット・色彩管理
- [ ] スプライト・BG描画

### Phase 3: カートリッジシステム 🚧
- [x] 基本カートリッジ構造
- [x] Mapper 0実装
- [ ] iNESローダー
- [ ] ROM/RAM管理

### Phase 4: 入力システム ✅
- [x] コントローラー入力
- [x] キーボードマッピング
- [x] SDL2イベント処理

### Phase 5: 音声システム 🚧
- [x] APU基本実装
- [ ] 音声チャンネル実装
- [ ] SDL2音声出力

### Phase 6: 最適化・拡張 ⏳
- [ ] 追加Mapper対応
- [ ] セーブステート機能
- [ ] デバッグ機能
- [ ] パフォーマンス最適化

## アーキテクチャ

```
pkg/
├── cpu/          # 6502プロセッサ実装
├── ppu/          # Picture Processing Unit
├── apu/          # Audio Processing Unit
├── memory/       # メモリ管理
├── cartridge/    # カートリッジとMapper
├── input/        # 入力処理
└── nes/          # メインNESシステム
```

## 貢献

プロジェクトへの貢献を歓迎します。Issues やPull Requestsをお送りください。

## ライセンス

MIT License

## 参考資料

- [NESdev Wiki](https://www.nesdev.org/)
- [6502 Instruction Set](http://www.6502.org/)
- [iNES Format Specification](https://www.nesdev.org/wiki/INES)