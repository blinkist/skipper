package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// RelConfigPath is the relative config path to the user's home folder
const RelConfigPath = ".skipper"

// GetPath returns the current path CWD
func GetPath() (*string, error) {
	dir, err := os.Getwd()
	if err != nil {
		ret := ""
		return &ret, err
	}
	return &dir, nil
}

// GetConfigDir return the path to the skipper config directory
func GetConfigDir() *string {
	home, err := UnixHome()
	if err != nil {
		fmt.Println("Cannot determine homedir")
		os.Exit(1)
	}
	path := fmt.Sprintf("%s/%s", home, RelConfigPath)
	return &path
}

// EnsureConfigDir ensure that the skipper config directory exists
func EnsureConfigDir() error {
	path := GetConfigDir()
	var err error
	if _, err = os.Stat(*path); os.IsNotExist(err) {
		return os.Mkdir(*path, 0700)
	}
	return nil
}

// UnixHome returns the Unix home dir of the executing skipper user
func UnixHome() (string, error) {
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if err != exec.ErrNotFound {
			return "", err
		}
	} else {
		if pw := strings.TrimSpace(out.String()); pw != "" {
			parts := strings.SplitN(pw, ":", 7)
			if len(parts) > 5 {
				return parts[5], nil
			}
		}
	}
	out.Reset()

	cmd = exec.Command("sh", "-c", "cd && pwd")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	res := strings.TrimSpace(out.String())
	if res != "" {
		return res, nil
	}

	return "", errors.New("cannot find a home directory for current user")
}
