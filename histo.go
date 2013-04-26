package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

type compRes struct {
	file string
	match float64
}
type matches map[string][]*compRes

var isJpeg = regexp.MustCompile(`.*\.(jpg|jpeg)$`)

func diffDir(dirpath string) (matches, error) {
	results := make(matches)
	f, err := os.Open(dirpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for _, v1 := range names {
		if !isJpeg.MatchString(strings.ToLower(v1)) {
			continue
		}
		var res []*compRes
		fv1 := filepath.Join(dirpath, v1)
		for _, v2 := range names {
			if v1 == v2 {
				continue
			}
			if !isJpeg.MatchString(strings.ToLower(v2)) {
				continue
			}
			fv2 := filepath.Join(dirpath, v2)
			match, err := diffFiles(fv1, fv2)
			if err != nil {
				log.Print(err)
				continue
			}
			res = append(res, &compRes{fv2, match})
			fmt.Printf("(%v, %v) : %f\n", v1, v2, match)
		}
		results[fv1] = res
	}
	return results, nil
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
*/
	_, err := diffDir("/home/mpl/Desktop/pics/passur/pleubian/")
	if err != nil {
		panic(err)
	}
}

/*
diff1: +9.679229e-002 ; +2.593293e+000
diff2: +8.978444e-001 ; +2.800825e-001
diff3: +8.978442e-001 ; +2.800825e-001

5000 5000 6
4998 5008 0


(20110427_001.jpg, 20110430_001.jpg) : 0.449549
(20110427_001.jpg, 20110430_003.jpg) : 0.401911
(20110427_001.jpg, 20110424_001.jpg) : 0.528231
(20110427_001.jpg, 20110430_002.jpg) : 0.464364
(20110430_001.jpg, 20110427_001.jpg) : 0.449549
(20110430_001.jpg, 20110430_003.jpg) : 0.471735
(20110430_001.jpg, 20110424_001.jpg) : 0.322794
(20110430_001.jpg, 20110430_002.jpg) : 0.435669
(20110430_003.jpg, 20110427_001.jpg) : 0.403487
(20110430_003.jpg, 20110430_001.jpg) : 0.473585
(20110430_003.jpg, 20110424_001.jpg) : 0.308662
(20110430_003.jpg, 20110430_002.jpg) : 0.399664
(20110424_001.jpg, 20110427_001.jpg) : 0.528231
(20110424_001.jpg, 20110430_001.jpg) : 0.322794
(20110424_001.jpg, 20110430_003.jpg) : 0.307457
(20110424_001.jpg, 20110430_002.jpg) : 0.399792
(20110430_002.jpg, 20110427_001.jpg) : 0.464364
(20110430_002.jpg, 20110430_001.jpg) : 0.435669
(20110430_002.jpg, 20110430_003.jpg) : 0.398103
(20110430_002.jpg, 20110424_001.jpg) : 0.399792
-> diff3 not convincing
*/
