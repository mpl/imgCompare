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

func diff1(im1, im2 image.Image) float64 {
	histo1 := Histo(im1)
	histo2 := Histo(im2)
	cumdiff := 0
	cumcount := 0 // == numpixels, no?
	for lum,count := range histo1 {
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

// TODO(mpl): compensate if len(histo1) != len(histo2)
// -> instead of looping over histo1; loop [0-256) ?
func diff2(im1, im2 image.Image) float64 {
	histo1 := Histo(im1)
	histo2 := Histo(im2)
	var cumratio float32
	for lum,count := range histo1 {
		count2 := histo2[lum]
		if count == 0 || count2 == 0 {
			continue
		}
		ratio := float32(count2) / float32(count)
		if ratio > 1 {
			ratio = 1. / ratio
		}
		cumratio += ratio
	}
	return float64(cumratio) / float64(len(histo1))
}

func diff3(im1, im2 image.Image) float64 {
	histo1 := Histo(im1)
	histo2 := Histo(im2)
	var cumratio float32
	for i := 0; i<256; i++ {
		count1 := histo1[uint8(i)]
		count2 := histo2[uint8(i)]
		if count1 == 0 || count2 == 0 {
			continue
		}
		ratio := float32(count2) / float32(count1)
		if ratio > 1 {
			ratio = 1. / ratio
		}
		cumratio += ratio
	}
	return float64(cumratio) / float64(len(histo1))
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
	return diff3(im1, im2), nil
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
	d, err := diffFiles("/home/mpl/Desktop/IMG_2336.JPG", "/home/mpl/Desktop/IMG_2337.JPG")
	if err != nil {
		panic(err)
	}
	println(d)
	d, err = diffFiles("/home/mpl/Desktop/IMG_2336.JPG", "/home/mpl/Desktop/20110430_002.jpg")
	if err != nil {
		panic(err)
	}
	println(d)
}

/*
diff1: +9.679229e-002 ; +2.593293e+000
diff2: +8.978444e-001 ; +2.800825e-001
diff3: +8.978442e-001 ; +2.800825e-001

5000 5000 6
4998 5008 0

*/
