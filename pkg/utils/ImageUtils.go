package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
)

const MaxImageSize = 500 * 1024 // 500KB

func ImageToString(path string) string {
	if !ExistsFiles(path) {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func ImageToPng(src string) error {
	des := ConcatSuffix(src, "png")
	fin, _ := os.Open(src)
	fin2, _ := os.Open(src)
	defer func(fin *os.File) {
		err := fin.Close()
		if err != nil {

		}
	}(fin)
	defer func(fin2 *os.File) {
		err := fin2.Close()
		if err != nil {

		}
	}(fin2)
	fout, createErr := os.Create(des)
	if createErr != nil {
		InfoFormat("err:%v", createErr)
		return createErr
	}
	defer func(fout *os.File) {
		err := fout.Close()
		if err != nil {

		}
	}(fout)
	config, _, _ := image.DecodeConfig(fin2)
	srcImage, fm, err := image.Decode(fin)
	if err != nil {
		InfoFormat("err:%v", err)
		return err
	}
	height := config.Height
	width := config.Width
	left := int(0.53 * float64(width))
	switch fm {
	case "jpeg":
		rgbImg := srcImage.(*image.YCbCr)
		subImg := rgbImg.SubImage(image.Rect(left, 0, width, height)).(*image.YCbCr)
		err := png.Encode(fout, subImg)
		if err != nil {
			return err
		}
	case "png":
		switch srcImage.(type) {
		case *image.NRGBA:
			img := srcImage.(*image.NRGBA)
			subImg := img.SubImage(image.Rect(left, 0, width, height)).(*image.NRGBA)
			return png.Encode(fout, subImg)
		case *image.RGBA:
			img := srcImage.(*image.RGBA)
			subImg := img.SubImage(image.Rect(left, 0, width, height)).(*image.RGBA)
			return png.Encode(fout, subImg)
		}
	}
	return nil

}

// CompressPngIfNeed 如果PNG图片大于500KB则压缩
func CompressPngIfNeed(filePath string) ([]byte, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if fileInfo.Size() <= MaxImageSize {
		return fileData, nil
	}

	img, err := png.Decode(bytes.NewReader(fileData))
	if err != nil {
		return fileData, nil
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return fileData, nil
	}

	if buf.Len() < len(fileData) {
		return buf.Bytes(), nil
	}

	return fileData, nil
}
