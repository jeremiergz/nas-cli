package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
	"golang.org/x/sync/errgroup"

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

// Converts a slice of images to meet the media server requirements.
func ConvertToRequirements(images []*Image) error {
	eg := errgroup.Group{}
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	for _, img := range images {
		eg.Go(func() error {
			newPath, err := convertImageFileToRequirements(img.FilePath, img.Kind)
			if err != nil {
				return fmt.Errorf("failed to convert %s image file %s: %w", img.Kind, img.FilePath, err)
			}
			img.FilePath = newPath
			return nil
		})
	}

	return eg.Wait()
}

// Converts the image file to meet the requirements of the media server (dimensions, format, DPI) depending on the kind
// of image (background or poster).
func convertImageFileToRequirements(src string, kind Kind) (string, error) {
	imgName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	imgExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))

	srcFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer srcFile.Close()

	var decoded image.Image
	shouldEncode := false
	shouldDeleteSourceFile := false

	switch imgExtension {
	case "jpg", "jpeg":
		decoded, err = jpeg.Decode(srcFile)
		if err != nil {
			return "", fmt.Errorf("failed to decode jpeg image file: %w", err)
		}
		if imgExtension == "jpeg" {
			err = os.Rename(src, imgName+".jpg")
			if err != nil {
				return "", fmt.Errorf("failed to rename jpeg image file: %w", err)
			}
		}

	case "png":
		decoded, err = png.Decode(srcFile)
		if err != nil {
			return "", fmt.Errorf("failed to decode png image file: %w", err)
		}
		shouldEncode = true
		shouldDeleteSourceFile = true

	case "webp":
		decoded, err = webp.Decode(srcFile)
		if err != nil {
			return "", fmt.Errorf("failed to decode webp image file: %w", err)
		}
		shouldEncode = true
		shouldDeleteSourceFile = true

	default:
		return "", fmt.Errorf("unsupported image file format: %s", imgExtension)
	}

	// Check if the image has the desired dimensions.
	var expectedX, expectedY int
	switch kind {
	case KindBackground:
		expectedX = KindBackgroundWidth
		expectedY = KindBackgroundHeight
	case KindPoster:
		expectedX = KindPosterWidth
		expectedY = KindPosterHeight
	}
	currentX := decoded.Bounds().Dx()
	currentY := decoded.Bounds().Dy()
	if currentX != expectedX || currentY != expectedY {
		shouldEncode = true
	}

	outputFilePath := imgName + ".jpg"

	if shouldEncode {
		decoded = Scale(decoded, kind)

		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to create output image file: %w", err)
		}
		defer outputFile.Close()

		err = jpeg.Encode(outputFile, decoded, &jpeg.Options{Quality: 90})
		if err != nil {
			return "", fmt.Errorf("failed to encode jpeg image file: %w", err)
		}
	}

	if shouldDeleteSourceFile {
		err = os.Remove(src)
		if err != nil {
			return "", fmt.Errorf("failed to remove original image file: %w", err)
		}
	}

	err = SetDPI(context.Background(), outputFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to set DPI for image file: %w", err)
	}

	return outputFilePath, nil
}
