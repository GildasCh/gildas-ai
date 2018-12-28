package gildasai

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/pkg/errors"
)

type Detection struct {
	Box   image.Rectangle
	Score float32
	Class float32
}

func Above(allDetections []Detection, threshold float32) []Detection {
	var above []Detection

	for _, d := range allDetections {
		if d.Score < threshold {
			continue
		}
		above = append(above, d)
	}

	return above
}

type Landmarks struct {
	Coords []float32
}

func (l *Landmarks) PointsOnImage(img image.Image) []image.Point {
	w, h := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	minX, minY := img.Bounds().Min.X, img.Bounds().Min.Y

	points := []image.Point{}
	for i := 0; i < len(l.Coords)-1; i += 2 {
		points = append(points, image.Point{
			X: minX + int(w*l.Coords[i]),
			Y: minY + int(h*l.Coords[i+1]),
		})
	}

	return points
}

func (l *Landmarks) DrawOnImage(img image.Image) image.Image {
	out := image.NewRGBA(img.Bounds())

	draw.Draw(out, img.Bounds(), img, image.ZP, draw.Src)

	for _, p := range l.PointsOnImage(img) {
		drawPoint(out, p)
	}

	return out
}

func (l *Landmarks) DrawOnFullImage(cropped, full image.Image) image.Image {
	out := image.NewRGBA(full.Bounds())

	draw.Draw(out, full.Bounds(), full, image.ZP, draw.Src)

	for _, p := range l.PointsOnImage(cropped) {
		drawPoint(out, p)
	}

	return out
}

func drawPoint(img *image.RGBA, p image.Point) {
	width := 3

	for i := p.X - width/2; i < p.X+width/2; i++ {
		for j := p.Y - width/2; j < p.Y+width/2; j++ {
			img.Set(i, j, color.RGBA{G: 255})
		}
	}
}

func (l *Landmarks) Center(cropped, full image.Image) image.Image {
	bounds := full.Bounds()
	minX, minY, maxX, maxY := bounds.Max.X, bounds.Max.Y, bounds.Min.X, bounds.Min.Y

	for _, p := range l.PointsOnImage(cropped) {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	rect := image.Rectangle{
		Min: image.Point{
			X: minX,
			Y: minY,
		},
		Max: image.Point{
			X: maxX,
			Y: maxY,
		},
	}

	rect = square(rect)
	rect = insideOf(rect, bounds)

	out := image.NewRGBA(rect)

	draw.Draw(out, out.Bounds(), full, rect.Min, draw.Src)

	return out
}

func square(rect image.Rectangle) image.Rectangle {
	width, height := rect.Max.X-rect.Min.X, rect.Max.Y-rect.Min.Y

	if height > width {
		left := (height - width) / 2
		right := height - width - left

		return image.Rectangle{
			Min: image.Point{
				X: rect.Min.X - left,
				Y: rect.Min.Y,
			},
			Max: image.Point{
				X: rect.Max.X + right,
				Y: rect.Max.Y,
			},
		}
	}

	if width > height {
		top := (width - height) / 2
		bottom := width - height - top

		return image.Rectangle{
			Min: image.Point{
				X: rect.Min.X,
				Y: rect.Min.Y - top,
			},
			Max: image.Point{
				X: rect.Max.X,
				Y: rect.Max.Y + bottom,
			},
		}
	}

	return rect
}

func insideOf(rect, bounds image.Rectangle) image.Rectangle {
	if bounds.Min.X > rect.Min.X {
		rect.Max.X += bounds.Min.X - rect.Min.X
		rect.Min.X = bounds.Min.X
	}

	if bounds.Min.Y > rect.Min.Y {
		rect.Max.Y += bounds.Min.Y - rect.Min.Y
		rect.Min.Y = bounds.Min.Y
	}

	if rect.Max.X > bounds.Max.X {
		rect.Min.X -= rect.Max.X - bounds.Max.X
		rect.Max.X = bounds.Max.X
	}

	if rect.Max.Y > bounds.Max.Y {
		rect.Min.Y -= rect.Max.Y - bounds.Max.Y
		rect.Max.Y = bounds.Max.Y
	}

	return rect
}

type Descriptors []float32

func (d Descriptors) DistanceTo(d2 Descriptors) (float32, error) {
	if len(d) != len(d2) {
		return 0, errors.Errorf(
			"cannot calculate distance between descriptors of dimensions %d and %d", len(d), len(d2))
	}

	sum := float32(0)

	for i := 0; i < len(d); i++ {
		sum += (d[i] - d2[i]) * (d[i] - d2[i])
	}

	return float32(math.Sqrt(float64(sum))), nil
}

type Detector interface {
	Detect(img image.Image) ([]Detection, error)
}

type Landmark interface {
	Detect(img image.Image) (*Landmarks, error)
}

type Descriptor interface {
	Compute(img image.Image) (Descriptors, error)
}

type Extractor struct {
	Detector   Detector
	Landmark   Landmark
	Descriptor Descriptor
}

func (e *Extractor) Extract(img image.Image) ([]image.Image, []Descriptors, error) {
	allDetections, err := e.Detector.Detect(img)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error detecting faces")
	}

	detections := Above(allDetections, 0.6)

	images := []image.Image{}
	descrs := []Descriptors{}
	for _, d := range detections {
		if d.Box.Dx() < 45 || d.Box.Dy() < 45 {
			continue // face is too small
		}

		cropped := image.NewRGBA(d.Box)
		draw.Draw(cropped, d.Box, img, d.Box.Min, draw.Src)

		landmarks, err := e.Landmark.Detect(cropped)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error detecting landmarks")
		}

		cropped2 := landmarks.Center(cropped, img)

		descriptors, err := e.Descriptor.Compute(cropped2)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error computing descriptors")
		}

		images = append(images, cropped2)
		descrs = append(descrs, descriptors)
	}

	return images, descrs, nil
}

func (e *Extractor) ExtractLandmarks(img image.Image) ([][]image.Point, []image.Image, error) {
	allDetections, err := e.Detector.Detect(img)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error detecting faces")
	}

	detections := Above(allDetections, 0.4)

	if len(detections) == 0 {
		return nil, nil, errors.New("no face detected")
	}

	var ret [][]image.Point
	var crops []image.Image
	for _, d := range detections {
		if d.Box.Dx() < 45 || d.Box.Dy() < 45 {
			continue // face is too small
		}

		cropped := image.NewRGBA(d.Box)
		draw.Draw(cropped, d.Box, img, d.Box.Min, draw.Src)

		landmarks, err := e.Landmark.Detect(cropped)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error detecting landmarks")
		}

		ret = append(ret, landmarks.PointsOnImage(cropped))
		crops = append(crops, cropped)
	}

	return ret, crops, nil
}