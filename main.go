package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func readFileAsBinary(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readFileAsChunkBinary(filename string, width, height, bufferSize int) {
	file, err := os.Open(filename)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file stats:", err)
		return
	}

	// Get file size
	fileSize := fileInfo.Size()

	fmt.Printf("The size of '%s' is %d bytes.\n", filename, fileSize)

	buffer := make([]byte, bufferSize)

	reader := bufio.NewReader(file)

	counter := 1

	for {

		bytesRead, err := reader.Read(buffer)

		if err != nil {
			if err.Error() != "EOF" {
				fmt.Println("Error reading chunk:", err)
			}
			break
		}
		img := BinaryToImage(buffer[:bytesRead], width, height)

		file, err = os.Create(strconv.Itoa(counter) + ".png")

		if err != nil {
			panic(err)
		}

		defer file.Close()
		png.Encode(file, img)

		fmt.Println(counter)
		counter = counter + 1
	}

}

func readFileAsChunkBinaryChannel(filename string, width, height, bufferSize, workers int) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file stats:", err)
		return
	}

	fileSize := fileInfo.Size()
	fmt.Printf("The size of '%s' is %d bytes.\n", filename, fileSize)

	chunks := int(fileSize) / bufferSize
	if int(fileSize)%bufferSize != 0 {
		chunks++
	}

	var wg sync.WaitGroup
	chunkChannel := make(chan int, chunks)

	for w := 1; w <= workers; w++ {
		wg.Add(1)
		go toImageWorker(w, chunkChannel, &wg, filename, width, height, bufferSize)
	}

	for i := 1; i <= chunks; i++ {
		chunkChannel <- i
	}
	close(chunkChannel)

	wg.Wait()
}

func toImageWorker(id int, chunks <-chan int, wg *sync.WaitGroup, filename string, width, height, bufferSize int) {
	defer wg.Done()

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Worker %d: Error opening file: %v\n", id, err)
		return
	}
	defer file.Close()

	for chunkIndex := range chunks {
		offset := int64((chunkIndex - 1) * bufferSize)
		buffer := make([]byte, bufferSize)

		file.Seek(offset, 0)
		bytesRead, err := file.Read(buffer)
		if err != nil {
			fmt.Printf("Worker %d: Error reading chunk: %v\n", id, err)
			continue
		}

		img := BinaryToImage(buffer[:bytesRead], width, height)
		outputFile, err := os.Create("temp/" + strconv.Itoa(chunkIndex) + ".png")
		if err != nil {
			fmt.Printf("Worker %d: Error creating image file: %v\n", id, err)
			continue
		}

		png.Encode(outputFile, img)
		outputFile.Close()

		fmt.Printf("Worker %d: Processed chunk %d\n", id, chunkIndex)
	}
}

func getFilePaths(dirPath string) ([]string, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var filePaths []string
	for _, file := range files {
		filePaths = append(filePaths, filepath.Join(dirPath, file.Name()))
	}

	sort.Slice(filePaths, func(i, j int) bool {
		numI, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(filePaths[i]), ""), ".png"))
		numJ, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(filePaths[j]), ""), ".png"))
		return numI < numJ
	})

	return filePaths, nil
}

func toFileWorker(id int, chunks <-chan int, wg *sync.WaitGroup, file_paths []string) {
	defer wg.Done()

	outfile_path := "decoded"
	output, err := os.OpenFile(outfile_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	for _, file := range file_paths {
		data, err := ImageToByte(file)
		if err != nil {
			log.Printf("Error decoding image %s: %v", file, err)
			continue
		}

		// Assuming `ImageToByte` handles determining the actual data length
		if _, err := output.Write(data); err != nil {
			log.Fatal(err)
		}
	}
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
			snippet := byte(grey >> 8)
			if snippet != 0 {
				data = append(data, snippet)
			}
		}
	}
	return data, nil
}

func encodeFileToImage(filename, output_image string, width, height int) {
	data, err := readFileAsBinary(filename)

	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	img := BinaryToImage(data, width, height)

	file, err := os.Create(output_image)

	if err != nil {
		panic(err)
	}

	defer file.Close()
	png.Encode(file, img)

}

func createVideoFromImagesFFMPEG(folder string) {
	cmd := exec.Command("ffmpeg", "-framerate", "30", "-i", "temp/%d.png", "-c:v", "ffv1", "-level", "3", "output.mkv")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		log.Printf("stderr: %v\n", stderr.String())
	}
	log.Printf("Output: %v\n", out.String())
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
}

func decodeVideoToBinaryFile(video, output_folder string, workers int) {
	cmd := exec.Command("ffmpeg", "-i", video, "-vf", "fps=30", "outp/frame%d.png")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		log.Printf("stderr: %v\n", stderr.String())
	}
	log.Printf("Output: %v\n", out.String())

	file_paths, _ := getFilePaths("temp")

	chunks := len(file_paths) / workers

	if int(len(file_paths))%workers != 0 {
		chunks++
	}

	var wg sync.WaitGroup
	chunkChannel := make(chan int, chunks)

	for w := 1; w <= workers; w++ {
		wg.Add(1)
		go toFileWorker(w, chunkChannel, &wg, file_paths)
	}

	for i := 1; i <= chunks; i++ {
		chunkChannel <- i
	}
	close(chunkChannel)

	wg.Wait()

}

func main() {
	os.Mkdir("temp", 0755)
	os.Mkdir("outp", 0755)
	readFileAsChunkBinaryChannel("test", 500, 500, 250000, 5)
	createVideoFromImagesFFMPEG("temp")
	decodeVideoToBinaryFile("output.mkv", "outp", 5)
}
