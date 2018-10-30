package main

import (
	"fmt"
	"os"

	"github.com/gildasch/gildas-ai/image"
	"github.com/gildasch/gildas-ai/tensor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s path/to/image.jpg\n", os.Args[0])
		return
	}
	imageName := os.Args[1]

	model, close, err := tensor.NewModel("myModel", "myTag")
	if err != nil {
		fmt.Printf("Error loading saved model: %s\n", err.Error())
		return
	}
	defer close()

	img, err := image.FromFile(imageName)
	if err != nil {
		fmt.Printf("cannot read image %q: %v\n", imageName, err)
		return
	}

	preds, err := model.Inception(img)
	if err != nil {
		fmt.Printf("there was an error while running the inception: %v\n", err)
		return
	}

	best, bestV := preds.Best()
	fmt.Printf("Result value: %v (%f) \n", best, bestV)
}
