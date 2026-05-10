package apppaths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	NameHome       = "home"
	NameAppData    = "appData"
	NameUserData   = "userData"
	NameSession    = "sessionData"
	NameTemp       = "temp"
	NameExe        = "exe"
	NameModule     = "module"
	NameDesktop    = "desktop"
	NameDocuments  = "documents"
	NameDownloads  = "downloads"
	NameMusic      = "music"
	NamePictures   = "pictures"
	NameVideos     = "videos"
	NameRecent     = "recent"
	NameLogs       = "logs"
	NameCrashDumps = "crashDumps"
	NameAssets     = "assets"
)

var (
	ErrUnknownPathName = errors.New("unknown app path name")
	ErrPathRequired    = errors.New("path is required")
	ErrPathNotAbsolute = errors.New("path must be absolute")
	ErrPathMissing     = errors.New("path does not exist")
	ErrUnsupported     = errors.New("unsupported on this platform")
)

type Environment struct {
	GOOS              string
	Home              string
	Temp              string
	Executable        string
	Env               map[string]string
	DeclaredProtocols map[string]struct{}
	WasOpenedAtLogin  bool
	WasOpenedAsHidden bool
	RestoreState      bool
	UnityRunning      bool
}

type Store struct {
	appName    string
	appPath    string
	env        Environment
	overrides  map[string]string
	recents    []string
	protocols  map[string]ProtocolRegistration
	loginItems map[string]loginItemRecord
	badge      BadgeState
}

func NewStore(appName, appPath string, env Environment) *Store {
	if env.GOOS == "" {
		env.GOOS = runtime.GOOS
	}
	if env.Home == "" {
		if home, err := os.UserHomeDir(); err == nil {
			env.Home = home
		}
	}
	if env.Temp == "" {
		env.Temp = os.TempDir()
	}
	if env.Executable == "" {
		if exe, err := os.Executable(); err == nil {
			env.Executable = exe
		}
	}
	if appName == "" {
		appName = "Electron"
	}
	return &Store{
		appName:    sanitizeName(appName),
		appPath:    appPath,
		env:        env,
		overrides:  make(map[string]string),
		protocols:  make(map[string]ProtocolRegistration),
		loginItems: make(map[string]loginItemRecord),
	}
}

func (s *Store) GetAppPath() string {
	return s.appPath
}

func (s *Store) GetPath(name string) (string, error) {
	if value, ok := s.overrides[name]; ok {
		return value, nil
	}
	value, err := s.defaultPath(name)
	if err != nil {
		return "", err
	}
	if name == NameLogs {
		s.overrides[NameLogs] = value
	}
	return value, nil
}

func (s *Store) SetPath(name, value string) error {
	if !isKnownPathName(name) {
		return fmt.Errorf("%w: %s", ErrUnknownPathName, name)
	}
	if strings.TrimSpace(value) == "" {
		return ErrPathRequired
	}
	if !s.isAbsPath(value) {
		return fmt.Errorf("%w: %s", ErrPathNotAbsolute, value)
	}
	info, err := os.Stat(value)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrPathMissing, value)
		}
		return err
	}
	if !info.IsDir() && name != NameExe && name != NameModule {
		return fmt.Errorf("%w: %s", ErrPathMissing, value)
	}
	s.overrides[name] = value
	return nil
}

func (s *Store) SetAppLogsPath(value string) error {
	if strings.TrimSpace(value) == "" {
		value = s.defaultLogsPath()
	}
	if err := os.MkdirAll(value, 0o755); err != nil {
		return err
	}
	s.overrides[NameLogs] = value
	return nil
}

func (s *Store) AddRecentDocument(path string) error {
	if !s.supportsRecentDocuments() {
		return fmt.Errorf("%w: recent documents", ErrUnsupported)
	}
	if strings.TrimSpace(path) == "" {
		return ErrPathRequired
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: %s", ErrPathNotAbsolute, path)
	}

	recents := []string{path}
	for _, existing := range s.recents {
		if existing != path {
			recents = append(recents, existing)
		}
	}
	s.recents = recents
	return nil
}

func (s *Store) ClearRecentDocuments() error {
	if !s.supportsRecentDocuments() {
		return fmt.Errorf("%w: recent documents", ErrUnsupported)
	}
	s.recents = nil
	return nil
}

func (s *Store) GetRecentDocuments() ([]string, error) {
	if !s.supportsRecentDocuments() {
		return nil, fmt.Errorf("%w: recent documents", ErrUnsupported)
	}
	return append([]string(nil), s.recents...), nil
}

func (s *Store) defaultPath(name string) (string, error) {
	switch name {
	case NameHome:
		return s.env.Home, nil
	case NameAppData:
		return s.defaultAppData(), nil
	case NameUserData:
		appData, err := s.GetPath(NameAppData)
		if err != nil {
			return "", err
		}
		return filepath.Join(appData, s.appName), nil
	case NameSession:
		return s.GetPath(NameUserData)
	case NameTemp:
		return s.env.Temp, nil
	case NameExe:
		return s.env.Executable, nil
	case NameModule:
		return s.env.Executable, nil
	case NameDesktop:
		return filepath.Join(s.env.Home, "Desktop"), nil
	case NameDocuments:
		return filepath.Join(s.env.Home, "Documents"), nil
	case NameDownloads:
		return filepath.Join(s.env.Home, "Downloads"), nil
	case NameMusic:
		return filepath.Join(s.env.Home, "Music"), nil
	case NamePictures:
		return filepath.Join(s.env.Home, "Pictures"), nil
	case NameVideos:
		return filepath.Join(s.env.Home, "Videos"), nil
	case NameRecent:
		if s.env.GOOS != "windows" {
			return "", fmt.Errorf("%w: %s", ErrUnknownPathName, name)
		}
		return filepath.Join(s.defaultAppData(), "Microsoft", "Windows", "Recent"), nil
	case NameLogs:
		if err := s.SetAppLogsPath(""); err != nil {
			return "", err
		}
		return s.overrides[NameLogs], nil
	case NameCrashDumps:
		userData, err := s.GetPath(NameUserData)
		if err != nil {
			return "", err
		}
		return filepath.Join(userData, "Crashpad"), nil
	case NameAssets:
		if s.env.GOOS == "darwin" {
			return "", fmt.Errorf("%w: %s", ErrUnknownPathName, name)
		}
		return filepath.Dir(s.env.Executable), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownPathName, name)
	}
}

func (s *Store) defaultAppData() string {
	switch s.env.GOOS {
	case "windows":
		if value := s.getenv("APPDATA"); value != "" {
			return value
		}
		return filepath.Join(s.env.Home, "AppData", "Roaming")
	case "linux":
		if value := s.getenv("XDG_CONFIG_HOME"); value != "" {
			return value
		}
		return filepath.Join(s.env.Home, ".config")
	case "darwin":
		return filepath.Join(s.env.Home, "Library", "Application Support")
	default:
		return filepath.Join(s.env.Home, ".config")
	}
}

func (s *Store) defaultLogsPath() string {
	if s.env.GOOS == "darwin" {
		return filepath.Join(s.env.Home, "Library", "Logs", s.appName)
	}
	userData, err := s.GetPath(NameUserData)
	if err != nil {
		return filepath.Join(s.env.Temp, s.appName, "logs")
	}
	return filepath.Join(userData, "logs")
}

func (s *Store) getenv(key string) string {
	if s.env.Env != nil {
		return s.env.Env[key]
	}
	return os.Getenv(key)
}

func (s *Store) supportsRecentDocuments() bool {
	return s.env.GOOS == "darwin" || s.env.GOOS == "windows"
}

func (s *Store) isAbsPath(path string) bool {
	if s.env.GOOS == "windows" {
		if len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
			return true
		}
		return strings.HasPrefix(path, `\\`)
	}
	return filepath.IsAbs(path)
}

func isKnownPathName(name string) bool {
	switch name {
	case NameHome, NameAppData, NameUserData, NameSession, NameTemp, NameExe, NameModule,
		NameDesktop, NameDocuments, NameDownloads, NameMusic, NamePictures, NameVideos,
		NameRecent, NameLogs, NameCrashDumps, NameAssets:
		return true
	default:
		return false
	}
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Electron"
	}
	return name
}
