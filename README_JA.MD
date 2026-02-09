# ✨ Gemini Watermark Remover Pro (v1.0)

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Gemini Watermark Remover Pro** は、Google Gemini（Imagen 2/3）によって生成された画像の右下隅に表示される半透明のウォーターマークを除去するためのローカルツールです。

高性能な Go バックエンドとモダンな HTML5/JS フロントエンドを組み合わせ、高度な三段階の NCC 検出アルゴリズムを搭載しており、ウォーターマークを賢く検出・精密に位置特定し、画像のディテールをできるだけ損なわずに復元します。

---

## 🚀 主な機能

- **🎨 デュアルワークフロー**：
  - **インタラクティブモード**：自動検出が不正確な場合、赤枠をドラッグして手動で修正し、前後をリアルタイムに比較できます。
  - **バッチ処理**：複数画像を一括でドラッグして投入でき、キュー処理で効率的に処理します。
- **🧠 インテリジェント検出アルゴリズム（NCC）**：
  - 空間相関、勾配エッジマッチング、分散（バリアンス）解析の三段階の加重スコアリングを採用。
  - 白背景や複雑なテクスチャを効果的に識別し、誤検出を大幅に低減します。
- **🛡️ プライバシー安全**：
  - すべての計算はローカルメモリ内で完結します。
  - 画像をクラウドにアップロードする必要はありません。
- **👁️ 高品質な無損比較**：
  - 4K 対応のライトボックスプレビューを内蔵。
  - ピクセル単位で復元結果を確認できる「押して比較」機能をサポート。
- **🌍 多言語・カスタマイズ**：
  - デフォルトで中文（CN）、English（EN）、日本語（JA）をサポート。ブラウザ言語の自動検出に対応。
  - テーマ切替（Dark, Light, Cyberpunk, Deep Ocean, Forest など）を提供。

---
## ✨ デモンストレーション
### 🖌️ **透かし消去デモ**
[![use](https://github.com/1024tool/GeminiWatermarkRemoveToolPro/blob/main/GeminiRemoverToolPro.gif?raw=true)]()

### 🎨 **言語と言語・テーマ設定デモ**
[![use](https://github.com/1024tool/GeminiWatermarkRemoveToolPro/blob/main/Language-theme.gif?raw=true)]()

## 📊 比較
[![use](https://github.com/1024tool/GeminiWatermarkRemoveToolPro/blob/main/Compare.png?raw=true)]()
---

## 🛠️ インストールと実行

### 1. 必要環境
事前に [Go (Golang)](https://go.dev/dl/) をインストールしてください。

### 2. ディレクトリ構成
特定のマスクファイルと静的アセットに依存するため、次の構成を維持してください：

\`\`\`text
ProjectRoot/
├── main.go                # メインプログラム
├── language/              # 言語パック
│   ├── zh-cn.json
│   ├── en.json
│   └── ja.json
└── assets/                # アセット
    ├── index.html         # フロントエンド UI
    ├── favicon.ico        # ファビコン
    ├── bg_48.png          # [重要] 48px ウォーターマスク
    └── bg_96.png          # [重要] 96px ウォーターマスク
\`\`\`

### 3. 実行／ビルド

直接実行：
\`\`\`bash
go run main.go
\`\`\`

単独実行ファイルにビルド（推奨）：
\`\`\`bash
# Windows
go build -o GeminiRemover.exe main.go

# macOS / Linux
go build -o GeminiRemover main.go
\`\`\`

起動後、自動で \`http://localhost:8080\` を既定のブラウザで開きます。

---

## 📖 使い方

### 1. インターフェース概要
- 上部バー：言語切替、閾値スライダー、テーマ切替。
- 検出閾値（Threshold）：
  - 初期値は **25%**。
  - ウォーターマークが検出されない場合は閾値を下げてください。
  - 通常の画像が誤って修正される場合は閾値を上げてください。

### 2. シングルモード（微調整）
1. 画像をアップロードまたはドラッグします。
2. 自動でウォーターマークを検出し、**赤い枠**でマーキングします。
   - 信頼度（例：\`Conf: 95%\`）が表示されます。
   - 手動修正：検出位置がずれている場合は赤枠をドラッグして調整します。
3. **「✨ Remove Watermark」** をクリックします。
4. 除去後、**「👁️ Toggle Compare」** または **「👆 Press-to-Compare」** で結果を確認します。
5. 満足したらダウンロードします。

### 3. バッチモード
1. 「Batch Queue」タブに切り替えます。
2. 複数の画像を一度にドラッグします。
3. 設定した閾値に基づいて処理されます：
   - **緑**：正常に検出・除去された。
   - **黄色**：信頼度が閾値未満のためスキップ（原画像保護のため）。
4. リストのサムネイルをクリックして詳細を確認します。
5. 下部の **「⬇️ Download All」** で結果を保存します。

---

## 🔬 技術的原理

### 1. 逆アルファ合成（Reverse Alpha Blending）
Gemini のウォーターマークは次のように合成されます：
Pixel_final = Pixel_orig * (1 - α) + White * α.
既知のマスクアルファを用いて逆算し、元のピクセルを復元します：
$$Pixel_{orig} = \frac{Pixel_{final} - 255 \times \alpha}{1 - \alpha}$$

### 2. 三段階の検出ロジック
ウォーターマークの存在と位置を確定するために加重スコアリングを使用します：
- Stage 1 (50%): 空間 NCC（輝度とマスクアルファの相関）。
- Stage 2 (30%): 勾配 NCC（Sobel によるエッジ抽出とマスクエッジの一致）。
- Stage 3 (20%): 統計解析（単色領域や露出過多領域を検出し重みを下げる）。

---

## 📝 FAQ

Q: なぜ "Skipped (Low Confidence)" と表示されるのですか？
A: プログラムがその領域をウォーターマークである可能性が低いと判断しています。背景が複雑、またはウォーターマークが薄いことが原因です。閾値を下げてシングルモードで再試行してください。

Q: 除去後にノイズが残るのはなぜですか？
A: 逆復元アルゴリズムの数学的性質によるものです。α が大きい（ウォーターマークが濃い）場合、元のピクセル情報が失われやすく、復元時に誤差が増幅されます。

Q: 対応フォーマットは？
A: \`.jpg\`, \`.jpeg\`, \`.png\` をサポートしています。

---

## ⚖️ 免責事項

本ツールは技術研究および個人学習を目的としています。著作権侵害や不正用途には使用しないでください。画像の著作権は原著作者に帰属します。

---

© 2026 Gemini Watermark Remover Tool Pro.