package imgutil

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os/exec"

	"golang.org/x/image/draw"

	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

const (
	DefaultDPIX int = 72
	DefaultDPIY int = 72
)

type ImageKind string

const (
	ImageKindBackground ImageKind = "background"
	ImageKindPoster     ImageKind = "poster"

	ImageKindBackgroundWidth  int = 3840
	ImageKindBackgroundHeight int = 2160
	ImageKindPosterWidth      int = 1000
	ImageKindPosterHeight     int = 1500
)

func (ik ImageKind) String() string {
	return string(ik)
}

// Scales given JPG image to specific dimensions depending on the image kind (background or poster).
func Scale(src image.Image, kind ImageKind) image.Image {
	var width, height int

	switch kind {
	case ImageKindBackground:
		width = ImageKindBackgroundWidth
		height = ImageKindBackgroundHeight
	case ImageKindPoster:
		width = ImageKindPosterWidth
		height = ImageKindPosterHeight
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
