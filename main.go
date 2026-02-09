package main

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io/fs"
	"math"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"path/filepath"
	"strings"
)

//go:embed assets/*
var assetsFS embed.FS

//go:embed language/*
var langFS embed.FS

const (
	MaskSmallSize = 48
	MaskLargeSize = 96
	MarginSmall   = 32
	MarginLarge   = 64
)

type ResponseData struct {
	Filename   string  `json:"filename"`
	Data       string  `json:"data"`
	Status     string  `json:"status"`
	Message    string  `json:"message"`
	Confidence float64 `json:"confidence"`
	BoxX       int     `json:"box_x"`
	BoxY       int     `json:"box_y"`
	BoxW       int     `json:"box_w"`
	BoxH       int     `json:"box_h"`
}

var (
	maskSmallImg, maskLargeImg     image.Image
	maskSmallAlpha, maskLargeAlpha []float64
	maskSmallGrad, maskLargeGrad   []float64
)

func main() {
	fmt.Println("-------------------------------------------")
	fmt.Println("  Gemini Watermark Remover Pro v1.0 ")
	fmt.Println("-------------------------------------------")

	if err := loadEmbeddedMasks(); err != nil {
		fmt.Printf("Error loading masks: %v\n", err)
		return
	}

	// 1. page
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/favicon.ico", handleFavicon)
	
	
	// language
	langContent, err := fs.Sub(langFS, "language")
	if err != nil {
		fmt.Printf("Warning: language folder not found in embed fs: %v\n", err)
	} else {
		
		http.Handle("/lang/", http.StripPrefix("/lang/", http.FileServer(http.FS(langContent))))
	}

	// 
	http.HandleFunc("/upload", handleUpload)

	url := "http://localhost:8080"
	fmt.Printf("Server started: %s\n", url)
	go openBrowser(url)

	http.ListenAndServe(":8080", nil)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := assetsFS.ReadFile("assets/index.html")
	if err != nil {
		http.Error(w, "Missing index.html", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

//  favicon.ico
func handleFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := assetsFS.ReadFile("assets/favicon.ico")
	if err != nil {
		http.Error(w, "Not Found", 404)
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=86400") // 
	w.Write(data)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(50 << 20)
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Invalid file", 400)
		return
	}
	defer file.Close()

	action := r.FormValue("action")
	if action == "" {
		action = "remove"
	}

	manualX, _ := strconv.Atoi(r.FormValue("manual_x"))
	manualY, _ := strconv.Atoi(r.FormValue("manual_y"))
	hasManual := r.FormValue("manual_x") != ""

	thresholdVal := 25.0
	if v, err := strconv.ParseFloat(r.FormValue("threshold"), 64); err == nil {
		thresholdVal = v
	}

	srcImg, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Decode failed", 400)
		return
	}

	bounds := srcImg.Bounds()
	rgbaImg := image.NewRGBA(bounds)
	draw.Draw(rgbaImg, bounds, srcImg, bounds.Min, draw.Src)

	// 1. Determine Mask
	var mask image.Image
	var maskAlpha, maskGrad []float64
	var margin int
	wImg, hImg := bounds.Dx(), bounds.Dy()

	if wImg > 1024 && hImg > 1024 {
		mask, maskAlpha, maskGrad, margin = maskLargeImg, maskLargeAlpha, maskLargeGrad, MarginLarge
	} else {
		mask, maskAlpha, maskGrad, margin = maskSmallImg, maskSmallAlpha, maskSmallGrad, MarginSmall
	}
	mW, mH := mask.Bounds().Dx(), mask.Bounds().Dy()

	// 2. Detection
	finalX, finalY := 0, 0
	confidence := 0.0

	if hasManual && manualX >= 0 {
		finalX, finalY = manualX, manualY
		confidence = 100.0
	} else {
		autoX, autoY := wImg - mW - margin, hImg - mH - margin
		if autoX >= 0 && autoY >= 0 {
			confidence, _ = detectWatermark(rgbaImg, autoX, autoY, mW, mH, maskAlpha, maskGrad)
			finalX, finalY = autoX, autoY
		}
	}

	status := "success"
	msg := "Success"

	// 3. Action
	if action == "detect" {
		if confidence < thresholdVal {
			status = "skipped"
			msg = fmt.Sprintf("未检测到 (%.0f%% < %.0f%%)", confidence, thresholdVal)
		} else {
			msg = fmt.Sprintf("检测到水印 (%.0f%%)", confidence)
		}
	} else {
		if confidence < thresholdVal && !hasManual {
			status = "skipped"
			msg = fmt.Sprintf("置信度低 (%.0f%% < %.0f%%)", confidence, thresholdVal)
		} else {
			removeWatermarkLogic(rgbaImg, mask, finalX, finalY)
			msg = "已去除水印✅"
		}
	}
    newFilename := generateFilename(header.Filename)
	buf := new(bytes.Buffer)
	png.Encode(buf, rgbaImg)

	resp := ResponseData{
		Filename:   newFilename,
		Data:       base64.StdEncoding.EncodeToString(buf.Bytes()),
		Status:     status,
		Message:    msg,
		Confidence: confidence,
		BoxX:       finalX, BoxY: finalY, BoxW: mW, BoxH: mH,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func generateFilename(original string) string {
	ext := filepath.Ext(original)
	name := strings.TrimSuffix(original, ext)
	if ext == "" {
		ext = ".png"
	}
	return name + "_RemoveWatermark" + ext
}

// --- Algorithm (Same as before) ---

func detectWatermark(img *image.RGBA, sx, sy, w, h int, mA, mG []float64) (float64, string) {
	luma := make([]float64, w*h)
	idx := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := img.PixOffset(sx+x, sy+y)
			luma[idx] = 0.299*float64(img.Pix[off]) + 0.587*float64(img.Pix[off+1]) + 0.114*float64(img.Pix[off+2])
			idx++
		}
	}
	s1 := computeNCC(luma, mA)
	if s1 < 0.15 {
		return s1 * 100, "low"
	}
	grad := computeSobel(luma, w, h)
	s2 := computeNCC(grad, mG)
	s3 := computeStats(luma)
	score := 0.5*s1 + 0.3*s2 + 0.2*s3
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score * 100, ""
}

func removeWatermarkLogic(img *image.RGBA, mask image.Image, sx, sy int) {
	mW, mH := mask.Bounds().Dx(), mask.Bounds().Dy()
	for y := 0; y < mH; y++ {
		for x := 0; x < mW; x++ {
			mr, _, _, ma := mask.At(x, y).RGBA()
			alpha := float64(ma) / 65535.0
			if alpha > 0.99 {
				alpha = float64(mr) / 65535.0
			}
			if alpha < 0.02 || alpha > 0.98 {
				continue
			}

			off := img.PixOffset(sx+x, sy+y)
			pr, pg, pb := float64(img.Pix[off]), float64(img.Pix[off+1]), float64(img.Pix[off+2])

			img.Pix[off] = clamp((pr - 255.0*alpha) / (1.0 - alpha))
			img.Pix[off+1] = clamp((pg - 255.0*alpha) / (1.0 - alpha))
			img.Pix[off+2] = clamp((pb - 255.0*alpha) / (1.0 - alpha))
		}
	}
}

func computeNCC(d1, d2 []float64) float64 {
	if len(d1) != len(d2) {
		return 0
	}
	var s1, s2, s1q, s2q, p float64
	n := float64(len(d1))
	for i := 0; i < len(d1); i++ {
		v1, v2 := d1[i], d2[i]
		s1 += v1
		s2 += v2
		s1q += v1 * v1
		s2q += v2 * v2
		p += v1 * v2
	}
	num := p - (s1 * s2 / n)
	den := math.Sqrt((s1q - s1*s1/n) * (s2q - s2*s2/n))
	if den == 0 {
		return 0
	}
	return num / den
}

func computeSobel(l []float64, w, h int) []float64 {
	g := make([]float64, len(l))
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			i := y*w + x
			gx := (l[i-w+1] + 2*l[i+1] + l[i+w+1]) - (l[i-w-1] + 2*l[i-1] + l[i+w-1])
			gy := (l[i+w-1] + 2*l[i+w] + l[i+w+1]) - (l[i-w-1] + 2*l[i-w] + l[i-w+1])
			g[i] = math.Sqrt(gx*gx + gy*gy)
		}
	}
	return g
}

func computeStats(l []float64) float64 {
	var s, sq float64
	n := float64(len(l))
	for _, v := range l {
		s += v
		sq += v * v
	}
	m := s / n
	v := sq/n - m*m
	sv := 1.0
	if v < 50 {
		sv = v / 50.0
	}
	sm := 1.0
	if m > 240 {
		sm = (255 - m) / 15.0
		if sm < 0 {
			sm = 0
		}
	}
	return sv * sm
}

func loadEmbeddedMasks() error {
	f1, _ := assetsFS.ReadFile("assets/bg_48.bin")
	maskSmallImg, _, _ = image.Decode(bytes.NewReader(f1))
	f2, _ := assetsFS.ReadFile("assets/bg_96.bin")
	maskLargeImg, _, _ = image.Decode(bytes.NewReader(f2))

	prep := func(m image.Image) ([]float64, []float64) {
		w, h := m.Bounds().Dx(), m.Bounds().Dy()
		a := make([]float64, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, _, _, al := m.At(x, y).RGBA()
				v := float64(al) / 65535.0
				if v > 0.99 {
					v = float64(r) / 65535.0
				}
				a[y*w+x] = v * 255.0
			}
		}
		return a, computeSobel(a, w, h)
	}
	maskSmallAlpha, maskSmallGrad = prep(maskSmallImg)
	maskLargeAlpha, maskLargeGrad = prep(maskLargeImg)
	return nil
}

func clamp(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}
func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}