package main

import (
	"crypto/md5"
	"fmt"
	"github.com/schwarmco/go-cartesian-product"
	"github.com/spf13/viper"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	// Load configuration from file
	viper.SetConfigName("config.toml")
	viper.SetConfigType("toml")
	viper.SetConfigFile("./config.toml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalln(err)
	}

	// Setup output directory
	outputDir := viper.GetString("outputDir")
	if outputDir == "" {
		outputDir = "output"
	}
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.Mkdir(outputDir, 755)
		if err != nil {
			log.Fatalln(err)
		}
	}

	// Setup image width
	imgWidth := viper.GetInt("imageWidth")
	if imgWidth < 1 {
		log.Fatalln("Invalid image width")
	}

	// Setup image height
	imgHeight := viper.GetInt("imageHeight")
	if imgHeight < 1 {
		log.Fatalln("Invalid image height")
	}

	// Prepare data structures
	dirs := viper.GetStringSlice("layers")
	if len(dirs) == 0 {
		log.Fatalln("Layers not defined")
	}
	var dirSlice [][]string
	for i := 0; i < len(dirs); i++ {
		dirSlice = append(dirSlice, []string{})
	}

	// Build map of all files
	for i, dir := range dirs {
		files, err := os.ReadDir("./layers/" + dir)
		if err != nil {
			log.Fatalln(err)
		}
		for _, file := range files {
			if !file.IsDir() {
				dirSlice[i] = append(dirSlice[i], "./layers/"+dir+"/"+file.Name())
			}
		}
	}

	// Create 2D interface slice
	var combinations int
	cartesianSlice := make([][]interface{}, len(dirSlice))
	for i := range dirSlice {
		if combinations < 1 {
			combinations = len(dirSlice[i])
		} else {
			combinations *= len(dirSlice[i])
		}
		for j := range dirSlice[i] {
			cartesianSlice[i] = append(cartesianSlice[i], dirSlice[i][j])
		}
	}
	log.Println("Generating", combinations, "images")

	// Create wait group
	var waitGroup sync.WaitGroup

	// Loop through all combinations
	c := cartesian.Iter(cartesianSlice...)
	for product := range c {
		// Add 1 to wait group
		waitGroup.Add(1)

		// Generate image for combination
		go func(product []interface{}) {
			// Store start time for performance measuring
			startTime := time.Now()

			// Create blank image
			img := image.NewRGBA(image.Rectangle{
				Min: image.Point{},
				Max: image.Point{X: imgWidth, Y: imgHeight},
			})

			// Loop through image layers
			for _, file := range product {
				imgFile, err := os.Open(fmt.Sprintf("%s", file))
				if err != nil {
					log.Fatalln(err)
				}
				ext := filepath.Ext(fmt.Sprintf("%s", file))

				// Decode compatible file types
				var layer image.Image
				if ext == ".jpg" || ext == ".jpeg" {
					layer, err = jpeg.Decode(imgFile)
				} else if ext == ".png" {
					layer, err = png.Decode(imgFile)
				} else {
					log.Fatalln("File extension not supported:", ext)
				}
				if err != nil {
					log.Fatalln(err)
				}

				// Add layer on top of the current image
				draw.Draw(img, img.Bounds(), layer, image.Point{}, draw.Over)
				imgFile.Close()
			}

			// Create 2D string slice for product
			stringSlice := make([]string, len(product))
			for i := range product {
				stringSlice[i] = fmt.Sprintf("%s", product[i])
			}

			// Save file
			productName := strings.Join(stringSlice, " ")
			filename := fmt.Sprintf("%x", md5.Sum([]byte(productName)))
			output, err := os.Create("./" + outputDir + "/" + filename + ".jpg")
			if err != nil {
				log.Fatalln(err)
			}
			jpeg.Encode(output, img, &jpeg.Options{Quality: jpeg.DefaultQuality})
			output.Close()

			// Output details
			log.Println("Generated "+filename+".jpg in", time.Since(startTime))

			// Remove worker from wait group
			waitGroup.Done()
		}(product)
	}

	// Wait for all goroutines
	waitGroup.Wait()
}
