package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var cachedFont *opentype.Font

type PlaceholderParams struct {
	Width      int
	Height     int
	Text       string
	FontSize   float64
	BgColor    string
	FontColor  string
}

// Load and cache the font at server startup
func preloadFont(fontPath string) error {
	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		return err
	}
	fnt, err := opentype.Parse(fontData)
	if err != nil {
		return err
	}
	cachedFont = fnt
	return nil
}

// Helper to get all query parameters with default values
func getRequestParams(r *http.Request) PlaceholderParams {
	params := PlaceholderParams{
		Width:     400,
		Height:    300,
		Text:      "Placeholder",
		FontSize:  0, // 0 means auto-calculate
		BgColor:   "#FFFFFF",
		FontColor: "#000000",
	}

	// Parse query parameters
	if value := r.URL.Query().Get("width"); value != "" {
		params.Width, _ = strconv.Atoi(value)
	}
	if value := r.URL.Query().Get("height"); value != "" {
		params.Height, _ = strconv.Atoi(value)
	}
	if value := r.URL.Query().Get("text"); value != "" {
		params.Text = value
	}
	if value := r.URL.Query().Get("font_size"); value != "" {
		params.FontSize, _ = strconv.ParseFloat(value, 64)
	}
	if value := r.URL.Query().Get("bg_color"); value != "" {
		params.BgColor = value
	}
	if value := r.URL.Query().Get("font_color"); value != "" {
		params.FontColor = value
	}

	return params
}

// Handler to generate a placeholder image
func placeholderHandler(w http.ResponseWriter, r *http.Request) {
	params := getRequestParams(r)

	// Parse colors
	background, err := parseHexColor(params.BgColor)
	if err != nil {
		http.Error(w, "Invalid background color", http.StatusBadRequest)
		return
	}
	fontColorParsed, err := parseHexColor(params.FontColor)
	if err != nil {
		http.Error(w, "Invalid font color", http.StatusBadRequest)
		return
	}

	// Automatically calculate font size if not specified
	if params.FontSize == 0 {
		params.FontSize = calculateFontSize(params.Text, params.Width, params.Height)
	}

	// Create a new image and draw background
	img := image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)

	// Create font face from the cached font and add text
	face, err := opentype.NewFace(cachedFont, &opentype.FaceOptions{Size: params.FontSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		http.Error(w, "Failed to create font face", http.StatusInternalServerError)
		log.Println("Font face error:", err)
		return
	}
	addText(img, params.Text, face, params.Width, params.Height, fontColorParsed)

	// Encode image as PNG
	w.Header().Set("Content-Type", "image/png")
	if err := png.Encode(w, img); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		log.Println("Encoding error:", err)
	}
}

// Helper to parse hex color
func parseHexColor(colorStr string) (color.Color, error) {
	colorStr = strings.TrimSpace(colorStr)
	if strings.HasPrefix(colorStr, "#") {
		var r, g, b int
		_, err := fmt.Sscanf(colorStr, "#%02x%02x%02x", &r, &g, &b)
		if err != nil {
			return nil, err
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, nil
	} else if strings.HasPrefix(colorStr, "rgb") {
		var r, g, b int
		_, err := fmt.Sscanf(colorStr, "rgb(%d,%d,%d)", &r, &g, &b)
		if err != nil {
			return nil, err
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, nil
	}
	return nil, fmt.Errorf("invalid color format")
}

// Helper to add text to the image
func addText(img *image.RGBA, text string, face font.Face, width, height int, fontColor color.Color) {
	textWidth := font.MeasureString(face, text).Ceil()
	ascent, descent := face.Metrics().Ascent.Ceil(), face.Metrics().Descent.Ceil()
	startX := (width - textWidth) / 2
	startY := (height + ascent - descent) / 2

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fontColor),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(startX << 6), Y: fixed.Int26_6(startY << 6)},
	}
	d.DrawString(text)
}

// Automatically calculate font size to fit the image
func calculateFontSize(text string, width, height int) float64 {
	maxFontSize := float64(height) * 0.4
	fontSize := maxFontSize
	face, _ := opentype.NewFace(cachedFont, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})

	// Decrease font size until the text fits
	for font.MeasureString(face, text).Ceil() > width-20 && fontSize > 10 {
		fontSize -= 1
		face, _ = opentype.NewFace(cachedFont, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})
	}

	return fontSize
}

// Helper to get local IP address dynamically
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("local IP not found")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	// Preload the font
	err := preloadFont("NewAmsterdam-Regular.ttf")
	if err != nil {
		log.Fatal("Failed to preload font:", err)
	}

	// Get the local IP address
	ip, err := getLocalIP()
	if err != nil {
		log.Fatal("Failed to get local IP address:", err)
	}

	http.HandleFunc("/", placeholderHandler)
	fmt.Printf("Server started at http://%s:%s\n", ip, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// example API Call
// GET /placeholder?width=600&height=400&text=Hello%20World&font_size=36&bg_color=%23FF5733&font_color=%23FFFFFF