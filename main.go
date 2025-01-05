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
var cachedFontFaces = map[float64]font.Face{}
var img *image.RGBA

type PlaceholderParams struct {
	Width     int
	Height    int
	Text      string
	FontSize  float64
	BgColor   string
	FontColor string
}

func getFontFace(fontSize float64) (font.Face, error) {
	// Check if font face is already cached
	if face, ok := cachedFontFaces[fontSize]; ok {
		return face, nil
	}

	// If not cached, create and cache it
	face, err := opentype.NewFace(cachedFont, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	// Limit cache size
	if len(cachedFontFaces) >= 5 {
		for k := range cachedFontFaces {
			delete(cachedFontFaces, k)
			break
		}
	}

	cachedFontFaces[fontSize] = face
	return face, nil
}

// @title Placeholder API
// @version 1.0
// @description This is an API to generate placeholder images.
// @BasePath /

// @Summary Generate a placeholder image
// @Description Generates a placeholder image with specified dimensions, text, and colors. The image size is limited to a maximum of 1000x1000 pixels.
// @Produce png
// @Param w query int false "Width of the image (max 1920)" default(400)
// @Param h query int false "Height of the image (max 1920)" default(300)
// @Param text query string false "Text to display" default(Placeholder)
// @Param font_size query float64 false "Font size of the text"
// @Param bg_color query string false "Background color in 8-character hex format without '#'. The last 2 characters represent alpha (transparency)" default(FFFFFF00)
// @Param font_color query string false "Font color in 8-character hex format without '#'. The last 2 characters represent alpha (transparency)" default(000000FF)
// @Success 200 {file} png "The generated placeholder image"
// @Failure 400 {string} string "Invalid input parameters or image too large"
// @Failure 500 {string} string "Internal server error, failed to generate image or encode it"
// @Router /placeholder [get]
func placeholderHandler(w http.ResponseWriter, r *http.Request) {
	params := getRequestParams(r)

	// Parse background and font colors
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

	// Reuse or create a new image if necessary
	if img == nil || img.Rect.Dx() != params.Width || img.Rect.Dy() != params.Height {
		if params.Width > 1920 || params.Height > 1920 {
			http.Error(w, "Image size too large", http.StatusBadRequest)
			return
		}
		img = image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))
	} else {
		// Reset image contents
		draw.Draw(img, img.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	}

	face, err := getFontFace(params.FontSize)
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
	// Unconditionally remove the '#' prefix if it exists
	colorStr = strings.TrimPrefix(colorStr, "#")

	// Ensure the color string is exactly 8 characters long (RGBA format)
	if len(colorStr) != 8 {
		return nil, fmt.Errorf("invalid color format: must be an 8-character hex string (RGBA)")
	}

	var r, g, b, a int
	_, err := fmt.Sscanf(colorStr, "%02x%02x%02x%02x", &r, &g, &b, &a)
	if err != nil {
		return nil, fmt.Errorf("invalid color format: %v", err)
	}

	// Return the parsed RGBA color
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}, nil
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

	// Pre-create the font face once for the starting size
	face, _ := getFontFace(fontSize)

	for font.MeasureString(face, text).Ceil() > width-20 && fontSize > 10 {
		fontSize -= 1
		face, _ = getFontFace(fontSize) // Reuse the same method to get the font face
	}

	return fontSize
}

func getRequestParams(r *http.Request) PlaceholderParams {
	params := PlaceholderParams{
		Width:     400,
		Height:    300,
		Text:      "Placeholder",
		FontSize:  0,
		BgColor:   "FFFFFF00",
		FontColor: "000000FF",
	}

	if value := r.URL.Query().Get("width"); value != "" || r.URL.Query().Get("w") != "" {
		if value == "" {
			value = r.URL.Query().Get("w") // Use `w` if `width` is empty
		}
		params.Width, _ = strconv.Atoi(value)
	}

	if value := r.URL.Query().Get("height"); value != "" || r.URL.Query().Get("h") != "" {
		if value == "" {
			value = r.URL.Query().Get("h") // Use `h` if `height` is empty
		}
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
