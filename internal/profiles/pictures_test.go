package profiles_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var pictures = filepath.Join(folder, "crosshairs")

func pngOf(t *testing.T, side int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	for i := 0; i < side; i++ {
		img.SetNRGBA(i, i, color.NRGBA{R: 255, A: 200})
	}

	var file bytes.Buffer
	if err := png.Encode(&file, img); err != nil {
		t.Fatal(err)
	}

	return file.Bytes()
}

func hashName(file []byte) string {
	sum := sha256.Sum256(file)

	return hex.EncodeToString(sum[:]) + ".png"
}

func TestAPictureIsKeptUnderTheHashOfItsBytes(t *testing.T) {
	disk := newMemory(map[string]string{})
	file := pngOf(t, 8)

	name, err := profiles.Store{Disk: disk, Dir: folder}.AddPicture(file)
	if err != nil {
		t.Fatal(err)
	}

	if name != hashName(file) {
		t.Errorf("named %s, want %s", name, hashName(file))
	}
	if want := map[string]string{filepath.Join(pictures, name): string(file)}; !reflect.DeepEqual(disk.files, want) {
		t.Errorf("files %v, want the picture alone in crosshairs", keys(disk.files))
	}
}

func TestABrokenOrOversizedPictureNeverReachesTheDisk(t *testing.T) {
	cases := map[string]struct {
		file []byte
		want error
	}{
		"text":            {file: []byte("not a picture"), want: crosshair.ErrNotPNG},
		"600 pixels wide": {file: pngOf(t, 600), want: crosshair.ErrTooLarge},
	}

	for name, c := range cases {
		disk := newMemory(map[string]string{})

		_, err := profiles.Store{Disk: disk, Dir: folder}.AddPicture(c.file)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
		if len(disk.files) != 0 || disk.dirs[pictures] {
			t.Errorf("%s: the disk was touched: files %v, crosshairs folder made %v", name, keys(disk.files), disk.dirs[pictures])
		}
	}
}

func TestAPictureThatFailsToSaveLeavesNothingBehind(t *testing.T) {
	for _, failing := range []string{"create", "write", "rename"} {
		disk := newMemory(map[string]string{})
		disk.fail[failing] = true

		_, err := profiles.Store{Disk: disk, Dir: folder}.AddPicture(pngOf(t, 8))
		if !errors.Is(err, profiles.ErrPictureNotSaved) || !errors.Is(err, errDenied) {
			t.Errorf("failing %s: got %v, want ErrPictureNotSaved wrapping the cause", failing, err)
		}
		if len(disk.files) != 0 {
			t.Errorf("failing %s: left %v behind", failing, keys(disk.files))
		}
	}
}

func TestOnlyAPictureNamedByItsHashIsRead(t *testing.T) {
	disk := newMemory(map[string]string{saved: `{}`})
	store := profiles.Store{Disk: disk, Dir: folder}

	for _, name := range []string{"../profiles.json", "..\\profiles.json", hashName([]byte("x"))[:10] + ".png"} {
		if _, err := store.Picture(name); !errors.Is(err, profiles.ErrNoPicture) || errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%q: got %v, want ErrNoPicture without a look at the disk", name, err)
		}
	}

	missing := hashName([]byte("gone"))
	if _, err := store.Picture(missing); !errors.Is(err, profiles.ErrNoPicture) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a deleted picture: got %v, want ErrNoPicture wrapping the cause", err)
	}
}

func TestACrosshairIsPaintedFromItsStyleUnlessItNamesAPicture(t *testing.T) {
	disk := newMemory(map[string]string{})
	store := profiles.Store{Disk: disk, Dir: folder}
	file := pngOf(t, 8)
	name, err := store.AddPicture(file)
	if err != nil {
		t.Fatal(err)
	}

	drawn, err := store.Paint(profiles.Crosshair{Style: specStyle, Scale: 300})
	if want, _ := crosshair.Draw(specStyle); err != nil || !reflect.DeepEqual(drawn, want) {
		t.Errorf("a style alone: got %v, %v; want the drawn crosshair", drawn.Rect, err)
	}

	pictured, err := store.Paint(profiles.Crosshair{Style: specStyle, Image: name, Scale: 300})
	if want, _ := crosshair.FromPNG(file, 300); err != nil || !reflect.DeepEqual(pictured, want) {
		t.Errorf("a picture: got %v, %v; want the picture at 300 percent", pictured.Rect, err)
	}
}

func TestAMissingPictureIsReportedRatherThanPainted(t *testing.T) {
	store := profiles.Store{Disk: newMemory(map[string]string{}), Dir: folder}

	_, err := store.Paint(profiles.Crosshair{Style: specStyle, Image: hashName([]byte("gone")), Scale: 100})
	if !errors.Is(err, profiles.ErrNoPicture) {
		t.Errorf("got %v, want ErrNoPicture", err)
	}
}

func keys(files map[string]string) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}

	return names
}
