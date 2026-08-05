package core

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFirmarOAuth1VectorRFC5849 valida la firma HMAC-SHA1 contra el ejemplo
// oficial del RFC 5849 (secciones 3.4.1 y 3.5.1). Si esta prueba falla, la
// firma no es interoperable con ninguna API real de OAuth 1.0a.
func TestFirmarOAuth1VectorRFC5849(t *testing.T) {
	apiKey := "9djdj82h48djs9d2"
	apiSecret := "j49sk3j29djd"
	accessToken := "kkk9d7dh3k39sjv7"
	accessSecret := "dh893hdasih9"
	// timestamp y nonce fijos del ejemplo del RFC.
	const timestamp = int64(137131201)
	const nonce = "7d8f3e4a"

	// El ejemplo del RFC incluye parámetros de petición form-urlencoded en la
	// firma (b5=%3D%253D, a3=a, c%40=, a2=r b, c2=, a3=2 q).
	extra := []oauthParam{
		{"b5", "=%3D"},
		{"a3", "a"},
		{"c@", ""},
		{"a2", "r b"},
		{"c2", ""},
		{"a3", "2 q"},
	}

	auth := firmarOAuth1("POST", "http://example.com/request", apiKey, apiSecret, accessToken, accessSecret, timestamp, nonce, false, extra...)

	esperado := `OAuth oauth_consumer_key="9djdj82h48djs9d2", oauth_nonce="7d8f3e4a", oauth_signature="r6%2FTJjbCOr97%2F%2BUU0NsvSne7s5g%3D", oauth_signature_method="HMAC-SHA1", oauth_timestamp="137131201", oauth_token="kkk9d7dh3k39sjv7"`

	if auth != esperado {
		t.Errorf("firma OAuth1 incorrecta.\nobtenida:  %s\nesperada:  %s", auth, esperado)
	}
}

func TestExtraerTextoPublicacion(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"publicá en X que estoy trabajando en Jarvis", "estoy trabajando en Jarvis"},
		{"publicá un tuit que diga hola mundo", "hola mundo"},
		{"publicá en linkedin diciendo demo lista", "demo lista"},
		{"publicá en X con el texto lanzamiento oficial", "lanzamiento oficial"},
		{"publicá en X lo siguiente resumen semanal", "resumen semanal"},
		{"publicá en X", ""},
		{"qué hora es", ""},
	}
	for _, c := range casos {
		got := extraerTextoPublicacion(c.entrada)
		if got != c.esperado {
			t.Errorf("extraerTextoPublicacion(%q) = %q, esperaba %q", c.entrada, got, c.esperado)
		}
	}
}

// TestPostearTweetXExitoso apunta el post a un servidor local y verifica que
// el cuerpo, el header de autorización y el manejo del 201 sean correctos.
func TestPostearTweetXExitoso(t *testing.T) {
	var recibidoAuth, recibidoBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método = %s, esperaba POST", r.Method)
		}
		if r.URL.Path != "/2/tweets" {
			t.Errorf("path = %s, esperaba /2/tweets", r.URL.Path)
		}
		recibidoAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		recibidoBody = string(b)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"data":{"id":"123"}}`)
	}))
	defer srv.Close()
	testXEndpoint = srv.URL + "/2/tweets"
	defer func() { testXEndpoint = "" }()

	h := &Hands{XApiKey: "k", XApiSecret: "s", XAccessToken: "t", XAccessSecret: "ts"}
	if err := h.postearTweetX("hola desde Jarvis"); err != nil {
		t.Fatalf("postearTweetX exitoso devolvió error: %v", err)
	}

	if !strings.HasPrefix(recibidoAuth, "OAuth ") {
		t.Errorf("Authorization no arranca con OAuth: %q", recibidoAuth)
	}
	for _, parte := range []string{"oauth_consumer_key", "oauth_signature", "oauth_signature_method=\"HMAC-SHA1\""} {
		if !strings.Contains(recibidoAuth, parte) {
			t.Errorf("Authorization no incluye %s: %q", parte, recibidoAuth)
		}
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(recibidoBody), &body); err != nil {
		t.Fatalf("cuerpo no es JSON: %v", err)
	}
	if body["text"] != "hola desde Jarvis" {
		t.Errorf("texto del tweet = %q", body["text"])
	}
}

func TestPostearTweetXErrorHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"title":"Unauthorized"}`)
	}))
	defer srv.Close()
	testXEndpoint = srv.URL
	defer func() { testXEndpoint = "" }()

	h := &Hands{XApiKey: "k", XApiSecret: "s", XAccessToken: "t", XAccessSecret: "ts"}
	err := h.postearTweetX("hola")
	if err == nil {
		t.Fatal("un 401 de X debería devolver error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("el error debería mencionar el código 401: %v", err)
	}
}

func TestPostearLinkedInExitoso(t *testing.T) {
	var recibidoAuth, recibidoBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método = %s, esperaba POST", r.Method)
		}
		recibidoAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		recibidoBody = string(b)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"post-1"}`)
	}))
	defer srv.Close()
	testLinkedInEndpoint = srv.URL + "/v2/ugcPosts"
	defer func() { testLinkedInEndpoint = "" }()

	h := &Hands{LinkedInToken: "tok123", LinkedInAuthor: "urn:li:person:42"}
	if err := h.postearLinkedIn("post profesional"); err != nil {
		t.Fatalf("postearLinkedIn exitoso devolvió error: %v", err)
	}

	if recibidoAuth != "Bearer tok123" {
		t.Errorf("Authorization = %q, esperaba Bearer tok123", recibidoAuth)
	}
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(recibidoBody), &body); err != nil {
		t.Fatalf("cuerpo no es JSON: %v", err)
	}
	if body["author"] != "urn:li:person:42" {
		t.Errorf("author = %v", body["author"])
	}
	if body["lifecycleState"] != "PUBLISHED" {
		t.Errorf("lifecycleState = %v", body["lifecycleState"])
	}
}

func TestManejarRedesSinCredenciales(t *testing.T) {
	h := &Hands{}
	if r := h.manejarRedes("publicá en x que hola"); !strings.Contains(r, "x_api_key") {
		t.Errorf("sin credenciales X debería explicar cómo configurarlas: %q", r)
	}
	if r := h.manejarRedes("publicá en linkedin un post sobre la demo"); !strings.Contains(r, "linkedin_token") {
		t.Errorf("sin credenciales LinkedIn debería explicar cómo configurarlas: %q", r)
	}
	if r := h.manejarRedes("hola jarvis"); !strings.Contains(r, "X (Twitter)") {
		t.Errorf("ayuda de redes incorrecta: %q", r)
	}
}

func TestManejarRedesConCredencialesYEndpoint(t *testing.T) {
	posted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posted = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	testXEndpoint = srv.URL
	defer func() { testXEndpoint = "" }()

	h := &Hands{XApiKey: "k", XApiSecret: "s", XAccessToken: "t", XAccessSecret: "ts"}
	r := h.manejarRedes("publicá en x que estoy probando")
	if !strings.Contains(r, "Publicado en X") {
		t.Errorf("respuesta = %q, esperaba confirmación de publicación", r)
	}
	if !posted {
		t.Error("con credenciales y endpoint debería haber llamado a la API")
	}
}
