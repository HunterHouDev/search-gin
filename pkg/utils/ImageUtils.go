package utils

import (
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
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(fin *os.File) {
		err := fin.Close()
		if err != nil {
			InfoFormat("ImageToPng: 关闭源文件失败: %v", err)
		}
	}(fin)
	fout, createErr := os.Create(des)
	if createErr != nil {
		InfoFormat("err:%v", createErr)
		return createErr
	}
	defer func(fout *os.File) {
		err := fout.Close()
		if err != nil {
			InfoFormat("ImageToPng: 关闭目标文件失败: %v", err)
		}
	}(fout)
	srcImage, fm, err := image.Decode(fin)
	if err != nil {
		InfoFormat("err:%v", err)
		return err
	}
	height := srcImage.Bounds().Max.Y
	width := srcImage.Bounds().Max.X
	left := int(0.53 * float64(width))
	switch fm {
	case "jpeg":
		rgbImg, ok := srcImage.(*image.YCbCr)
		if !ok {
			InfoFormat("ImageToPng: jpeg 类型断言失败")
			return nil
		}
		subImg, ok := rgbImg.SubImage(image.Rect(left, 0, width, height)).(*image.YCbCr)
		if !ok {
			InfoFormat("ImageToPng: jpeg SubImage 类型断言失败")
			return nil
		}
		err := png.Encode(fout, subImg)
		if err != nil {
			return err
		}
	case "png":
		switch srcImage.(type) {
		case *image.NRGBA:
			img, ok := srcImage.(*image.NRGBA)
			if !ok {
				InfoFormat("ImageToPng: png NRGBA 类型断言失败")
				return nil
			}
			subImg, ok := img.SubImage(image.Rect(left, 0, width, height)).(*image.NRGBA)
			if !ok {
				return nil
			}
			return png.Encode(fout, subImg)
		case *image.RGBA:
			img, ok := srcImage.(*image.RGBA)
			if !ok {
				InfoFormat("ImageToPng: png RGBA 类型断言失败")
				return nil
			}
			subImg, ok := img.SubImage(image.Rect(left, 0, width, height)).(*image.RGBA)
			if !ok {
				return nil
			}
			return png.Encode(fout, subImg)
		}
	}
	return nil

}

// CompressPngIfNeed 如果PNG图片大于500KB则压缩
// func CompressPngIfNeed(filePath string) ([]byte, error) {
// 	fileInfo, err := os.Stat(filePath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	fileData, err := os.ReadFile(filePath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if fileInfo.Size() <= MaxImageSize {
// 		return fileData, nil
// 	}

// 	img, err := png.Decode(bytes.NewReader(fileData))
// 	if err != nil {
// 		return fileData, nil
// 	}

// 	var buf bytes.Buffer
// 	encoder := png.Encoder{CompressionLevel: png.BestCompression}
// 	if err := encoder.Encode(&buf, img); err != nil {
// 		return fileData, nil
// 	}

// 	if buf.Len() < len(fileData) {
// 		return buf.Bytes(), nil
// 	}

// 	return fileData, nil
// }
