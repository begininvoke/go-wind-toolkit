package configexporter

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNormalizeEtcdEndpoints(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "localhost:2379", want: []string{"localhost:2379"}},
		{in: "http://localhost:2379", want: []string{"localhost:2379"}},
		{in: "https://etcd.example.com:2379/", want: []string{"etcd.example.com:2379"}},
		{in: "http://n1:2379, n2:2379 ,http://n3:2379", want: []string{"n1:2379", "n2:2379", "n3:2379"}},
		{in: "", want: nil},
		{in: " , ,", want: nil},
	}
	for _, tt := range tests {
		got := normalizeEtcdEndpoints(tt.in)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("normalizeEtcdEndpoints(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

// genTestPEM 在测试内自签发一对 CA + 叶证书,返回三段 PEM。
func genTestPEM(t *testing.T) (caPEM, certPEM, keyPEM string) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		NotAfter:              time.Now().Add(time.Hour),
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, caKey.Public(), caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-leaf"},
		NotAfter:     time.Now().Add(time.Hour),
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, leafKey.Public(), caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		t.Fatal(err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

// buildTLSConfig 行为契约:全空→nil;CA+证书对→RootCAs 与 Certificates 均就绪;
// 垃圾 PEM / 单边证书对→错误。
func TestBuildTLSConfig(t *testing.T) {
	if cfg, err := buildTLSConfig("", "", ""); cfg != nil || err != nil {
		t.Fatalf("empty input: cfg=%v err=%v, want nil/nil", cfg, err)
	}

	ca, cert, key := genTestPEM(t)
	cfg, err := buildTLSConfig(ca, cert, key)
	if err != nil {
		t.Fatalf("valid material: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil || len(cfg.Certificates) != 1 {
		t.Fatalf("valid material: cfg=%v, want RootCAs + 1 client certificate", cfg)
	}

	if _, err = buildTLSConfig("not a pem", "", ""); err == nil {
		t.Fatal("garbage CA PEM must error")
	}
	if _, err = buildTLSConfig("", cert, ""); err == nil {
		t.Fatal("cert without key must error")
	}
	if _, err = buildTLSConfig("", "", key); err == nil {
		t.Fatal("key without cert must error")
	}
}

// Nacos 登录流程:带凭据→先登录并把 accessToken 附到发布请求;
// 无凭据→不登录、不带 token(与免认证服务端行为一致)。
func TestWriteNacosAuthFlow(t *testing.T) {
	var loginCalled bool
	var publishToken string
	var publishCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/nacos/v1/auth/login":
			loginCalled = true
			_ = r.ParseForm()
			if r.Form.Get("username") != "u" || r.Form.Get("password") != "p" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"accessToken": "tok-123"})
		case "/nacos/v1/cs/configs":
			publishCalls++
			_ = r.ParseForm()
			publishToken = r.Form.Get("accessToken")
			_, _ = w.Write([]byte("true"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ep := srv.Listener.Addr().String()

	rc := &RemoteConfig{
		Type:        Nacos,
		Endpoint:    ep,
		ProjectName: "p",
		Username:    "u",
		Password:    "p",
	}
	if err := writeNacos(rc, "app", "content"); err != nil {
		t.Fatalf("with credentials: %v", err)
	}
	if !loginCalled {
		t.Fatal("login endpoint must be called when credentials are set")
	}
	if publishToken != "tok-123" {
		t.Fatalf("publish accessToken = %q, want tok-123", publishToken)
	}

	loginCalled = false
	publishToken = "<unset>"
	rc2 := &RemoteConfig{Type: Nacos, Endpoint: ep, ProjectName: "p"}
	if err := writeNacos(rc2, "app", "content"); err != nil {
		t.Fatalf("without credentials: %v", err)
	}
	if loginCalled {
		t.Fatal("login must not be called without credentials")
	}
	if publishToken != "" {
		t.Fatalf("publish accessToken = %q, want empty", publishToken)
	}
}
