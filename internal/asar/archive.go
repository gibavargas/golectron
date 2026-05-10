package asar

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrInvalidPath   = errors.New("invalid ASAR path")
	ErrEntryNotFound = errors.New("ASAR entry not found")
	ErrEntryUnpacked = errors.New("ASAR entry is unpacked")
)

type Entry struct {
	Path     string
	Offset   int64
	Size     int64
	Mode     uint32
	Dir      bool
	Unpacked bool
}

type Archive struct {
	Path        string
	headerSize  int64
	contentBase int64
	entries     map[string]Entry
}

type SplitPath struct {
	ArchivePath  string
	InnerPath    string
	UnpackedPath string
}

func NewArchive(path string, entries []Entry) (*Archive, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || !strings.HasSuffix(path, ".asar") {
		return nil, fmt.Errorf("%w: archive path", ErrInvalidPath)
	}
	archive := &Archive{Path: path, contentBase: 0, entries: make(map[string]Entry, len(entries))}
	for _, entry := range entries {
		normalized, err := normalizeEntry(entry)
		if err != nil {
			return nil, err
		}
		archive.entries[normalized.Path] = normalized
	}
	return archive, nil
}

func Open(path string) (*Archive, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || !strings.HasSuffix(path, ".asar") {
		return nil, fmt.Errorf("%w: archive path", ErrInvalidPath)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < 8 {
		return nil, fmt.Errorf("%w: header", ErrInvalidPath)
	}
	var sizePickle [8]byte
	if _, err := io.ReadFull(file, sizePickle[:]); err != nil {
		return nil, err
	}
	headerSize, err := readPickleUint32(sizePickle[:])
	if err != nil {
		return nil, err
	}
	if headerSize == 0 || int64(headerSize) > info.Size()-8 {
		return nil, fmt.Errorf("%w: header size", ErrInvalidPath)
	}
	headerPickle := make([]byte, headerSize)
	if _, err := io.ReadFull(file, headerPickle); err != nil {
		return nil, err
	}
	headerJSON, err := readPickleString(headerPickle)
	if err != nil {
		return nil, err
	}
	entries, err := parseHeaderEntries([]byte(headerJSON))
	if err != nil {
		return nil, err
	}
	archive := &Archive{
		Path:        path,
		headerSize:  int64(headerSize),
		contentBase: 8 + int64(headerSize),
		entries:     make(map[string]Entry, len(entries)),
	}
	for _, entry := range entries {
		normalized, err := normalizeEntry(entry)
		if err != nil {
			return nil, err
		}
		archive.entries[normalized.Path] = normalized
	}
	return archive, nil
}

func Write(path string, packed map[string][]byte, unpacked map[string][]byte) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || !strings.HasSuffix(path, ".asar") {
		return fmt.Errorf("%w: archive path", ErrInvalidPath)
	}
	root := &headerWriteNode{Files: map[string]*headerWriteNode{}}
	var content []byte
	for _, name := range sortedByteMapKeys(packed) {
		data := packed[name]
		if err := addHeaderWriteFile(root, name, &headerWriteNode{Offset: strconv.Itoa(len(content)), Size: int64(len(data))}); err != nil {
			return err
		}
		content = append(content, data...)
	}
	for _, name := range sortedByteMapKeys(unpacked) {
		data := unpacked[name]
		if err := addHeaderWriteFile(root, name, &headerWriteNode{Size: int64(len(data)), Unpacked: true}); err != nil {
			return err
		}
		target := filepath.Join(path+".unpacked", filepath.FromSlash(normalizeInnerPath(name)))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	headerJSON, err := json.Marshal(root)
	if err != nil {
		return err
	}
	headerPickle := writePickleString(string(headerJSON))
	sizePickle := writePickleUint32(uint32(len(headerPickle)))
	data := append(sizePickle, headerPickle...)
	data = append(data, content...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Split(path string) (SplitPath, bool) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	parts := strings.Split(cleaned, string(filepath.Separator))
	for i, part := range parts {
		if strings.HasSuffix(part, ".asar") {
			archivePath := filepath.Join(parts[:i+1]...)
			inner := filepath.Join(parts[i+1:]...)
			if inner == "." {
				inner = ""
			}
			return SplitPath{
				ArchivePath:  archivePath,
				InnerPath:    filepath.ToSlash(inner),
				UnpackedPath: archivePath + ".unpacked" + string(filepath.Separator) + filepath.Join(parts[i+1:]...),
			}, true
		}
	}
	return SplitPath{}, false
}

func (a *Archive) Stat(innerPath string) (Entry, error) {
	key := normalizeInnerPath(innerPath)
	entry, ok := a.entries[key]
	if !ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrEntryNotFound, innerPath)
	}
	return entry, nil
}

func (a *Archive) ResolveModule(specifier string) (Entry, error) {
	specifier = normalizeInnerPath(specifier)
	candidates := []string{
		specifier,
		specifier + ".js",
		specifier + ".json",
		filepath.ToSlash(filepath.Join(specifier, "index.js")),
	}
	for _, candidate := range candidates {
		entry, ok := a.entries[candidate]
		if !ok || entry.Dir {
			continue
		}
		if entry.Unpacked {
			return Entry{}, fmt.Errorf("%w: %s", ErrEntryUnpacked, candidate)
		}
		return entry, nil
	}
	return Entry{}, fmt.Errorf("%w: %s", ErrEntryNotFound, specifier)
}

func (a *Archive) CopyFileSource(innerPath string) (string, error) {
	entry, err := a.Stat(innerPath)
	if err != nil {
		return "", err
	}
	if entry.Dir {
		return "", fmt.Errorf("%w: directory", ErrInvalidPath)
	}
	if entry.Unpacked {
		return filepath.Join(a.Path+".unpacked", filepath.FromSlash(entry.Path)), nil
	}
	return a.Path + string(filepath.Separator) + filepath.FromSlash(entry.Path), nil
}

func (a *Archive) ReadFile(innerPath string) ([]byte, error) {
	entry, err := a.Stat(innerPath)
	if err != nil {
		return nil, err
	}
	if entry.Dir {
		return nil, fmt.Errorf("%w: directory", ErrInvalidPath)
	}
	if entry.Unpacked {
		return os.ReadFile(filepath.Join(a.Path+".unpacked", filepath.FromSlash(entry.Path)))
	}
	if entry.Offset < 0 {
		return nil, fmt.Errorf("%w: offset", ErrInvalidPath)
	}
	file, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data := make([]byte, entry.Size)
	if _, err := file.ReadAt(data, a.contentBase+entry.Offset); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return data, nil
}

func normalizeEntry(entry Entry) (Entry, error) {
	entry.Path = normalizeInnerPath(entry.Path)
	if entry.Path == "" || strings.HasPrefix(entry.Path, "../") {
		return Entry{}, fmt.Errorf("%w: entry path", ErrInvalidPath)
	}
	if entry.Size < 0 {
		return Entry{}, fmt.Errorf("%w: size", ErrInvalidPath)
	}
	if entry.Offset < 0 {
		return Entry{}, fmt.Errorf("%w: offset", ErrInvalidPath)
	}
	return entry, nil
}

func normalizeInnerPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "." {
		return ""
	}
	return path
}

func readPickleUint32(buffer []byte) (uint32, error) {
	if len(buffer) < 8 {
		return 0, fmt.Errorf("%w: pickle", ErrInvalidPath)
	}
	payloadSize := binary.LittleEndian.Uint32(buffer[:4])
	if payloadSize < 4 || int(payloadSize) > len(buffer)-4 {
		return 0, fmt.Errorf("%w: pickle payload", ErrInvalidPath)
	}
	return binary.LittleEndian.Uint32(buffer[4:8]), nil
}

func readPickleString(buffer []byte) (string, error) {
	if len(buffer) < 8 {
		return "", fmt.Errorf("%w: pickle", ErrInvalidPath)
	}
	payloadSize := binary.LittleEndian.Uint32(buffer[:4])
	if payloadSize < 4 || int(payloadSize) > len(buffer)-4 {
		return "", fmt.Errorf("%w: pickle payload", ErrInvalidPath)
	}
	length := int(binary.LittleEndian.Uint32(buffer[4:8]))
	if length < 0 || length > int(payloadSize)-4 || 8+length > len(buffer) {
		return "", fmt.Errorf("%w: pickle string", ErrInvalidPath)
	}
	return string(buffer[8 : 8+length]), nil
}

func writePickleUint32(value uint32) []byte {
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint32(buffer[:4], 4)
	binary.LittleEndian.PutUint32(buffer[4:8], value)
	return buffer
}

func writePickleString(value string) []byte {
	payloadSize := align4(4 + len(value))
	buffer := make([]byte, 4+payloadSize)
	binary.LittleEndian.PutUint32(buffer[:4], uint32(payloadSize))
	binary.LittleEndian.PutUint32(buffer[4:8], uint32(len(value)))
	copy(buffer[8:], value)
	return buffer
}

func align4(n int) int {
	if n%4 == 0 {
		return n
	}
	return n + (4 - n%4)
}

type headerRoot struct {
	Files map[string]headerEntry `json:"files"`
}

type headerWriteNode struct {
	Files    map[string]*headerWriteNode `json:"files,omitempty"`
	Offset   string                      `json:"offset,omitempty"`
	Size     int64                       `json:"size,omitempty"`
	Unpacked bool                        `json:"unpacked,omitempty"`
}

type headerEntry struct {
	Files    map[string]headerEntry `json:"files,omitempty"`
	Offset   string                 `json:"offset,omitempty"`
	Size     *int64                 `json:"size,omitempty"`
	Mode     *uint32                `json:"mode,omitempty"`
	Unpacked bool                   `json:"unpacked,omitempty"`
}

func parseHeaderEntries(data []byte) ([]Entry, error) {
	var root headerRoot
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("%w: header JSON", ErrInvalidPath)
	}
	if root.Files == nil {
		return nil, fmt.Errorf("%w: header files", ErrInvalidPath)
	}
	var entries []Entry
	for name, child := range root.Files {
		if err := appendHeaderEntry(&entries, name, child); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func appendHeaderEntry(entries *[]Entry, path string, entry headerEntry) error {
	path = normalizeInnerPath(path)
	if path == "" {
		return fmt.Errorf("%w: header entry path", ErrInvalidPath)
	}
	if entry.Files != nil {
		*entries = append(*entries, Entry{Path: path, Dir: true})
		for name, child := range entry.Files {
			if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
				return fmt.Errorf("%w: header entry name", ErrInvalidPath)
			}
			if err := appendHeaderEntry(entries, filepath.ToSlash(filepath.Join(path, name)), child); err != nil {
				return err
			}
		}
		return nil
	}
	if entry.Size == nil {
		return fmt.Errorf("%w: header entry size", ErrInvalidPath)
	}
	out := Entry{Path: path, Size: *entry.Size, Unpacked: entry.Unpacked}
	if entry.Mode != nil {
		out.Mode = *entry.Mode
	}
	if !entry.Unpacked {
		if entry.Offset == "" {
			return fmt.Errorf("%w: header entry offset", ErrInvalidPath)
		}
		offset, err := strconv.ParseInt(entry.Offset, 10, 64)
		if err != nil || offset < 0 {
			return fmt.Errorf("%w: header entry offset", ErrInvalidPath)
		}
		out.Offset = offset
	}
	*entries = append(*entries, out)
	return nil
}

func addHeaderWriteFile(root *headerWriteNode, name string, file *headerWriteNode) error {
	normalized := normalizeInnerPath(name)
	if normalized == "" || strings.HasPrefix(normalized, "../") {
		return fmt.Errorf("%w: header entry path", ErrInvalidPath)
	}
	current := root
	parts := strings.Split(normalized, "/")
	for i, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, `/\`) {
			return fmt.Errorf("%w: header entry name", ErrInvalidPath)
		}
		if i == len(parts)-1 {
			current.Files[part] = file
			return nil
		}
		if current.Files[part] == nil {
			current.Files[part] = &headerWriteNode{Files: map[string]*headerWriteNode{}}
		}
		current = current.Files[part]
	}
	return nil
}

func sortedByteMapKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
