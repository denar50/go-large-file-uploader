package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"
)

var FILE_TO_UPLOAD = "./1gb_file"

const UPLOAD_ROUTINES = 10
const DOWNLOAD_ROUTINES = 10

const url = "http://localhost:3000"

type createFileRequestBody struct {
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
}

type createFileResponseBody struct {
	Id         string `json:"id"`
	ChunkSize  int    `json:"chunk_size"`
	ChunkCount int    `json:"chunk_count"`
}

type fileChunkRequestBody struct {
	Checksum     string `json:"checksum"`
	FromPosition int64  `json:"from_position"`
	ChunkNumber  int    `json:"chunk_number"`
}

type getFileResponseBody struct {
	Id         string `json:"id"`
	Checksum   string `json:"checksum"`
	Size       int64  `json:"size"`
	ChunkCount int    `json:"chunk_count"`
}

type getFileChunkResponseBody struct {
	Checksum     string `json:"checksum"`
	FromPosition int64  `json:"from_position"`
}

type fileChunk struct {
	fromPosition int64
	chunkNumber  int
	checksum     string
	data         []byte
}

func readFileChunks(fileName string, chunkSize int, chunksChan chan fileChunk) {
	position := int64(0)
	chunkNumber := 0
	fmt.Println("Read file chunks called")
	fileHandle, err := os.Open(fileName)

	if err != nil {
		fmt.Println("Failed to open the file to upload")
		panic(err)
	}

	fileInfo, err := fileHandle.Stat()

	if err != nil {
		fmt.Println("Failed to read file stats", err.Error())
		panic(err)
	}

	fileSize := fileInfo.Size()

	if fileSize < int64(chunkSize) {
		chunkSize = int(fileSize)
	}

	for {
		if fileSize-position <= 0 {
			break
		}
		if fileSize-position < int64(chunkSize) {
			chunkSize = int(fileSize - position)
		}
		buffer := make([]byte, chunkSize)
		bytesRead, err := fileHandle.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading from file:", err.Error())
			}
			break
		}

		sha := sha256.Sum256(buffer)

		fmt.Println("Sending chunk to chan", chunkSize)

		chunksChan <- fileChunk{
			fromPosition: position,
			chunkNumber:  chunkNumber,
			checksum:     hex.EncodeToString(sha[:]),
			data:         buffer,
		}
		chunkNumber += 1
		position += int64(bytesRead)
	}

	fmt.Println("Closing chan")
	close(chunksChan)
}

func uploadChunks(id string, chunksChan chan fileChunk) {
	fmt.Println("Entered upload chunkss")
	wg := &sync.WaitGroup{}
	for i := 0; i < UPLOAD_ROUTINES; i++ {
		wg.Add(1)
		fmt.Println("Calling upload chunk")
		go uploadChunk(id, chunksChan, wg)
	}
	wg.Wait()
	fmt.Println("left upload chunks")
}

func uploadChunk(id string, chunksChan <-chan fileChunk, wg *sync.WaitGroup) {
	defer wg.Done()
	for chunk := range chunksChan {
		fmt.Printf("Processing chunk")
		body := fileChunkRequestBody{Checksum: chunk.checksum, FromPosition: chunk.fromPosition, ChunkNumber: chunk.chunkNumber}

		marshalledBody, err := json.Marshal(body)
		if err != nil {
			fmt.Print("Error marshalling body ", err)
			panic(err)
		}

		var requestBody bytes.Buffer

		writer := multipart.NewWriter(&requestBody)

		jsonPart, err := writer.CreateFormField("json")
		if err != nil {
			fmt.Print("Error while creating json form field", err)
			panic(err)
		}

		_, err = jsonPart.Write(marshalledBody)
		if err != nil {
			fmt.Print("Error while writing the marshalled body", err)
			panic(err)
		}

		filePart, err := writer.CreateFormFile("file", "chunk")
		if err != nil {
			fmt.Print("Error while creating file form field", err)
			panic(err)
		}

		_, err = filePart.Write(chunk.data)
		if err != nil {
			fmt.Print("Error writing file data to the form", err)
			panic(err)
		}

		writer.Close()

		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/upload/%s/chunk", url, id), &requestBody)

		if err != nil {
			fmt.Print("Error while creating the put request", err)
			panic(err)
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			fmt.Print("Error while sending the chunk request to the server", err)
			panic(err)
		}

		// Check the response status
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("unexpected response status: %s", resp.Status)
			panic(errors.New("received unexpected response from server"))
		}

		resp.Body.Close()
	}
}

func fetchFileChunk(id string, chunkIdChan <-chan int, wg *sync.WaitGroup, chunksChan chan<- fileChunk) {
	defer wg.Done()
	for chunkId := range chunkIdChan {
		response, err := http.Get(fmt.Sprintf("%s/upload/%s/%d", url, id, chunkId))

		if err != nil {
			fmt.Println("Error while downloading a chunk", err.Error(), chunkId)
			panic(err)
		}

		defer response.Body.Close()

		logHeaders(response.Header)

		// Check if response Content-Type is multipart
		contentType := response.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			fmt.Println("Error: Response is not multipart")
			return
		}

		mr := multipart.NewReader(response.Body, contentType[strings.Index(contentType, "boundary=")+len("boundary="):])

		var responseBody getFileChunkResponseBody

		var chunkByte []byte

		for i := 0; i < 2; i++ {
			part, err := mr.NextPart()

			fmt.Println("read part ", i, part.FormName(), chunkId)

			if err == io.EOF {
				break
			}

			if err != nil {
				fmt.Println("Error while reading multipart response", err.Error())
				panic(err)
			}

			if part.FormName() == "file" {
				fmt.Println("AM I ENTERING HRE!???")
				chunkByte, err = io.ReadAll(part)
				if err != nil {
					fmt.Println("error reading the chunk file", err.Error())
					panic(err)
				}
				fmt.Println("Read chunk", len(chunkByte))
			}

			if part.FormName() == "json" {
				err = json.NewDecoder(part).Decode(&responseBody)
				if err != nil {
					fmt.Println("error decoding the chunk response body ", err.Error())
					panic(err)
				}
			}
		}

		// sleepTime := 1000
		// fmt.Println("Sleeping for ", sleepTime)
		// time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		chunksChan <- fileChunk{
			fromPosition: responseBody.FromPosition,
			chunkNumber:  chunkId,
			checksum:     responseBody.Checksum,
			data:         chunkByte,
		}
	}

	fmt.Println("Done fetching chunks")
}

func fetchFileChunks(id string, chunkCount int, chunksChan chan<- fileChunk, doneChan chan<- bool) {
	wg := &sync.WaitGroup{}
	chunkIdChan := make(chan int, chunkCount)

	for i := 0; i < chunkCount; i++ {
		chunkIdChan <- i
	}

	fmt.Printf("posted %d chunks", chunkCount)

	close(chunkIdChan)

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go fetchFileChunk(id, chunkIdChan, wg, chunksChan)
	}

	wg.Wait()

	doneChan <- true

	fmt.Println("AFTER WAITGROUP")
}

func assembleFile(expectedChecksum string, chunksChan <-chan fileChunk, fileHandle *os.File, doneChan chan<- bool) {
	for chunkInfo := range chunksChan {
		_, err := fileHandle.WriteAt(chunkInfo.data, chunkInfo.fromPosition)
		if err != nil {
			fmt.Println("Failed to write chunk", chunkInfo.chunkNumber, chunkInfo.fromPosition, err.Error())
			return
		}
		fmt.Println("Done writing chunk", chunkInfo.chunkNumber, chunkInfo.fromPosition, len(chunkInfo.data))
	}

	fmt.Println("Finished writing chunks")

	h := sha256.New()

	if _, err := io.Copy(h, fileHandle); err != nil {
		fmt.Println("Failed to hash file", err.Error())
		return
	}

	checksum := hex.EncodeToString(h.Sum(nil))

	if expectedChecksum != checksum {
		fmt.Println("Checksums dont match ", expectedChecksum, checksum)
	} else {
		println("Checksums match")
	}

	doneChan <- true
}

func logHeaders(headers http.Header) {
	fmt.Println("Headers:")
	for key, values := range headers {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
}

func main() {
	fileHandle, err := os.Open(FILE_TO_UPLOAD)

	if err != nil {
		fmt.Printf("Failed to open file %s with %s", FILE_TO_UPLOAD, err.Error())
		return
	}

	defer fileHandle.Close()

	h := sha256.New()

	if _, err := io.Copy(h, fileHandle); err != nil {
		fmt.Printf("Failed to hash file %s", err.Error())
		return
	}

	checksum := hex.EncodeToString(h.Sum(nil))

	fileInfo, err := fileHandle.Stat()
	if err != nil {
		fmt.Printf("Failed to read file info %s", err.Error())
		return
	}

	body := createFileRequestBody{Name: FILE_TO_UPLOAD, Size: fileInfo.Size(), Checksum: checksum}

	byteBody, err := json.Marshal(body)

	if err != nil {
		fmt.Printf("Error while marshalling the file %s", err.Error())
		return
	}

	response, err := http.Post(fmt.Sprintf("%s/upload", url), "application/json", bytes.NewBuffer(byteBody))

	if err != nil {
		fmt.Printf("Error while sending the request to create a file %s", err.Error())
		return
	}

	defer response.Body.Close()

	var responseBody createFileResponseBody
	err = json.NewDecoder(response.Body).Decode(&responseBody)

	if err != nil {
		fmt.Printf("Error while parsing create file response %s", err.Error())
		return
	}

	chunksChan := make(chan fileChunk)
	go readFileChunks(FILE_TO_UPLOAD, responseBody.ChunkSize, chunksChan)
	uploadChunks(responseBody.Id, chunksChan)

	// Dowload and assemble

	response, err = http.Get(fmt.Sprintf("%s/upload/%s", url, responseBody.Id))

	if err != nil {
		fmt.Printf("Error while sending the request to get a file %s", err.Error())
		return
	}

	defer response.Body.Close()

	var fileResponseBody getFileResponseBody
	err = json.NewDecoder(response.Body).Decode(&fileResponseBody)
	if err != nil {
		fmt.Printf("Error while parsing get file response %s", err.Error())
		return
	}

	fileHandle, err = os.Create("./1gb_file_fetched")
	if err != nil {
		fmt.Printf("Error creating the fetched file %s", err.Error())
		return
	}

	if err := fileHandle.Truncate(fileResponseBody.Size); err != nil {
		fmt.Printf("Error while setting the size of the file %s", err.Error())
		return
	}

	chunksChan = make(chan fileChunk)
	doneChan := make(chan bool)
	doneAssemblingChan := make(chan bool)
	go fetchFileChunks(fileResponseBody.Id, fileResponseBody.ChunkCount, chunksChan, doneChan)
	go assembleFile(fileResponseBody.Checksum, chunksChan, fileHandle, doneAssemblingChan)

	<-doneChan
	close(chunksChan)
	<-doneAssemblingChan
}
