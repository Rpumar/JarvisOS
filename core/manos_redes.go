package core

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// === REDES SOCIALES (F3): publicar en X (Twitter) y LinkedIn ===
// Sin librerías externas: se usa la firma OAuth 1.0a (HMAC-SHA1) de la
// librería estándar para la API v2 de X, y un Bearer token OAuth 2.0 para
// la API de LinkedIn. Publicar es una acción externa: el clasificador de
// seguridad la marca como RequiereAprobacion (PIN/panel) antes de salir.

// xTweetsEndpoint es la API v2 de X para publicar tweets.
const xTweetsEndpoint = "https://api.twitter.com/2/tweets"

// linkedinUgcEndpoint es la API de LinkedIn para publicar posts (UGC).
const linkedinUgcEndpoint = "https://api.linkedin.com/v2/ugcPosts"

// manejarRedes despacha los comandos de redes sociales hacia X o LinkedIn.
func (h *Hands) manejarRedes(cmd string) string {
	lower := strings.ToLower(cmd)

	if strings.Contains(lower, "linkedin") {
		return h.publicarLinkedIn(cmd)
	}

	if strings.Contains(lower, "x ") || strings.Contains(lower, " x") ||
		strings.Contains(lower, "tuit") || strings.Contains(lower, "twit") ||
		strings.Contains(lower, "twitter") {
		return h.publicarEnX(cmd)
	}

	return "Puedo publicar en X (Twitter) o LinkedIn, señor. Por ejemplo: 'publicá en X que estoy trabajando en Jarvis' o 'publicá en linkedin un post sobre la demo'."
}

// publicarEnX extrae el texto y lo publica como tweet vía la API v2 con
// OAuth 1.0a de usuario (consumer + access token).
func (h *Hands) publicarEnX(cmd string) string {
	if h.XApiKey == "" || h.XApiSecret == "" || h.XAccessToken == "" || h.XAccessSecret == "" {
		return "Para publicar en X necesito las claves de la app y los tokens en config.json (x_api_key, x_api_secret, x_access_token, x_access_secret), señor. Se generan en el portal de desarrolladores de X (developer.twitter.com)."
	}
	texto := extraerTextoPublicacion(cmd)
	if texto == "" {
		return "¿Qué texto querés publicar en X, señor? Por ejemplo: 'publicá en X que estoy trabajando en Jarvis'."
	}
	if err := h.postearTweetX(texto); err != nil {
		return fmt.Sprintf("No pude publicar en X, señor: %v", err)
	}
	return "Publicado en X, señor."
}

// publicarLinkedIn extrae el texto y publica un post UGC de LinkedIn.
func (h *Hands) publicarLinkedIn(cmd string) string {
	if h.LinkedInToken == "" || h.LinkedInAuthor == "" {
		return "Para publicar en LinkedIn necesito el token de acceso y el URN del autor en config.json (linkedin_token, linkedin_author), señor. Se obtienen creando una app en developer.linkedin.com y pidiendo un token OAuth 2.0."
	}
	texto := extraerTextoPublicacion(cmd)
	if texto == "" {
		return "¿Qué texto querés publicar en LinkedIn, señor? Por ejemplo: 'publicá en linkedin un post sobre la demo'."
	}
	if err := h.postearLinkedIn(texto); err != nil {
		return fmt.Sprintf("No pude publicar en LinkedIn, señor: %v", err)
	}
	return "Publicado en LinkedIn, señor."
}

// extraerTextoPublicacion toma el texto que sigue a los marcadores verbales
// de publicación. Devuelve "" si no encuentra un contenido razonable.
func extraerTextoPublicacion(cmd string) string {
	lower := strings.ToLower(cmd)

	marcadores := []string{
		"que diga ", "diga ", "que publique ", "diciendo ",
		"con el texto ", "el texto ", "lo siguiente ",
		"que ",
	}
	inicio := -1
	lenMarcador := 0
	for _, m := range marcadores {
		if i := strings.Index(lower, m); i >= 0 && (inicio < 0 || i < inicio) {
			inicio = i
			lenMarcador = len(m)
		}
	}
	if inicio < 0 {
		return ""
	}
	texto := strings.TrimSpace(cmd[inicio+lenMarcador:])
	return strings.Trim(texto, " .,;:")
}

// === FIRMA OAuth 1.0a (RFC 5849) ===

// oauthParam es un parámetro del header Authorization.
type oauthParam struct {
	clave string
	valor string
}

// firmarOAuth1 construye el header Authorization "OAuth ..." firmado con
// HMAC-SHA1 para una petición de la API v2 de X. Se usa la firma de
// "usuario": las claves de la app más el access token del dueño.
// extra admite parámetros de petición adicionales que forman parte de la
// firma (para las peticiones con body form-urlencoded; la API v2 de X usa
// JSON y no los incluye, así que se llama sin extras).
// conVersion controla si se incluye "oauth_version=1.0" (X lo exige; el
// vector canónico del RFC 5849 lo omite y se usa solo en los tests).
func firmarOAuth1(metodo, endpoint string, apiKey, apiSecret, accessToken, accessSecret string, timestamp int64, nonce string, conVersion bool, extra ...oauthParam) string {
	oauth := []oauthParam{
		{"oauth_consumer_key", apiKey},
		{"oauth_nonce", nonce},
		{"oauth_signature_method", "HMAC-SHA1"},
		{"oauth_timestamp", strconv.FormatInt(timestamp, 10)},
		{"oauth_token", accessToken},
	}
	if conVersion {
		oauth = append(oauth, oauthParam{"oauth_version", "1.0"})
	}

	// La firma combina los parámetros oauth_* con los de petición (si hay).
	firmaParams := append(append([]oauthParam{}, oauth...), extra...)
	sort.Slice(firmaParams, func(i, j int) bool {
		ci, cj := rfc3986(firmaParams[i].clave), rfc3986(firmaParams[j].clave)
		if ci != cj {
			return ci < cj
		}
		return rfc3986(firmaParams[i].valor) < rfc3986(firmaParams[j].valor)
	})
	var sb strings.Builder
	for i, p := range firmaParams {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(rfc3986(p.clave))
		sb.WriteString("=")
		sb.WriteString(rfc3986(p.valor))
	}

	base := strings.ToUpper(metodo) + "&" + rfc3986(endpoint) + "&" + rfc3986(sb.String())
	clave := rfc3986(apiSecret) + "&" + rfc3986(accessSecret)
	mac := hmac.New(sha1.New, []byte(clave))
	mac.Write([]byte(base))
	firma := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// El header Authorization lleva solo los oauth_* (más la firma).
	headerParams := append(append([]oauthParam{}, oauth...), oauthParam{"oauth_signature", firma})
	sort.Slice(headerParams, func(i, j int) bool { return headerParams[i].clave < headerParams[j].clave })
	partes := make([]string, 0, len(headerParams))
	for _, p := range headerParams {
		partes = append(partes, fmt.Sprintf(`%s="%s"`, rfc3986(p.clave), rfc3986(p.valor)))
	}
	return "OAuth " + strings.Join(partes, ", ")
}

// rfc3986 codifica un valor según RFC 3986 (solo deja -_.~ y alfanuméricos),
// como exige OAuth 1.0a.
func rfc3986(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0x0f])
	}
	return b.String()
}

// nuevoNonce genera un nonce aleatorio de 32 caracteres hex.
func nuevoNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// postearTweetX publica el texto en la API v2 de X.
func (h *Hands) postearTweetX(texto string) error {
	now := time.Now()
	auth := firmarOAuth1("POST", xEndpoint(), h.XApiKey, h.XApiSecret, h.XAccessToken, h.XAccessSecret, now.Unix(), nuevoNonce(), true)

	cuerpo, err := json.Marshal(map[string]string{"text": texto})
	if err != nil {
		return fmt.Errorf("no se pudo armar el tweet: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, xEndpoint(), strings.NewReader(string(cuerpo)))
	if err != nil {
		return fmt.Errorf("no se pudo armar la petición: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)

	cliente := &http.Client{Timeout: 20 * time.Second}
	resp, err := cliente.Do(req)
	if err != nil {
		return fmt.Errorf("error de red con la API de X: %w", err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("X respondió %d: %s", resp.StatusCode, strings.TrimSpace(string(datos)))
	}
	return nil
}

// postearLinkedIn publica un post UGC de LinkedIn con Bearer token.
func (h *Hands) postearLinkedIn(texto string) error {
	payload := map[string]interface{}{
		"author":           h.LinkedInAuthor,
		"lifecycleState":   "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": map[string]interface{}{
				"shareCommentary": map[string]interface{}{
					"text": texto,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}
	cuerpo, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("no se pudo armar el post: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, linkedinEndpoint(), strings.NewReader(string(cuerpo)))
	if err != nil {
		return fmt.Errorf("no se pudo armar la petición: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.LinkedInToken)

	cliente := &http.Client{Timeout: 20 * time.Second}
	resp, err := cliente.Do(req)
	if err != nil {
		return fmt.Errorf("error de red con la API de LinkedIn: %w", err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LinkedIn respondió %d: %s", resp.StatusCode, strings.TrimSpace(string(datos)))
	}
	return nil
}

// urlPorDefecto es la base para que los tests apunten a un servidor local
// (httptest) sin tocar la red. Las funciones de red usan variables por
// defecto que los tests pueden sobreescribir.
var testXEndpoint = ""
var testLinkedInEndpoint = ""

// uso las constantes por defecto salvo que el test defina otra.
func xEndpoint() string {
	if testXEndpoint != "" {
		return testXEndpoint
	}
	return xTweetsEndpoint
}

func linkedinEndpoint() string {
	if testLinkedInEndpoint != "" {
		return testLinkedInEndpoint
	}
	return linkedinUgcEndpoint
}
