package profiles

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"path/filepath"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var (
	ErrNoPicture       = errors.New("profiles: no such picture among the crosshairs")
	ErrPictureNotSaved = errors.New("profiles: the picture was not saved")
)

const pictureFolder = "crosshairs"

// AddPicture reads the file the way the overlay will before a byte of it reaches the disk, so a
// picture on the disk is one that can be drawn.
func (s Store) AddPicture(file []byte) (string, error) {
	if _, err := crosshair.FromPNG(file, 100); err != nil {
		return "", err
	}

	sum := sha256.Sum256(file)
	name := hex.EncodeToString(sum[:]) + ".png"
	folder := filepath.Join(s.Dir, pictureFolder)
	if err := s.Disk.MkdirAll(folder); err != nil {
		return "", fmt.Errorf("%w: %w", ErrPictureNotSaved, err)
	}

	path := filepath.Join(folder, name)
	if err := s.replace(path, file); err != nil {
		return "", fmt.Errorf("%w: %w", ErrPictureNotSaved, errors.Join(err, s.discard(path+".tmp")))
	}

	return name, nil
}

func (s Store) Picture(name string) ([]byte, error) {
	if !pictureName.MatchString(name) {
		return nil, fmt.Errorf("%w: %q", ErrNoPicture, name)
	}

	file, err := s.Disk.ReadFile(filepath.Join(s.Dir, pictureFolder, name))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoPicture, err)
	}

	return file, nil
}

func (s Store) Paint(c Crosshair) (*image.RGBA, error) {
	if c.Image == "" {
		return crosshair.Draw(c.Style)
	}

	file, err := s.Picture(c.Image)
	if err != nil {
		return nil, err
	}

	return crosshair.FromPNG(file, c.Scale)
}
