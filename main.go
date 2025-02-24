package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

var imgPath = "./kachok.jpg"

var (
	gradient  = " .:!/r(lZ4H9W8$@"
	gradient2 = "$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/|()1{}[]?-_+~<>i!lI;:"
)

func resizeImage(img image.Image, newWidth, newHeight int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func grayscaleImage(img image.Image) *image.Gray {
	bounds := img.Bounds()
	grayImg := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayImg.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return grayImg
}

func openImg(filename string) (image.Image, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(filename)
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

func main() {
	inFlag := flag.String("input", "", "path to file to asciifying")
	flag.Parse()

	if *inFlag == "" {
		log.Printf("no user's image provided")
		*inFlag = imgPath
	}

	img, err := openImg(*inFlag)
	if err != nil {
		fmt.Print(err)
		return
	}

	img = resizeImage(img, 200, 100)
	// img = grayscaleImage(img)

	bounds := img.Bounds()

	for y := 0; y < bounds.Max.Y; y++ {
		for x := 0; x < bounds.Max.X; x++ {
			color := img.At(x, y)
			r, g, b, _ := color.RGBA()

			r8 := r >> 8
			g8 := g >> 8
			b8 := b >> 8

			/*			if int(r8+g8+b8) != 3*255 {
							fmt.Printf("RGB at (w:%v, h:%v):\nR:%v, G:%v, B:%v\n", x, y, r>>8, g>>8, b>>8)
						}

			*/

			clr := math.Round((float64(r8) + float64(g8) + float64(b8)) / 3)
			idx := int(math.Round(clr / 255 * float64(len(gradient)-1)))

			//   fmt.Print(gradient[idx])

			fmt.Printf("\033[38;2;%d;%d;%dm%s", r8, g8, b8, string(gradient[idx]))
		}
		fmt.Println("\033[0m")
	}
}
