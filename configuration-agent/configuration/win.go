package configuration

import (
	"log"
	"os"
	"time"

	"github.com/kardianos/service"
)

type WindowsService struct {
	service.Service
	running   bool
	hWaitStop chan bool
}

func (ws *WindowsService) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go ws.run()
	return nil
}

func (ws *WindowsService) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	ws.running = false
	ws.hWaitStop <- true
	return nil
}

func (ws *WindowsService) run() {
	ws.running = true
	for ws.running {
		select {
		case <-ws.hWaitStop:
			ws.running = false
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func NewWindowsService() *WindowsService {
	return &WindowsService{
		hWaitStop: make(chan bool),
	}
}

func RunService(name, displayName, description string, svc service.Interface) {
	svcConfig := &service.Config{
		Name:        name,
		DisplayName: displayName,
		Description: description,
	}

	s, err := service.New(svc, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		err = service.Control(s, os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}
