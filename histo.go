package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"os"
	"path/filepath"
)

func Histo(im image.Image) map[uint8]int {
	histo := map[uint8]int{}
	w := im.Bounds().Dx()
	h := im.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c, ok := im.At(x, y).(color.YCbCr)
			if !ok {
				panic("not a YCbCr")
			}
			if count, ok := histo[c.Y]; ok {
				histo[c.Y] = count + 1
			} else {
				histo[c.Y] = 1
			}
		}
	}
	return histo
}

func printHisto(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	im, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	histo := Histo(im)
	ext := filepath.Ext(filename)
	outname := filename[:len(filename)-len(ext)] + ".dat"
	g, err := os.Create(outname)
	if err != nil {
		return err
	}
	defer g.Close()
	for k, v := range histo {
		fmt.Fprintf(g, "%d	%d\n", k, v)
	}
	return nil
}

func diff(im1, im2 image.Image) float64 {
	histo1 := Histo(im1)
	histo2 := Histo(im2)
	cumdiff := 0
	cumcount := 0 // == numpixels, no?
	for lum, count := range histo1 {
		count2 := histo2[lum]
		diff := count - count2
		if diff < 0 {
			diff = 0 - diff
		}
		cumdiff += diff
		cumcount += count2
	}
	return float64(cumdiff) / float64(cumcount)
}

func diffFiles(file1, file2 string) (float64, error) {
	f, err := os.Open(file1)
	if err != nil {
		return 0., err
	}
	defer f.Close()
	im1, _, err := image.Decode(f)
	if err != nil {
		return 0., err
	}
	g, err := os.Open(file2)
	if err != nil {
		return 0., err
	}
	defer g.Close()
	im2, _, err := image.Decode(g)
	if err != nil {
		return 0., err
	}
	return diff(im1, im2), nil
}

func main() {
	/*
		err := printHisto("/home/mpl/Desktop/IMG_2336.JPG")
		if err != nil {
			panic(err)
		}
		err = printHisto("/home/mpl/Desktop/IMG_2337.JPG")
		if err != nil {
			panic(err)
		}
	*/
	//	d, err := diffFiles("/home/mpl/Desktop/IMG_2336.JPG", "/home/mpl/Desktop/IMG_2337.JPG")
	d, err := diffFiles("/home/mpl/Desktop/IMG_2336.JPG", "/home/mpl/Desktop/20110430_002.jpg")
	if err != nil {
		panic(err)
	}
	println(d)
}
