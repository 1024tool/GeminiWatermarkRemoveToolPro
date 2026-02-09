# ✨ Gemini Watermark Remover Pro (v1.0)

[![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Gemini Watermark Remover Pro** 是一款专为去除 Google Gemini (Imagen 2/3) 生成图片中右下角半透明水印而设计的本地化工具。

它结合了 **Go 语言** 的高性能后端与现代化的 **HTML5/JS** 前端，搭载了先进的 **三阶段 NCC 检测算法**，能够智能识别、精准定位并无损还原图片细节。

---

## 🚀 核心功能

* **🎨 双模式工作流**：
    * **常规模式**：支持交互式操作。如果自动检测位置不准，您可以手动拖拽红框进行修正，并实时对比去水印前后的效果。
    * **批量处理**：支持一次性拖入多张图片，自动排队处理，高效快捷。
* **🧠 智能检测算法 (NCC)**：
    * 采用 **空间相关性 (Spatial)**、**梯度边缘匹配 (Gradient)** 和 **统计方差分析 (Variance)** 三阶段加权评分。
    * 有效区分白色背景、复杂纹理，极大降低误伤率。
* **🛡️ 隐私安全**：
    * 所有计算均在 **本地内存** 中完成。
    * 无需上传图片到云端，保护您的创作隐私。
* **👁️ 高清无损对比**：
    * 内置预览。
    * 支持 **“按住对比”** 功能，直观查看像素级的还原效果。
* **🌍 国际化与个性化**：
    * **多语言支持**：内置 中文 (CN)、English (EN)、日本語 (JA)，支持自动检测浏览器语言。
    * **多主题切换**：提供 Dark, Light, Cyberpunk, Deep Ocean, Forest 等多种酷炫主题。

---

## ✨ 使用效果演示
### 🖌️ **去水印演示**
[![use](https://github.com/1024tool/GeminiWatermarkRemoveToolPro/blob/main/GeminiRemoverToolPro.gif?raw=true)]()

### 🎨 **修改语言和主题演示**
[![use](https://github.com/1024tool/GeminiWatermarkRemoveToolPro/blob/main/Language-theme.gif?raw=true)]()

## 📊 对比

[![use](https://github.com/1024tool/GeminiWatermarkRemoveToolPro/blob/main/Compare.png?raw=true)]()

---

## 🛠️ 安装与运行

### 1. 环境准备
确保您的电脑上已安装 [Go (Golang)](https://go.dev/dl/) 环境。

### 2. 目录结构
由于工具依赖特定的遮罩文件和静态资源，请确保目录结构如下：

```text
ProjectRoot/
├── main.go                # 主程序代码
├── language/              # 语言包目录
│   ├── zh-cn.json
│   ├── en.json
│   └── ja.json
└── assets/                # 资源目录
    ├── index.html         # 前端界面
    ├── favicon.ico        # 网站图标
    ├── bg_48.bin          # [重要] 48px 水印遮罩
    └── bg_96.bin          # [重要] 96px 水印遮罩
```



### 3. 运行或编译

**直接运行：**
```bash
go run main.go
```

**编译为独立程序 (推荐)：**
```bash
# Windows
go build -o GeminiWatermarkRemoveToolPro.exe main.go

# macOS / Linux
go build -o GeminiRemover main.go
```

程序启动后，会自动打开默认浏览器访问 `http://localhost:8080`。

---

## 📖 使用指南

### 1. 界面概览
* **顶部栏**：包含语言切换、阈值滑块、主题切换。
* **检测阈值 (Threshold)**：
    * 默认为 **25%**。
    * 如果发现水印没去掉（漏检），请**调低**阈值。
    * 如果发现正常图片被误修（误检），请**调高**阈值。

### 2. 单张精修模式 (Single Mode)
1.  点击上传或拖入一张图片。
2.  程序会自动检测水印位置并用 **红框** 标记。
    * **状态提示**：会显示置信度（如 `Conf: 95%`）。
    * **手动修正**：如果红框位置不对，鼠标拖动红框到正确位置。
3.  点击 **“✨ 去除水印”**。
4.  去除后，可使用 **“👁️ 切换对比”** 或 **“👆 按住对比”** 查看效果。
5.  满意后点击下载。

### 3. 批量处理模式 (Batch Mode)
1.  切换到“批量队列”标签。
2.  一次性拖入多张图片。
3.  程序会自动根据设定的阈值处理：
    * **绿色**：成功检测并去除。
    * **黄色**：置信度低于阈值，跳过处理（防止破坏原图）。
4.  点击列表中的缩略图可查看详情。
5.  点击底部的 **“⬇️ 批量下载全部”** 保存结果。

---

## 🔬 技术原理

### 1. 逆向 Alpha 混合 (Reverse Alpha Blending)
Gemini 的水印是通过 `Pixel_final = Pixel_orig * (1-α) + White * α` 合成的。
本工具通过已知的水印 Alpha 通道（遮罩），利用逆运算公式还原原始像素：
$$Pixel_{orig} = \frac{Pixel_{final} - 255 \times \alpha}{1 - \alpha}$$

### 2. 三阶段检测逻辑
为了确定水印是否存在以及位置，我们使用了加权评分系统：
* **Stage 1 (50%)**: 空间 NCC。对比图像亮度与遮罩 Alpha 的相关性。
* **Stage 2 (30%)**: 梯度 NCC。利用 Sobel 算子提取边缘，对比图像边缘与水印边缘的重合度。
* **Stage 3 (20%)**: 统计学分析。检测区域是否为纯色或过曝区域（这些区域容易导致数学误判），进行降权处理。

---

## 📝 常见问题 (FAQ)

**Q: 为什么显示 "Skipped (Low Confidence)"？**
A: 这表示程序认为该区域不像水印。可能是背景太杂乱或水印太淡。您可以尝试在顶部将 **“🛡️ 阈值”** 滑块向左拖动（调低数值），然后在单张模式下重试。

**Q: 去除后的图片有些许噪点？**
A: 这是逆向还原算法的数学特性。当 Alpha 值很高（水印很浓）时，原始像素信息丢失较多，还原时会被放大误差。这属于正常现象。

**Q: 支持哪些图片格式？**
A: 支持 `.jpg`, `.jpeg`, `.png` 格式。

---

## ⚖️ 免责声明

本工具仅供技术研究与个人学习使用。请勿用于侵犯版权或非法用途。生成的图片版权归原作者所有。

---

© 2026 Gemini Watermark Remover Tool Pro.