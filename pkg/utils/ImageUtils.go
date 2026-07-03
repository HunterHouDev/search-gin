package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
)

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
	default:
	 errMsg := fmt.Sprintf("ImageToPng: 不支持的图片格式: %s", fm)
	 InfoNormal(errMsg)
	 return errors.New(errMsg)
	}
	return nil

}
