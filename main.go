package main

import (
	"fmt"
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

func WriteBinaryFile(filename string, chunk []byte) (string, error) {
	err := os.WriteFile(filename, chunk, 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func BinaryToImage(data []byte, width, height int) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, width, height))

	for i, b := range data {
		x := i % width
		y := i / width

		if x < width && y < height {
			img.SetGray(x, y, color.Gray{Y: b})
		}
	}
	return img
}

func ImageToByte(filename string) ([]byte, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)

	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	data := make([]byte, 0, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grey, _, _, _ := img.At(x, y).RGBA()
			data = append(data, byte(grey>>8))
		}
	}
	return data, nil
}

func encodeFileToImage(filename, output_image string, width, height int) {
	data, err := readFileAsBinary(filename)

	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	fmt.Println(data)

	img := BinaryToImage(data, width, height)

	file, err := os.Create(output_image)

	if err != nil {
		panic(err)
	}

	defer file.Close()
	png.Encode(file, img)

}

func decodeImageToFile(input_image, output_file string) {
	outputFilename := output_file
	// imageFilenames := []string{input_image}

	outputFile, err := os.Create(outputFilename)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	data, err := ImageToByte(input_image)
	fmt.Println(data)

	if err != nil {
		panic(err)
	}

	_, err = outputFile.Write(data)
	if err != nil {
		panic(err)
	}

	// for _, filename := range imageFilenames {
	// 	data, err := ImageToByte(filename) // Convert each image back to binary data
	// 	if err != nil {
	// 		panic(err) // Proper error handling is advised
	// 	}

	// 	_, err = outputFile.Write(data) // Write the binary data to the output file
	// 	if err != nil {
	// 		panic(err) // Proper error handling is advised
	// 	}
	// }
}

func main() {
	encodeFileToImage(".gitignore", "output.png", 1080, 720)

	decodeImageToFile("output.png", "decoded")

}
