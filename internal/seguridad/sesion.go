package seguridad

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTokenInvalido = errors.New("token de sesion invalido")
	ErrTokenVencido  = errors.New("token de sesion vencido")
)

type contenido struct {
	Usuario string `json:"sub"`
	Expira  int64  `json:"exp"`
	Emitido int64  `json:"iat"`
}

// Gestor crea sesiones autocontenidas cuya integridad se protege con HMAC.
type Gestor struct {
	secreto  []byte
	duracion time.Duration
	reloj    func() time.Time
}

func NuevoGestor(secreto []byte, duracion time.Duration) (*Gestor, error) {
	if len(secreto) < 32 {
		return nil, errors.New("el secreto de sesion debe tener al menos 32 bytes")
	}
	if duracion <= 0 {
		return nil, errors.New("la duracion de sesion debe ser positiva")
	}
	copia := append([]byte(nil), secreto...)
	return &Gestor{secreto: copia, duracion: duracion, reloj: time.Now}, nil
}

func (g *Gestor) Crear(usuario string) (string, time.Time, error) {
	usuario = strings.TrimSpace(usuario)
	if usuario == "" {
		return "", time.Time{}, errors.New("usuario vacio")
	}
	ahora := g.reloj().UTC()
	expira := ahora.Add(g.duracion)
	payload, err := json.Marshal(contenido{Usuario: usuario, Expira: expira.Unix(), Emitido: ahora.Unix()})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("codificar sesion: %w", err)
	}
	parte := base64.RawURLEncoding.EncodeToString(payload)
	firma := g.firmar(parte)
	return parte + "." + base64.RawURLEncoding.EncodeToString(firma), expira, nil
}

func (g *Gestor) Validar(token string) (string, error) {
	partes := strings.Split(token, ".")
	if len(partes) != 2 || partes[0] == "" || partes[1] == "" {
		return "", ErrTokenInvalido
	}
	firma, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil || !hmac.Equal(firma, g.firmar(partes[0])) {
		return "", ErrTokenInvalido
	}
	payload, err := base64.RawURLEncoding.DecodeString(partes[0])
	if err != nil {
		return "", ErrTokenInvalido
	}
	var c contenido
	if err := json.Unmarshal(payload, &c); err != nil || strings.TrimSpace(c.Usuario) == "" || c.Expira <= 0 || c.Emitido <= 0 {
		return "", ErrTokenInvalido
	}
	ahora := g.reloj().UTC().Unix()
	if ahora >= c.Expira {
		return "", ErrTokenVencido
	}
	if c.Emitido > ahora+60 {
		return "", ErrTokenInvalido
	}
	return c.Usuario, nil
}

func (g *Gestor) firmar(parte string) []byte {
	mac := hmac.New(sha256.New, g.secreto)
	_, _ = mac.Write([]byte(parte))
	return mac.Sum(nil)
}

// CredencialesIguales evita comparaciones de contrasena con salida temprana.
func CredencialesIguales(recibida, esperada string) bool {
	a := sha256.Sum256([]byte(recibida))
	b := sha256.Sum256([]byte(esperada))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}
