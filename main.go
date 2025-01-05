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

	_ "placeholder-api/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

var cachedFont *opentype.Font

type PlaceholderParams struct {
	Width     int
	Height    int
	Text      string
	FontSize  float64
	BgColor   string
	FontColor string
}

// @title Placeholder API
// @version 1.0
// @description This is an API to generate placeholder images.
// @BasePath /

// @Summary Generate a placeholder image
// @Description Generates a placeholder image with specified dimensions, text, and colors.
// @Produce png
// @Param width query int false "Width of the image" default(400)
// @Param height query int false "Height of the image" default(300)
// @Param text query string false "Text to display" default(Placeholder)
// @Param font_size query float64 false "Font size of the text"
// @Param bg_color query string false "Background color in hex format" default(#FFFFFF)
// @Param font_color query string false "Font color in hex format" default(#000000)
// @Success 200 {file} png
// @Failure 400 {string} string "Invalid input parameters"
// @Router /placeholder [get]
func placeholderHandler(w http.ResponseWriter, r *http.Request) {
	params := getRequestParams(r)

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

	if params.FontSize == 0 {
		params.FontSize = calculateFontSize(params.Text, params.Width, params.Height)
	}

	img := image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)

	face, err := opentype.NewFace(cachedFont, &opentype.FaceOptions{Size: params.FontSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		http.Error(w, "Failed to create font face", http.StatusInternalServerError)
		log.Println("Font face error:", err)
		return
	}
	addText(img, params.Text, face, params.Width, params.Height, fontColorParsed)

	w.Header().Set("Content-Type", "image/png")
	if err := png.Encode(w, img); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		log.Println("Encoding error:", err)
	}
}

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

func calculateFontSize(text string, width, height int) float64 {
	maxFontSize := float64(height) * 0.4
	fontSize := maxFontSize
	face, _ := opentype.NewFace(cachedFont, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})

	for font.MeasureString(face, text).Ceil() > width-20 && fontSize > 10 {
		fontSize -= 1
		face, _ = opentype.NewFace(cachedFont, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})
	}

	return fontSize
}

func getRequestParams(r *http.Request) PlaceholderParams {
	params := PlaceholderParams{
		Width:     400,
		Height:    300,
		Text:      "Placeholder",
		FontSize:  0,
		BgColor:   "#FFFFFF",
		FontColor: "#000000",
	}

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

	err := preloadFont("NewAmsterdam-Regular.ttf")
	if err != nil {
		log.Fatal("Failed to preload font:", err)
	}

	ip, err := getLocalIP()
	if err != nil {
		log.Fatal("Failed to get local IP address:", err)
	}

	http.HandleFunc("/", httpSwagger.WrapHandler)
	http.HandleFunc("/placeholder", placeholderHandler)
	fmt.Printf("Server started at http://%s:%s\n", ip, port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
