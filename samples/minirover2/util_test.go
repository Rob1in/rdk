package main

import (
	"image/color"
	"strings"
	"testing"

	"github.com/echolabsinc/robotcore/vision"
	"github.com/echolabsinc/robotcore/vision/segmentation"

	"github.com/disintegration/imaging"
)

type MyDebug struct {
}

func (ddd MyDebug) Process(d *vision.MultipleImageTestDebugger, fn string, img vision.Image) error {
	dm, err := vision.ParseDepthMap(strings.Replace(fn, ".png", ".dat.gz", 1))
	if err != nil {
		return err
	}

	pc := vision.PointCloud{dm, img}

	pc, err = pc.CropToDepthData()
	if err != nil {
		return err
	}
	colorImg, err := pc.Color.ToImage()
	if err != nil {
		return err
	}
	d.GotDebugImage(colorImg, "cropped")

	walked, _ := roverWalk(&pc, true)
	d.GotDebugImage(walked, "depth2")

	return nil
}

func Test1(t *testing.T) {
	d := vision.NewMultipleImageTestDebugger("minirover2/autodrive", "*.png")
	err := d.Process(MyDebug{})
	if err != nil {
		t.Fatal(err)
	}

}

// ----

type ChargeDebug struct {
}

func (cd ChargeDebug) Process(d *vision.MultipleImageTestDebugger, fn string, img vision.Image) error {
	goImg, err := img.ToImage()
	if err != nil {
		return err
	}

	goImg = imaging.Rotate(goImg, 180, color.Black)

	img, err = vision.NewImage(goImg)
	if err != nil {
		return err
	}

	d.GotDebugImage(goImg, "rotated")

	m2, err := segmentation.ShapeWalkEntireDebug(img, false)
	if err != nil {
		return err
	}
	d.GotDebugImage(m2, "segments")

	return nil
}

func TestCharge1(t *testing.T) {
	d := vision.NewMultipleImageTestDebugger("minirover2/charging2", "*.both.gz")
	err := d.Process(ChargeDebug{})
	if err != nil {
		t.Fatal(err)
	}

}
