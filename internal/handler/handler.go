package handler

import "search-gin/internal/service"

type FileHandler struct {
	engine  service.SearchEngineInterface
	fileSvc service.FileServiceInterface
	ve      service.VideoEncoderInterface
}

var fileHandler = &FileHandler{
	engine:  &service.SearchEngine,
	fileSvc: service.SearchApp,
	ve:      service.VideoEncoder,
}
