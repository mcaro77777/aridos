package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aridos/internal/basedatos"
	"aridos/internal/configuracion"
	"aridos/internal/httpapi"
	"aridos/internal/seguridad"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := ejecutar(logger); err != nil {
		logger.Error("la API termino con error", "error", err)
		os.Exit(1)
	}
}

func ejecutar(logger *slog.Logger) error {
	cfg, err := configuracion.Cargar()
	if err != nil {
		return fmt.Errorf("configuracion: %w", err)
	}
	ctxConexion, cancelarConexion := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelarConexion()
	db, err := basedatos.Abrir(ctxConexion, cfg.DSN, cfg.MaxConexiones, cfg.MaxConexionesOciosas, cfg.VidaConexion)
	if err != nil {
		return err
	}
	defer db.Close()

	sesiones, err := seguridad.NuevoGestor(cfg.SecretoAutenticacion, cfg.DuracionSesion)
	if err != nil {
		return fmt.Errorf("sesiones: %w", err)
	}
	api, err := httpapi.Nuevo(httpapi.Opciones{
		DB: db, Logger: logger, Sesiones: sesiones,
		UsuarioAdministrador: cfg.UsuarioAdministrador,
		ClaveAdministrador:   cfg.ClaveAdministrador,
		CookieSegura:         cfg.Produccion,
		OrigenPermitido:      cfg.OrigenPermitido,
		DirectorioCliente:    cfg.DirectorioCliente,
	})
	if err != nil {
		return err
	}

	servidor := &http.Server{
		Addr:              net.JoinHostPort("", cfg.Puerto),
		Handler:           api.Handler(),
		ReadTimeout:       cfg.TiempoLectura,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      cfg.TiempoEscritura,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	ctxSenal, detenerSenal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer detenerSenal()
	errores := make(chan error, 1)
	go func() {
		logger.Info("API de aridos iniciada", "direccion", servidor.Addr, "cliente", cfg.DirectorioCliente)
		errores <- servidor.ListenAndServe()
	}()

	select {
	case err := <-errores:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctxSenal.Done():
		logger.Info("senal de apagado recibida")
	}
	ctxApagado, cancelarApagado := context.WithTimeout(context.Background(), cfg.TiempoApagado)
	defer cancelarApagado()
	if err := servidor.Shutdown(ctxApagado); err != nil {
		_ = servidor.Close()
		return fmt.Errorf("apagado HTTP: %w", err)
	}
	logger.Info("API detenida correctamente")
	return nil
}
