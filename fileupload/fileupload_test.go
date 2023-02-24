package fileupload

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ngamux/ngamux"
)

func TestFileUpload(t *testing.T) {
	mux := ngamux.New()
	mux.Put("/profiles", New(Config{
		Destination: "uploads/pictures",
		FormKey:     "report",
		FilenameFunc: func(req *http.Request) (string, error) {
			return "profile-picture.jpg", nil
		},
	})(func(rw http.ResponseWriter, r *http.Request) error {
		rw.WriteHeader(200)
		rw.Write([]byte("Ok"))
		return nil
	}))

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer writer.Close()
		//we create the form data field 'fileupload'
		//wich returns another writer to write the actual file
		part, err := writer.CreateFormFile("report", "someimg.png")
		if err != nil {
			t.Error(err)
		}

		//https://yourbasic.org/golang/create-image/
		img := createImage()
		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	request := httptest.NewRequest("PUT", "/profiles", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()

	request.Header.Add("Content-Type", writer.FormDataContentType())
	mux.ServeHTTP(rec, request)

	result := strings.ReplaceAll(rec.Body.String(), "\n", "")
	expected := "Ok"

	if result != expected {
		t.Errorf("TestPost need %v, but got %v", expected, result)
	}

	_, err := os.Stat("uploads/pictures/profile-picture.jpg")

	if err != nil {
		t.Error("File is not uploaded to given destination")
	}

	os.RemoveAll("uploads")
}

func createImage() *image.RGBA {
	width := 200
	height := 100

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	// Colors are defined by Red, Green, Blue, Alpha uint8 values.
	cyan := color.RGBA{100, 200, 200, 0xff}

	// Set color for each pixel.
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			switch {
			case x < width/2 && y < height/2: // upper left quadrant
				img.Set(x, y, cyan)
			case x >= width/2 && y >= height/2: // lower right quadrant
				img.Set(x, y, color.White)
			default:
				// Use zero value.
			}
		}
	}

	return img
}
