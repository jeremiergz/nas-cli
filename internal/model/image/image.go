package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os/exec"

	"golang.org/x/image/draw"

	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

type Image struct {
	FilePath string
	Name     string
	Kind     Kind
}

// Creates a new Image instance with the given name, file path, and kind (background or poster).
func New(name, filePath string, kind Kind) *Image {
	return &Image{
		FilePath: filePath,
		Name:     name,
		Kind:     kind,
	}
}

type Kind string

const (
	DefaultDPIX int = 72
	DefaultDPIY int = 72

	KindBackground Kind = "background"
	KindPoster     Kind = "poster"

	KindBackgroundWidth  int = 3840
	KindBackgroundHeight int = 2160
	KindPosterWidth      int = 1000
	KindPosterHeight     int = 1500
)

var (
	ValidExtensions = []string{"jpg", "jpeg", "png", "webp"}
)

func (ik Kind) String() string {
	return string(ik)
}

// Scales given JPG image to specific dimensions depending on the image kind (background or poster).
func Scale(src image.Image, kind Kind) image.Image {
	var width, height int

	switch kind {
	case KindBackground:
		width = KindBackgroundWidth
		height = KindBackgroundHeight
	case KindPoster:
		width = KindPosterWidth
		height = KindPosterHeight
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	return dst
}

// Sets the DPI (Dots Per Inch) EXIF metadata of the given image file using ExifTool.
func SetDPI(ctx context.Context, inputPath string) error {
	options := []string{
		fmt.Sprintf("-XResolution=%d", DefaultDPIX),
		fmt.Sprintf("-YResolution=%d", DefaultDPIY),
		"-ResolutionUnit=inches",
		"-overwrite_original",
		inputPath,
	}

	exiftool := exec.CommandContext(ctx, cmdutil.CommandExifTool, options...)

	bufErr := new(bytes.Buffer)
	exiftool.Stderr = bufErr

	if err := exiftool.Run(); err != nil {
		return fmt.Errorf("failed to run ExifTool: %w\n%s", err, bufErr.String())
	}

	return nil
}
