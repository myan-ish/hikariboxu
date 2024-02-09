package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func readFileAsBinary(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func WriteBinaryFile(chunk []byte) (string, error) {
	filename := "1.byt"
	err := os.WriteFile(filename, chunk, 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func BinaryToImage(data []byte, height, width int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, height, width))

	for i, b := range data {
		x := i % width
		y := i / width

		if x < width && y < height {
			img.SetGray(x, y, color.Gray{Y: b})
		}
	}
	return img
}

func main() {
	filename := ".gitignore"

	data, err := readFileAsBinary(filename)

	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	img := BinaryToImage(data, 720, 1080)

	file, err := os.Create("output.png")

	if err != nil {
		panic(err)
	}

	defer file.Close()
	png.Encode(file, img)

	// filename, err = WriteBinaryFile(data)

	// if err != nil {
	// 	log.Fatalf("Failed to read file: %v", err)
	// }
	// fmt.Println("data", filename)
}
