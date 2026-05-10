package desktop

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidDialog  = errors.New("invalid dialog options")
	ErrInvalidShell   = errors.New("invalid shell request")
	ErrClipboardEmpty = errors.New("clipboard format not available")
	ErrInvalidSource  = errors.New("invalid desktop capturer source")
)

type OpenDialogOptions struct {
	Title       string
	DefaultPath string
	Properties  []string
	Filters     []FileFilter
}

type FileFilter struct {
	Name       string
	Extensions []string
}

type OpenDialogResult struct {
	Canceled  bool
	FilePaths []string
}

func NormalizeOpenDialog(options OpenDialogOptions) (OpenDialogOptions, error) {
	options.Title = strings.TrimSpace(options.Title)
	options.DefaultPath = strings.TrimSpace(options.DefaultPath)
	if options.DefaultPath != "" && !filepath.IsAbs(options.DefaultPath) {
		return OpenDialogOptions{}, fmt.Errorf("%w: defaultPath must be absolute", ErrInvalidDialog)
	}
	for _, property := range options.Properties {
		switch property {
		case "openFile", "openDirectory", "multiSelections", "showHiddenFiles", "createDirectory", "promptToCreate":
		default:
			return OpenDialogOptions{}, fmt.Errorf("%w: property %s", ErrInvalidDialog, property)
		}
	}
	for i, filter := range options.Filters {
		filter.Name = strings.TrimSpace(filter.Name)
		if filter.Name == "" || len(filter.Extensions) == 0 {
			return OpenDialogOptions{}, fmt.Errorf("%w: file filter", ErrInvalidDialog)
		}
		for _, ext := range filter.Extensions {
			if strings.TrimSpace(ext) == "" || strings.Contains(ext, ".") {
				return OpenDialogOptions{}, fmt.Errorf("%w: extension", ErrInvalidDialog)
			}
		}
		options.Filters[i] = filter
	}
	return options, nil
}

type Shell struct {
	openedExternal []string
	shownItems     []string
	trashedItems   []string
}

func (s *Shell) OpenExternal(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("%w: external URL", ErrInvalidShell)
	}
	s.openedExternal = append(s.openedExternal, parsed.String())
	return nil
}

func (s *Shell) OpenPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return "Path must be absolute", fmt.Errorf("%w: open path", ErrInvalidShell)
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "Path does not exist", nil
		}
		return err.Error(), nil
	}
	return "", nil
}

func (s *Shell) ShowItemInFolder(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return fmt.Errorf("%w: item path", ErrInvalidShell)
	}
	s.shownItems = append(s.shownItems, path)
	return nil
}

func (s *Shell) TrashItem(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return fmt.Errorf("%w: trash path", ErrInvalidShell)
	}
	s.trashedItems = append(s.trashedItems, path)
	return nil
}

func (s *Shell) Events() []string {
	events := make([]string, 0, len(s.openedExternal)+len(s.shownItems)+len(s.trashedItems))
	for _, value := range s.openedExternal {
		events = append(events, "openExternal:"+value)
	}
	for _, value := range s.shownItems {
		events = append(events, "showItemInFolder:"+value)
	}
	for _, value := range s.trashedItems {
		events = append(events, "trashItem:"+value)
	}
	return events
}

type Clipboard struct {
	text    string
	html    string
	rtf     string
	image   []byte
	buffers map[string][]byte
}

func (c *Clipboard) WriteText(text string) {
	c.text = text
}

func (c *Clipboard) ReadText() (string, error) {
	if c.text == "" {
		return "", ErrClipboardEmpty
	}
	return c.text, nil
}

func (c *Clipboard) WriteHTML(html string) {
	c.html = html
}

func (c *Clipboard) ReadHTML() (string, error) {
	if c.html == "" {
		return "", ErrClipboardEmpty
	}
	return c.html, nil
}

func (c *Clipboard) WriteRTF(rtf string) {
	c.rtf = rtf
}

func (c *Clipboard) ReadRTF() (string, error) {
	if c.rtf == "" {
		return "", ErrClipboardEmpty
	}
	return c.rtf, nil
}

func (c *Clipboard) WriteImage(image []byte) {
	c.image = append([]byte(nil), image...)
}

func (c *Clipboard) ReadImage() ([]byte, error) {
	if len(c.image) == 0 {
		return nil, ErrClipboardEmpty
	}
	return append([]byte(nil), c.image...), nil
}

func (c *Clipboard) WriteBuffer(format string, data []byte) error {
	format = strings.TrimSpace(format)
	if format == "" {
		return fmt.Errorf("%w: buffer format", ErrInvalidDialog)
	}
	if c.buffers == nil {
		c.buffers = make(map[string][]byte)
	}
	c.buffers[format] = append([]byte(nil), data...)
	return nil
}

func (c *Clipboard) ReadBuffer(format string) ([]byte, error) {
	format = strings.TrimSpace(format)
	if format == "" {
		return nil, fmt.Errorf("%w: buffer format", ErrInvalidDialog)
	}
	data, ok := c.buffers[format]
	if !ok {
		return nil, ErrClipboardEmpty
	}
	return append([]byte(nil), data...), nil
}

type SourceType string

const (
	SourceScreen SourceType = "screen"
	SourceWindow SourceType = "window"
)

type Source struct {
	ID        string
	Name      string
	Type      SourceType
	Thumbnail []byte
}

type Capturer struct {
	sources []Source
}

func NewCapturer(sources []Source) (*Capturer, error) {
	normalized := make([]Source, len(sources))
	for i, source := range sources {
		source.ID = strings.TrimSpace(source.ID)
		source.Name = strings.TrimSpace(source.Name)
		if source.ID == "" || source.Name == "" {
			return nil, fmt.Errorf("%w: id/name", ErrInvalidSource)
		}
		if source.Type != SourceScreen && source.Type != SourceWindow {
			return nil, fmt.Errorf("%w: type", ErrInvalidSource)
		}
		source.Thumbnail = append([]byte(nil), source.Thumbnail...)
		normalized[i] = source
	}
	return &Capturer{sources: normalized}, nil
}

func (c *Capturer) GetSources(types ...SourceType) []Source {
	want := make(map[SourceType]bool, len(types))
	for _, typ := range types {
		want[typ] = true
	}
	out := make([]Source, 0, len(c.sources))
	for _, source := range c.sources {
		if len(want) > 0 && !want[source.Type] {
			continue
		}
		source.Thumbnail = append([]byte(nil), source.Thumbnail...)
		out = append(out, source)
	}
	return out
}
