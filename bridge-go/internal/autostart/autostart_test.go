package autostart

import (
	"errors"
	"testing"
)

type memoryStore struct {
	value string
	err   error
}

func (s *memoryStore) Read() (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.value == "" {
		return "", ErrNotFound
	}
	return s.value, nil
}

func (s *memoryStore) Write(value string) error {
	if s.err != nil {
		return s.err
	}
	s.value = value
	return nil
}

func (s *memoryStore) Delete() error {
	if s.err != nil {
		return s.err
	}
	s.value = ""
	return nil
}

func Test_req_BRG_09_AutostartCanBeEnabledAndDisabled(t *testing.T) {
	store := &memoryStore{}
	executable := `C:\Users\Cody\AppData\Local\Programs\Cody\cody.exe`

	enabled, err := Enabled(store, executable)
	if err != nil || enabled {
		t.Fatalf("initial Enabled() = %t, %v; want false, nil", enabled, err)
	}
	if err := SetEnabled(store, executable, true); err != nil {
		t.Fatal(err)
	}
	if got, want := store.value, `"`+executable+`"`; got != want {
		t.Fatalf("stored Run command = %q, want %q", got, want)
	}
	enabled, err = Enabled(store, executable)
	if err != nil || !enabled {
		t.Fatalf("Enabled() = %t, %v; want true, nil", enabled, err)
	}
	if err := SetEnabled(store, executable, false); err != nil {
		t.Fatal(err)
	}
	enabled, err = Enabled(store, executable)
	if err != nil || enabled {
		t.Fatalf("Enabled() after disable = %t, %v; want false, nil", enabled, err)
	}
}

func Test_req_BRG_16_AutostartValueMustPointToThisApp(t *testing.T) {
	store := &memoryStore{value: `"C:\Programs\OldCody.exe"`}
	enabled, err := Enabled(store, `C:\Programs\Cody.exe`)
	if err != nil || enabled {
		t.Fatalf("Enabled() for a different executable = %t, %v; want false, nil", enabled, err)
	}
	store.value = `"c:\PROGRAMS\CODY.EXE"`
	enabled, err = Enabled(store, `C:\Programs\Cody.exe`)
	if err != nil || !enabled {
		t.Fatalf("Enabled() for case-insensitive Windows path = %t, %v; want true, nil", enabled, err)
	}
}

func Test_req_BRG_09_AutostartStoreErrorsAreReturned(t *testing.T) {
	want := errors.New("registry unavailable")
	store := &memoryStore{err: want}
	if _, err := Enabled(store, `C:\Programs\Cody.exe`); !errors.Is(err, want) {
		t.Fatalf("Enabled() error = %v, want %v", err, want)
	}
	if err := SetEnabled(store, `C:\Programs\Cody.exe`, true); !errors.Is(err, want) {
		t.Fatalf("SetEnabled() error = %v, want %v", err, want)
	}
}
