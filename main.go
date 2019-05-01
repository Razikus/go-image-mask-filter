package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	_ "image/png"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type PixelFilterIterator interface {
	calculateMask() int64
	nextX() bool
	nextY() bool
	filerWithMask() bool
	filterWithMask() bool
	getRelativePixel(int, int) color.Color
}

type PixelFilterIteratorStruct struct {
	originalData  *image.Image
	processedData *image.RGBA64
	x             int
	y             int
	sizeX         int
	sizeY         int
	currentPixel  color.Color
	mask          *[][]float64
	maskSum       float64
}

func (pixelIterator *PixelFilterIteratorStruct) calculateMask() float64 {
	summedMaskValue := 0.0
	for y := 0; y < len(*pixelIterator.mask); y++ {
		for x := 0; x < len((*pixelIterator.mask)[0]); x++ {
			summedMaskValue = summedMaskValue + (*pixelIterator.mask)[y][x]
		}
	}
	if summedMaskValue == 0 {
		return 1
	}
	return summedMaskValue
}

func (pixelIterator *PixelFilterIteratorStruct) nextX() bool {
	if pixelIterator.x < pixelIterator.sizeX {
		pixelIterator.x = pixelIterator.x + 1
		original := *pixelIterator.originalData
		pixelIterator.currentPixel = original.At(pixelIterator.x, pixelIterator.y)
		return true
	} else {
		return false
	}
}

func (pixelIterator *PixelFilterIteratorStruct) nextY() bool {
	if pixelIterator.y < pixelIterator.sizeY {
		pixelIterator.y = pixelIterator.y + 1
		pixelIterator.x = 0
		original := *pixelIterator.originalData
		fmt.Print(pixelIterator.y, "/", pixelIterator.sizeY, "\r")
		pixelIterator.currentPixel = original.At(pixelIterator.x, pixelIterator.y)
		return true
	} else {
		return false
	}
}

func (pixelIterator *PixelFilterIteratorStruct) getRelativePixel(xDelta int, yDelta int) color.Color {
	xAfter := pixelIterator.x + xDelta
	yAfter := pixelIterator.y + yDelta
	if xAfter >= pixelIterator.sizeX || xAfter < 0 {
		return nil
	} else if yAfter >= pixelIterator.sizeY || yAfter < 0 {
		return nil
	}
	original := *pixelIterator.originalData
	return original.At(xAfter, yAfter)
}

func (pixelIterator *PixelFilterIteratorStruct) filterCurrentPixel() bool {
	center := int(len(*pixelIterator.mask) / 2)
	summed := []float64{0, 0, 0, 0}
	pixelData := []uint16{0, 0, 0, 0}
	maxMaskY := len(*pixelIterator.mask)
	maxMaskX := len((*pixelIterator.mask)[0])
	for y := 0; y < maxMaskY; y++ {
		for x := 0; x < maxMaskX; x++ {
			maskVal := (*pixelIterator.mask)[y][x]
			if maskVal == 0 {
				continue
			}
			pixelVal := pixelIterator.getRelativePixel(x-center, y-center)
			if pixelVal == nil {
				continue
			}

			r, g, b, _ := pixelVal.RGBA()

			summed[0] = summed[0] + float64(r)*maskVal
			summed[1] = summed[1] + float64(g)*maskVal
			summed[2] = summed[2] + float64(b)*maskVal
		}
	}
	for index := 0; index < len(summed); index++ {
		sum := summed[index] / (pixelIterator.maskSum)

		if sum < 0 {
			pixelData[index] = 0
		} else if sum >= 65535 {
			pixelData[index] = 65535
		} else {
			pixelData[index] = uint16(summed[index] / (pixelIterator.maskSum))
		}
	}
	_, _, _, a := pixelIterator.getRelativePixel(0, 0).RGBA()
	newPixelData := color.RGBA64{R: pixelData[0], G: pixelData[1], B: pixelData[2], A: uint16(a)}
	(*pixelIterator.processedData).SetRGBA64(pixelIterator.x, pixelIterator.y, newPixelData)

	return true
}

func (pixelIterator *PixelFilterIteratorStruct) filterWithMask() bool {
	pixelIterator.maskSum = pixelIterator.calculateMask()
	for y := 0; y < pixelIterator.sizeY; y++ {
		for x := 0; x < pixelIterator.sizeX; x++ {
			pixelIterator.filterCurrentPixel()
			pixelIterator.nextX()
		}
		pixelIterator.nextY()

	}
	return true
}

func searchFor(needle string, hay []string) bool {
	for index := 0; index < len(hay); index++ {
		if hay[index] == needle {
			return true
		}
	}
	return false
}

func main() {
	SUPPORTED_TYPES := make([]string, 2)
	SUPPORTED_TYPES[0] = "png"
	SUPPORTED_TYPES[1] = "jpeg"

	if len(os.Args) < 4 {
		fmt.Println("Usage: imageToProcess imageResult maskFile")
		os.Exit(1)
	}
	existingImageFile, err := os.Open(os.Args[1])
	if err != nil {

	}
	defer existingImageFile.Close()

	imageData, imageType, err := image.Decode(existingImageFile)
	if err != nil {
	}
	fmt.Println("ImageType:", imageType)
	if !searchFor(imageType, SUPPORTED_TYPES) {
		fmt.Println("Not supporting this image type")
		os.Exit(1)
	}

	bounds := imageData.Bounds()
	newDataRect := image.Rect(0, 0, bounds.Size().X, bounds.Size().Y)
	newDataImage := image.NewRGBA64(newDataRect)

	readed, err := ioutil.ReadFile(os.Args[3])
	if err != nil {
		fmt.Println(err)
	}

	splitted := strings.Split(strings.TrimSpace(string(readed)), "\n")

	var mask = make([][]float64, len(splitted))
	for yIndex, value := range splitted {
		xRow := strings.Split(value, ",")
		mask[yIndex] = make([]float64, len(xRow))

		for xIndex, valueX := range xRow {
			converted, err := strconv.ParseFloat(strings.TrimSpace(valueX), 64)
			if err != nil {
				fmt.Println(err)
				break
			}
			mask[yIndex][xIndex] = converted
		}
	}
	fmt.Println("Loaded mask:")
	fmt.Println(mask)

	iteratorStruct := PixelFilterIteratorStruct{&imageData, newDataImage, 0, 0, bounds.Size().X, bounds.Size().Y, imageData.At(0, 0), &mask, 0}
	iteratorStruct.filterWithMask()

	outputFile, err := os.Create(os.Args[2])
	if err != nil {
	}
	if imageType == "png" {
		png.Encode(outputFile, iteratorStruct.processedData)
	} else if imageType == "jpeg" {
		jpeg.Encode(outputFile, iteratorStruct.processedData, nil)
	}

	outputFile.Close()

}
