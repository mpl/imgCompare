package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
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

// TODO(mpl): compensate if len(histo1) != len(histo2)
// -> instead of looping over histo1; loop [0-256) ?
func diff2(im1, im2 image.Image) float64 {
	histo1 := Histo(im1)
	histo2 := Histo(im2)
	var cumratio float32
	for lum, count := range histo1 {
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
	for i := 0; i < 256; i++ {
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

func mean(x []float64) float64 {
	sx := 0.0
	for _, v := range x {
		sx += v
	}
	return sx / float64(len(x))
}

func denominator(x, y []float64, mx, my float64) float64 {
	sx, sy := 0., 0.
	for i := 0; i < len(x); i++ {
		sx += (x[i] - mx) * (x[i] - mx)
		sy += (y[i] - my) * (y[i] - my)
	}
	return math.Sqrt(sx * sy)
}

func xCorrelation(x, y []float64) float64 {
	mx := mean(x)
	my := mean(y)

	denom := denominator(x, y, mx, my)

	sxy := 0.0
	for i := 0; i < len(x); i++ {
		sxy += (x[i] - mx) * (y[i] - my)
	}
	return sxy / denom
}

func diff4(im1, im2 image.Image) float64 {
	histo1 := Histo(im1)
	histo2 := Histo(im2)
	var x []float64
	var y []float64
	for i := 0; i < 256; i++ {
		x = append(x, float64(histo1[uint8(i)]))
		y = append(y, float64(histo2[uint8(i)]))
	}
	return xCorrelation(x, y)
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
	return diff4(im1, im2), nil
}

type compRes struct {
	file  string
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
//	c := make(chan int)
	var wg sync.WaitGroup
	for k1, v1 := range names {
		wg.Add(1)
		go func(k int, v string) {
			defer wg.Done()
			if !isJpeg.MatchString(strings.ToLower(v)) {
				return
			}
			var res []*compRes
			fv1 := filepath.Join(dirpath, v)
//			fv1 := v
			for k2, v2 := range names {
				if k2 <= k {
					continue
				}
				if !isJpeg.MatchString(strings.ToLower(v2)) {
					continue
				}
				fv2 := filepath.Join(dirpath, v2)
//				fv2 := v2
				match, err := diffFiles(fv1, fv2)
				if err != nil {
					log.Print(err)
					continue
				}
				res = append(res, &compRes{fv2, match})
				fmt.Printf("(%v, %v) : %f\n", v, v2, match)
			}
			if len(res) > 0 {
				results[fv1] = res
			}
//			c <- 1
		}(k1, v1)
	}

/*
	n := 0
	for {
		<-c
		n++
		println(n)
		if n == len(names) {
			break
		}
	}
*/
	wg.Wait()
	return results, nil
}

func uniquify(m matches) map[string]*compRes {
	bestpairs := make(map[string]*compRes)
	for k1, v1 := range m {
		best := 0.
		keep := 0
		for k, cres := range v1 {
			if math.Abs(cres.match) > best {
				best = math.Abs(cres.match)
				keep = k
			}
		}
		fmt.Printf("(%v, %v) : %f\n", k1, v1[keep].file, v1[keep].match)
		bestpairs[k1] = v1[keep]
	}
	return bestpairs
}

type rankedPair struct {
	pic1 string
	pic2 string
	rank float64
}

type sortedPairs []*rankedPair

// Len is part of sort.Interface.
func (s sortedPairs) Len() int {
	return len(s)
}

func (s sortedPairs) Less(i, j int) bool {
	return math.Abs(s[i].rank) > math.Abs(s[j].rank)
}

// Swap is part of sort.Interface.
func (s sortedPairs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func bestPairsToSortedPairs(m map[string]*compRes) sortedPairs {
	s := make(sortedPairs, len(m))
	i := 0
	for k, v := range m {
		meh := &rankedPair{
			pic1: k,
			pic2: v.file,
			rank: v.match,
		}
		s[i] = meh
		i++
	}
	sort.Sort(s)
	return s
}

func renameAll(pairs sortedPairs, destDir string) {
	if destDir == "" {
		destDir = "sorted"
	}
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		panic(err)
	}
	done := make(map[string]bool)
	var ext1, name1, ext2, name2, dest string
	for k, v := range pairs {
		if _, ok := done[v.pic1]; !ok {
			ext1 = filepath.Ext(v.pic1)
			name1 = fmt.Sprintf("%d%s", k*2, ext1)
			dest = filepath.Join(destDir, name1)
			cmd := exec.Command("cp", v.pic1, dest)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
			done[v.pic1] = true
		}
		if _, ok := done[v.pic2]; !ok {
			ext2 = filepath.Ext(v.pic2)
			name2 = fmt.Sprintf("%d%s", k*2+1, ext2)
			dest = filepath.Join(destDir, name2)
			cmd := exec.Command("cp", v.pic2, dest)
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
			done[v.pic2] = true
		}
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	dir := ""
	if len(args) != 1 {
		dir = "/home/mpl/Desktop/test/pleubian/"
	} else {
		dir = args[0]
	}
	allpairs, err := diffDir(dir)
	if err != nil {
		panic(err)
	}
	/*
		allpairstest := matches{
			"maison.jpg": []*compRes{
				&compRes{"plage.jpg", 0.791355},
				&compRes{"voiture.jpg", 0.575118},
				&compRes{"tour.jpg", -0.243935},
				&compRes{"vipere.jpg", 0.054535},
			},
			"plage.jpg": []*compRes{
				&compRes{"voiture.jpg", 0.427355},
				&compRes{"tour.jpg", -0.184758},
				&compRes{"vipere.jpg", 0.082849},
			},
			"voiture.jpg": []*compRes{
				&compRes{"tour.jpg", 0.020259},
				&compRes{"vipere.jpg", -0.165653},
			},
			"tour.jpg": []*compRes{
				&compRes{"vipere.jpg", 0.412750},
			},
		}
	*/
	println("bestpairs")
	//	bestpairs := uniquify(allpairstest)
	bestpairs := uniquify(allpairs)
	println("sorted pairs")
	sortedPairs := bestPairsToSortedPairs(bestpairs)
	for _, v := range sortedPairs {
		fmt.Println(*v)
	}
	renameAll(sortedPairs, "/home/mpl/Desktop/pleubian/sorted")
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

diff4:
(maison.jpg, plage.jpg) : 0.791355
(maison.jpg, voiture.jpg) : 0.575118
(maison.jpg, tour.jpg) : -0.243935
(maison.jpg, vipere.jpg) : 0.054535
(plage.jpg, voiture.jpg) : 0.427355
(plage.jpg, tour.jpg) : -0.184758
(plage.jpg, vipere.jpg) : 0.082849
(voiture.jpg, tour.jpg) : 0.020259
(voiture.jpg, vipere.jpg) : -0.165653
(tour.jpg, vipere.jpg) : 0.412750

(maison.jpg, plage.jpg) : 0.791355
(maison.jpg, voiture.jpg) : 0.575118
(plage.jpg, voiture.jpg) : 0.427355
(tour.jpg, vipere.jpg) : 0.412750
(plage.jpg, vipere.jpg) : 0.082849
(maison.jpg, vipere.jpg) : 0.054535
(voiture.jpg, tour.jpg) : 0.020259
(voiture.jpg, vipere.jpg) : -0.165653
(plage.jpg, tour.jpg) : -0.184758
(maison.jpg, tour.jpg) : -0.243935

-> fuck yeah!

uniquify:

bestpairs
(maison.jpg, plage.jpg) : 0.791355
(plage.jpg, voiture.jpg) : 0.427355
(voiture.jpg, vipere.jpg) : -0.165653
(tour.jpg, vipere.jpg) : 0.412750

with sialet -> not so convincing.
Is the initial method no good, or is it the following sort/grouping?

(IMG_20130526_175706.jpg, IMG_20130526_175401.jpg) : 0.898434
=> initial method no good.
but is it the cross correlation, or the idea to decompose the luminance by channels that fails?
-> I need to compare their distributions with gnuplot.

*/
