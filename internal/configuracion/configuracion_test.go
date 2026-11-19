package configuracion

import "testing"

func TestCargarMinimo(t *testing.T) {
	t.Setenv("DATABASE_DSN", "usuario:clave@tcp(mysql:3306)/gestion_aridos")
	t.Setenv("APP_ADMIN_USER", "admin")
	t.Setenv("APP_ADMIN_PASSWORD", "una-clave-muy-segura")
	t.Setenv("APP_AUTH_SECRET", "01234567890123456789012345678901")
	c, err := Cargar()
	if err != nil {
		t.Fatal(err)
	}
	if c.Puerto != "8080" || c.DirectorioCliente != "ClientApp" || c.DuracionSesion.Hours() != 12 {
		t.Fatalf("valores predeterminados inesperados: %+v", c)
	}
}

func TestCargarRechazaSecretosDebiles(t *testing.T) {
	t.Setenv("DATABASE_DSN", "usuario:clave@tcp(mysql:3306)/gestion_aridos")
	t.Setenv("APP_ADMIN_USER", "admin")
	t.Setenv("APP_ADMIN_PASSWORD", "corta")
	t.Setenv("APP_AUTH_SECRET", "tambien-corto")
	if _, err := Cargar(); err == nil {
		t.Fatal("se esperaba error de validacion")
	}
}
