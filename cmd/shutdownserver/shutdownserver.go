/*
shutdownserver получает запрос get и сравнивает его со строкой формата /?shutdown=x
если есть shutdown, получает параметр х, переводит его в минуты и выключает ПК
Если время больше 8 часов, то будет равно 8 часов.
/?shutdown=false - отменяет отключение
При повторной установке таймера будет проивзедено отключение текущего, и установка нового.

// Собрать приложение, чтобы не было окна консоли.
go build -ldflags "-H=windowsgui" -o timeAllow.exe
*/
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const secIn8hours = 8 * 60 * 60

func setShutdown(timeMin int) error {
	timeSec := min(timeMin*60, secIn8hours)
	cmd := exec.Command("shutdown", "/s", "/t", strconv.Itoa(timeSec))
	return cmd.Run()
}
func canselShutDown() error {
	cmd := exec.Command("shutdown", "/a")
	return cmd.Run()
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		params := r.URL.Query()
		shutdownStr := params.Get("shutdown")
		var timeSec = 60
		if shutdownStr != "" {
			// Если передано значение "false", отменяем выключение
			if strings.EqualFold(shutdownStr, "false") {
				//Выполняем команду отмены shutdown
				err := canselShutDown()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintln(w, "Ошибка при отмене shutdown:", err.Error())
					return
				}
				fmt.Fprintln(w, "Выключение отменено.")
				return
			} else {
				timeMin, err := strconv.Atoi(shutdownStr)
				if err != nil || timeSec < 0 {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Некорректное значение времени:", shutdownStr)
					return
				}
				err = setShutdown(timeMin)
				if err != nil {
					err := canselShutDown()
					if err == nil {
						err = setShutdown(timeMin)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							fmt.Fprintln(w, "Ошибка при установке таймера shutdown:", err.Error())
						}
					} else {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintln(w, "Ошибка при установке таймера shutdown:", err.Error())
						return
					}
				} else {
					fmt.Fprintln(w, "Время отключения установлено:", timeMin)
				}
			}
		}
	}
}

func main() {
	// Создаем контекст для остановки сервера
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Регистрируем обработчик для маршрута /
		http.HandleFunc("/", httpHandler)
		// запускаем сервер на порту 8080
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	select {
	case <-signalChan:
		srv := &http.Server{Addr: ":8080"}
		if err := srv.Shutdown(ctx); err != nil {
			// ошибка остановки сервера
			os.Exit(1)
		}
		// остановлен без ошибки
		os.Exit(0)
	}
}
