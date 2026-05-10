package apppaths

import (
	"errors"
	"fmt"
)

var ErrInvalidBadgeCount = errors.New("invalid badge count")

type BadgeState struct {
	Supported bool
	Count     int
	Visible   bool
	Unknown   bool
}

func (s *Store) SetBadgeCount(count *int) (bool, error) {
	if !s.supportsBadgeCount() {
		return false, nil
	}
	if count == nil {
		if s.env.GOOS == "darwin" {
			s.badge = BadgeState{Supported: true, Count: 0, Visible: true, Unknown: true}
			return true, nil
		}
		s.badge = BadgeState{Supported: true}
		return true, nil
	}
	if *count < 0 {
		return false, fmt.Errorf("%w: %d", ErrInvalidBadgeCount, *count)
	}
	s.badge = BadgeState{
		Supported: true,
		Count:     *count,
		Visible:   *count > 0,
		Unknown:   false,
	}
	return true, nil
}

func (s *Store) GetBadgeCount() int {
	return s.badge.Count
}

func (s *Store) BadgeState() BadgeState {
	state := s.badge
	state.Supported = s.supportsBadgeCount()
	return state
}

func (s *Store) IsUnityRunning() bool {
	return s.env.GOOS == "linux" && s.env.UnityRunning
}

func (s *Store) supportsBadgeCount() bool {
	return s.env.GOOS == "darwin" || (s.env.GOOS == "linux" && s.env.UnityRunning)
}
