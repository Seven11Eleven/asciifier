package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
	"golang.org/x/term"
)

var (
	gradient  = " .:!/r(lZ4H9W8$@"
	gradient2 = " .'`^\",:;Il!i~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"
)

func getTerminalSize() (int, int) {
	width, height, err := term.GetSize(int(syscall.Stdout))
	if err != nil {
		return 120, 40
	}
	return width, height
}

func openImg(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(file)
	case ".png":
		return png.Decode(file)
	case ".webp":
		return webp.Decode(file)
	default:
		return nil, fmt.Errorf("unsupported file extension: %v", ext)
	}
}

func frameToTest(img image.Image, width, height int, mode string) string {
	bounds := img.Bounds()
	var result strings.Builder

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			px := img.At(x*bounds.Dx()/width, y*bounds.Dy()/height)
			r, g, b, _ := px.RGBA()
			r8, g8, b8 := r>>8, g>>8, b>>8

			grayscale := 0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8)
			idx := int(math.Round(grayscale / 255 * float64(len(gradient2)-1)))

			switch mode {
			case "ascii":
				result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm%s", r8, g8, b8, string(gradient2[idx])))
			case "blocks":
				result.WriteString(fmt.Sprintf("\033[48;2;%d;%d;%dm ", r8, g8, b8))
			case "halfblocks":
				if y+1 < height {
					lowerPx := img.At(x*bounds.Dx()/width, (y+1)*bounds.Dy()/height)
					r2, g2, b2, _ := lowerPx.RGBA()
					r2, g2, b2 = r2>>8, g2>>8, b2>>8

					upperColor := fmt.Sprintf("\033[38;2;%d;%d;%dm", r8, g8, b8)
					lowerColor := fmt.Sprintf("\033[48;2;%d;%d;%dm", r2, g2, b2)

					result.WriteString(lowerColor + upperColor + "▀")
				} else {
					result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▀", r8, g8, b8))
				}
			}
		}
		result.WriteString("\033[0m\n")
	}

	return result.String()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func resizeImage(img image.Image, newWidth, newHeight int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func processVidos(path string, mode string, width int) {
	cmd := exec.Command("ffmpeg", "-i", path, "-vf", fmt.Sprintf("fps=6,scale=%d:-1:flags=lanczos", width), "-f", "image2pipe", "-vcodec", "mjpeg", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 4096)
	var frameData []byte
	startMarker := []byte{0xFF, 0xD8}
	endMarker := []byte{0xFF, 0xD9}

	fmt.Print("\033[?25l")

	for {
		n, err := stdout.Read(buf)
		if err != nil {
			break
		}
		frameData = append(frameData, buf[:n]...)
		startIdx := bytes.Index(frameData, startMarker)
		endIdx := bytes.Index(frameData, endMarker)

		if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
			jpegFrame := frameData[startIdx : endIdx+2]
			frameData = frameData[endIdx+2:]

			img, err := jpeg.Decode(bytes.NewReader(jpegFrame))
			if err != nil {
				continue
			}

			if mode == "blocks" {
				fmt.Print("\033[2J")
			}
			fmt.Print("\033[1;1H")
			fmt.Println(frameToTest(img, width, width/2, mode))
			fmt.Print("\033[0m")
			time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Print("\033[?25h")
}

func processImage(path string, mode string, width int) {
	img, err := openImg(path)
	if err != nil {
		log.Fatalf("Ошибка загрузки изображения: %v", err)
	}

	height := width / 2

	img = resizeImage(img, width, height)
	fmt.Println(frameToTest(img, width, height, mode))
}

func resizeImageBlocks(img image.Image, newWidth int) image.Image {
	bounds := img.Bounds()
	newHeight := int(float64(bounds.Dy()) / float64(bounds.Dx()) * float64(newWidth) * 1.0)

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func main() {
	inFlag := flag.String("input", "", "Путь к файлу (изображение или видео)")
	modeFlag := flag.String("mode", "ascii", "Режим вывода: ascii, blocks, halfblocks")
	widthFlag := flag.Int("width", 0, "Ширина в символах (0 - авто)")

	flag.Parse()

	if *inFlag == "" {
		log.Fatal("Ошибка: укажите путь к файлу через --input")
	}

	termWidth, _ := getTerminalSize()
	width := *widthFlag
	if width == 0 {
		width = termWidth - 2
	}

	ext := strings.ToLower(filepath.Ext(*inFlag))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		processImage(*inFlag, *modeFlag, width)
	case ".mp4", ".mov", ".avi", ".mkv", ".webm":
		processVidos(*inFlag, *modeFlag, width)
	default:
		log.Fatal("Ошибка: неподдерживаемый формат файла")
	}
}
