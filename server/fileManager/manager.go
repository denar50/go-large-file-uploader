package fileManager

import (
	"fmt"
	"math"
	"os"

	"github.com/google/uuid"
)

type FileManager struct {
	files map[string]fileInfo
}

type AllocateFileResult struct {
	Id         string
	ChunkSize  int
	ChunkCount int
}

type GetFileResult struct {
	Id         string
	Checksum   string
	Size       int64
	ChunkCount int
}

type GetFileChunkResult struct {
	Checksum          string
	ChunkFileLocation string
	FromPosition      int64
}

type fileInfo struct {
	name     string
	checksum string
	size     int64
	chunks   []chunkInfo
}

type chunkInfo struct {
	startPosition int64
	fileLocation  string
	checksum      string
}

// Errors
type FileNotFoundError struct {
	id string
}

type ChunkNotSavedError struct {
	message string
}

type ChunkOverflowError struct {
	position    int
	maxPosition int
}

type ChunkFileError struct {
	message string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("File %s not found", e.id)
}

func (e *ChunkNotSavedError) Error() string {
	return fmt.Sprintf("Chunk not saved error: %s", e.message)
}

func (e *ChunkOverflowError) Error() string {
	return fmt.Sprintf("Chunk overflow error: position %d is bigger than max position %d", e.position, e.maxPosition)
}

func (e *ChunkFileError) Error() string {
	return fmt.Sprintf("Chunk file errored with %s", e.message)
}

const FILES_DIR = "~/Projects/large-file-uploader/files"
const CHUNK_SIZE = 1_000

var Manager = &FileManager{
	files: make(map[string]fileInfo),
}

func (m *FileManager) AllocateFile(name string, size int64, checksum string) (AllocateFileResult, error) {
	id := uuid.New().String()
	chunkCount := int(math.Ceil(float64(size) / float64(CHUNK_SIZE)))
	fmt.Println(float64(size) / float64(CHUNK_SIZE))
	fmt.Println(chunkCount)
	chunks := make([]chunkInfo, chunkCount)
	m.files[id] = fileInfo{
		name,
		checksum,
		size,
		chunks,
	}

	fmt.Println("Allocated file ", id)

	return AllocateFileResult{Id: id, ChunkSize: CHUNK_SIZE, ChunkCount: int(chunkCount)}, nil
}

func (m *FileManager) ProcessChunk(id string, chunkNumber int, fromPosition int64, checksum string, chunk []byte) error {
	fileInfo, found := m.files[id]

	if !found {
		return &FileNotFoundError{id}
	}

	if len(fileInfo.chunks)-1 < chunkNumber {
		return &ChunkOverflowError{maxPosition: len(fileInfo.chunks) - 1, position: chunkNumber}
	}

	// Ensure the dir exists
	chunksDir := fmt.Sprintf("./files/%s", id)
	os.Mkdir(chunksDir, 0700)

	chunkFileName := fmt.Sprintf("%s/%s", chunksDir, uuid.New().String())

	file, err := os.Create(chunkFileName)

	if err != nil {
		return &ChunkNotSavedError{err.Error()}
	}

	defer file.Close()

	_, err = file.Write(chunk)

	if err != nil {
		return &ChunkNotSavedError{err.Error()}
	}

	fileInfo.chunks[chunkNumber] = chunkInfo{startPosition: fromPosition, fileLocation: chunkFileName, checksum: checksum}

	return nil
}

func (m *FileManager) GetFileInfo(id string) (GetFileResult, error) {
	fileInfo, found := m.files[id]
	if !found {
		return GetFileResult{}, &FileNotFoundError{id}
	}

	// TODO: Verify all chunks have been uploaded

	return GetFileResult{Id: id, Checksum: fileInfo.checksum, Size: fileInfo.size, ChunkCount: len(fileInfo.chunks)}, nil
}

func (m *FileManager) GetFileChunk(id string, chunkNumber int) (GetFileChunkResult, error) {
	fileInfo, found := m.files[id]
	if !found {
		return GetFileChunkResult{}, &FileNotFoundError{id}
	}

	// TODO: Chunk overflow error here

	chunkInfo := fileInfo.chunks[chunkNumber]

	return GetFileChunkResult{Checksum: chunkInfo.checksum, ChunkFileLocation: chunkInfo.fileLocation, FromPosition: chunkInfo.startPosition}, nil
}
