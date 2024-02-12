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
	"time"

	"github.com/joho/godotenv"
)

type Res struct {
	width  int
	height int
}

var RES = map[string]Res{
	"120":  {width: 160, height: 120},   // Very low resolution
	"240":  {width: 320, height: 240},   // Low resolution
	"480":  {width: 640, height: 480},   // Standard definition
	"720":  {width: 1280, height: 720},  // HD
	"1080": {width: 1920, height: 1080}, // Full HD
	"1440": {width: 2560, height: 1440}, // Quad HD
	"4K":   {width: 3840, height: 2160}, // 4K Ultra HD
}

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

	for chunkIndex := range chunks {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Worker %d: Error opening file: %v\n", id, err)
			continue // Skip this chunk if the file cannot be opened
		}

		offset := int64((chunkIndex - 1) * bufferSize)
		buffer := make([]byte, bufferSize)

		_, err = file.Seek(offset, 0)
		if err != nil {
			fmt.Printf("Worker %d: Error seeking in file: %v\n", id, err)
			file.Close() // Ensure file is closed even if seek fails
			continue
		}

		bytesRead, err := file.Read(buffer)
		file.Close() // Close the file after reading the necessary chunk
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
		// Extracting number from filename assuming format "frame<number>.png"
		numI, errI := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(filePaths[i]), "frame"), ".png"))
		numJ, errJ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(filePaths[j]), "frame"), ".png"))
		// If error occurs in Atoi, handle it or log it, for now just return false
		if errI != nil || errJ != nil {
			fmt.Printf("Error parsing filenames: %v, %v\n", errI, errJ)
			return false // Consider how to handle this case in your context
		}
		return numI < numJ
	})
	return filePaths, nil
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
			// if snippet != 0 {
			// 	data = append(data, snippet)
			// }
			data = append(data, snippet)
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

func createVideoFromImagesFFMPEG(folder, encoded_video string) {
	cmd := exec.Command("ffmpeg", "-framerate", "30", "-i", "temp/%d.png", "-c:v", "ffv1", "-level", "3", encoded_video)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		log.Printf("stderr: %v\n", stderr.String())
	}
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
	cmd := exec.Command("ffmpeg", "-i", video, "-vf", "fps=30", output_folder+"/frame%d.png")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("cmd.Run() failed with %s\n", err)
		log.Printf("stderr: %v\n", stderr.String())
		return
	}
	log.Printf("Output: %v\n", out.String())

	file_paths, _ := getFilePaths(output_folder)

	chunks := len(file_paths) / workers
	if len(file_paths)%workers != 0 {
		chunks++
	}

	// Divide file paths evenly among workers
	var wg sync.WaitGroup
	numFiles := len(file_paths)
	filesPerWorker := (numFiles + workers - 1) / workers // Ensure rounding up if not divisible

	for w := 0; w < workers; w++ {
		wg.Add(1)
		start := w * filesPerWorker
		end := start + filesPerWorker
		if end > numFiles {
			end = numFiles
		}
		go toFileWorker(w+1, file_paths[start:end], &wg, output_folder)
	}

	wg.Wait()

	// Combine the chunks
	combineChunks(workers, output_folder)
}

func toFileWorker(id int, file_paths []string, wg *sync.WaitGroup, output_folder string) {
	defer wg.Done()

	outfile_path := output_folder + "/output_chunk" + strconv.Itoa(id)
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
		if _, err := output.Write(data); err != nil {
			log.Fatal(err)
		}
	}
}

func combineChunks(workers int, output_folder string) {
	finalOutputPath := "decoded"
	finalOutput, err := os.OpenFile(finalOutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer finalOutput.Close()

	for i := 1; i <= workers; i++ {
		chunkPath := filepath.Join(output_folder, fmt.Sprintf("output_chunk%d", i))
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			log.Printf("Error reading chunk %s: %v", chunkPath, err)
			continue
		}
		if _, err := finalOutput.Write(chunkData); err != nil {
			log.Fatal(err)
		}
	}
}

func encode(input_file string) {
	os.Mkdir("temp", 0755)

	res := os.Getenv("resolution")
	workers := os.Getenv("workers")

	resolution, ok := RES[res]

	if !ok {
		fmt.Println("Invalid resolution set in env.")
	}

	workerCount, err := strconv.Atoi(workers)
	if err != nil {
		log.Fatal(err)
	}

	readFileAsChunkBinaryChannel(input_file, resolution.width, resolution.height, resolution.height*resolution.width, workerCount)
	createVideoFromImagesFFMPEG("temp", "encoded_vid.mkv")

	os.RemoveAll("temp")
}

func decode(file_path string) {
	os.Mkdir("outp", 0755)

	workers := os.Getenv("workers")

	workerCount, err := strconv.Atoi(workers)
	if err != nil {
		log.Fatal(err)
	}

	decodeVideoToBinaryFile(file_path, "outp", workerCount)
	os.RemoveAll("outp")
}

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage: <program> <file> <action: encode/decode>")
		os.Exit(1)
	}

	file := os.Args[1]
	action := os.Args[2]

	if action != "encode" && action != "decode" {
		fmt.Println("action should be either encode or decode")
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	start := time.Now()
	fmt.Println("Start time: ", start)
	if action == "encode" {
		encode(file)
	} else {
		decode(file)
	}
	elapsed := time.Since(start)
	fmt.Println("Elapsed time: ", elapsed)
}
